package src

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/avfs/avfs"
)

// Stream repräsentiert einen einzelnen Stream
type Stream struct {
	mu        sync.Mutex
	Name      string
	Clients   map[string]*Client
	Buffer    BufferInterface
	ErrorChan chan ErrorInfo
	Ctx       context.Context
	Cancel    context.CancelFunc

	Folder            string
	URL               string
	BackupChannel1URL string
	BackupChannel2URL string
	BackupChannel3URL string
	UseBackup         bool
	BackupNumber      int
	DoAutoReconnect   bool

	StopTimer *time.Timer
	TimerCancel context.CancelFunc
}

type Client struct {
	w http.ResponseWriter
	r *http.Request
	buffer *bytes.Buffer
	flushChannel chan struct{}
	doneChannel chan struct{}
}

type ErrorInfo struct {
	Error        error
	ErrorCode    int
	Stream       *Stream
	ClientID     string
	BufferClosed bool
}

/*
CreateStream will create and return a new Stream struct, it will also start the new buffer.
*/
func CreateStream(streamInfo *StreamInfo, fileSystem avfs.VFS, errorChan chan ErrorInfo) *Stream {
	ctx, cancel := context.WithCancel(context.Background())
	folder := System.Folder.Temp + streamInfo.PlaylistID + string(os.PathSeparator) + streamInfo.URLid
	pipeReader, pipeWriter := io.Pipe()
	streamBuffer := StreamBuffer{
		FileSystem: fileSystem,
		PipeWriter: pipeWriter,
		PipeReader: pipeReader,
		StopChan: make(chan struct{}),
		CloseChan: make(chan struct{}),
	}
	var buffer BufferInterface
	switch Settings.Buffer {
	case "vlc", "ffmpeg":
		buffer = &ThirdPartyBuffer{
			StreamBuffer: streamBuffer,
		}
	case "threadfin":
		buffer = &ThreadfinBuffer{
			StreamBuffer: streamBuffer,
		}
	default:
		cancel()
		return nil		
	}
	
	stream := &Stream{
		Name:              streamInfo.Name,
		Buffer:            buffer,
		ErrorChan:         errorChan,
		Ctx:               ctx,
		Cancel:            cancel,
		URL:               streamInfo.URL,
		BackupChannel1URL: streamInfo.BackupChannel1URL,
		BackupChannel2URL: streamInfo.BackupChannel2URL,
		BackupChannel3URL: streamInfo.BackupChannel3URL,
		Folder:            folder,
		Clients:           make(map[string]*Client),
		BackupNumber:      0,
		UseBackup:         false,
		DoAutoReconnect:   Settings.BufferAutoReconnect,
	}
	if err := buffer.StartBuffer(stream); err != nil {
		return nil
	}
	go buffer.addBufferedFilesToPipe()
	return stream
}

func CreateTunerLimitReachedStream() *Stream {
	ctx, cancel := context.WithCancel(context.Background())
	// Create a minimal Stream object
	pipeReader, pipeWriter := io.Pipe()
	streamBuffer := &StreamBuffer{
		PipeWriter: pipeWriter,
		PipeReader: pipeReader,
		StopChan: make(chan struct{}),
		CloseChan: make(chan struct{}),
	}
	var buffer BufferInterface = streamBuffer
	stream := &Stream{
		Clients: make(map[string]*Client),
		Buffer: buffer,
		Name: "Tuner limit reached",
		Ctx: ctx,
		Cancel: cancel,
		DoAutoReconnect: false,
		Folder: "",
	}
	streamBuffer.Stream = stream

	return stream
}

func CloseClientConnection(w http.ResponseWriter) {
	// Set the header
	w.Header().Set("Connection", "close")
	// Close the connection explicitly
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
	if hijacker, ok := w.(http.Hijacker); ok {
		conn, _, err := hijacker.Hijack()
		if err == nil {
			conn.Close()
		}
	}
}

func (s *Stream) Broadcast() {
    buffer := make([]byte, 4096)
	var stopChan chan struct{} = s.Buffer.GetStopChan()
	var pipeReader *io.PipeReader = s.Buffer.GetPipeReader()
    for {
		select {
		case <- stopChan:
			// Pipe was closed stop reading from it!
			return
		default:
			n, err := pipeReader.Read(buffer)
			if err != nil {
				if err != io.EOF {
					fmt.Printf("Streaming:Error when reading from pipe: %v\n", err)
				}
				break
			}
			
			s.mu.Lock()
			for clientID, client := range s.Clients {
				client.buffer.Write(buffer[:n])
				select {
                case client.flushChannel <- struct{}{}:
					<-client.doneChannel
                default:
					// Skip sending if the channel is full
					ShowDebug(fmt.Sprintf("Skipped sending data to client: %s", clientID), 3)
                }
			}
			s.mu.Unlock()
		}
    }
}

func (s *Stream) handleClientWrites(client *Client) {
    for {
        select {
        case <-client.flushChannel:
            client.buffer.WriteTo(client.w)
            if flusher, ok := client.w.(http.Flusher); ok {
                flusher.Flush()
            }
			client.doneChannel <- struct{}{}
        case <-client.r.Context().Done():
            return
        }
    }
}

func (s *Stream) ReportError(err error, errCode int, clientID string, closed bool) {
	s.ErrorChan <- ErrorInfo{err, errCode, s, clientID, closed}
}

func (s *Stream) StopStream(streamID string) {
	var pipeWriter *io.PipeWriter
	switch buffer := s.Buffer.(type) {
	case *ThirdPartyBuffer:
		pipeWriter = buffer.PipeWriter
	}
	pipeWriter.Close()
	for clientID, client := range s.Clients {
		CloseClientConnection(client.w)
		delete(s.Clients, clientID)
		ShowInfo(fmt.Sprintf("Streaming:Client kicked %s, total: %d", streamID, len(s.Clients)))
		if len(s.Clients) == 0 {
			s.Cancel() // Tell everyone about the ending of the stream
			s.Buffer.CloseBuffer()
		}
	}
}

func (s *Stream) RemoveClientFromStream(streamID, clientID string) {
	s.Buffer.(*StreamBuffer).PipeWriter.Close()
	if client, exists := s.Clients[clientID]; exists {
		CloseClientConnection(client.w)
		delete(s.Clients, clientID)
		ShowInfo(fmt.Sprintf("Streaming:Removed client from %s, total: %d", streamID, len(s.Clients)))
	}
}

/*
UpdateStreamURLForBackup will set the ther stream url when a backup will be used
*/
func (s *Stream) UpdateStreamURLForBackup() {
	switch s.BackupNumber {
	case 1:
		s.URL = s.BackupChannel1URL
		ShowHighlight("START OF BACKUP 1 STREAM")
		ShowInfo("Backup Channel 1 URL: " + s.URL)
	case 2:
		s.URL = s.BackupChannel2URL
		ShowHighlight("START OF BACKUP 2 STREAM")
		ShowInfo("Backup Channel 2 URL: " + s.URL)
	case 3:
		s.URL = s.BackupChannel3URL
		ShowHighlight("START OF BACKUP 3 STREAM")
		ShowInfo("Backup Channel 3 URL: " + s.URL)
	}
}

/*
HandleBufferError will retry running the Buffer function with the next backup number
*/
func (s *Stream) handleBufferError(err error) {
	ShowError(err, 4011)
	if !s.UseBackup || (s.UseBackup && s.BackupNumber >= 0 && s.BackupNumber <= 3) {
		s.BackupNumber++
		if s.BackupChannel1URL != "" || s.BackupChannel2URL != "" || s.BackupChannel3URL != "" {
			s.UseBackup = true
			s.UpdateStreamURLForBackup()
			s.Buffer.StartBuffer(s)
		}
	}
}
