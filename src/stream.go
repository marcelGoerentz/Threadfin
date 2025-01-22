package src

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"strconv"
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
func CreateStream(streamInfo StreamInfo, fileSystem avfs.VFS, errorChan chan ErrorInfo) *Stream {
	ctx, cancel := context.WithCancel(context.Background())
	folder := System.Folder.Temp + streamInfo.PlaylistID + string(os.PathSeparator) + streamInfo.URLid
	pipeReader, pipeWriter := io.Pipe()
	streamBuffer := StreamBuffer{
		FileSystem: fileSystem,
		PipeWriter: pipeWriter,
		PipeReader: pipeReader,
		StopChan: make(chan struct{}),
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

/*
GetStreamLimitContent will check if there is already a custuom video that will be provided to client.

Otherwise it will check if there has been uploaded a image that will be converted into an video.
Finally it will provide either the default content or the new content.
*/
func GetStreamLimitContent() ([]byte, bool) {
	var content []byte
	var contentOk bool
	imageFileList, err := os.ReadDir(System.Folder.Custom)
	if err != nil {
		ShowError(err, 0)
	}
	fileList, err := os.ReadDir(System.Folder.Video)
	if err == nil {
		createContent := ShouldCreateContent(fileList)
		if createContent && len(imageFileList) > 0 {
			err := CreateAlternativNoMoreStreamsVideo(System.Folder.Custom + imageFileList[0].Name())
			if err == nil {
				contentOk = true
			} else {
				ShowError(err, 0)
				return nil, false
			}
			content, err = os.ReadFile(System.Folder.Video + fileList[0].Name())
			if err != nil {
				ShowError(err, 0)
			}
			contentOk = true
		}
	}
	if !contentOk {
		if value, ok := webUI["html/video/stream-limit.ts"]; ok && !contentOk {
			contentOk = true
			content = GetHTMLString(value.(string))
		}
	}
	return content, contentOk
}

/*
HandleStreamLimit sends an info to the client that the stream limit has been reached.
The content that will provided to client will be fetched with GetStreamLimitContent() function
*/
func HandleStreamLimit(w http.ResponseWriter) {
	ShowInfo("Streaming Status: No new connections available. Tuner limit reached.")
	content, contentOk := GetStreamLimitContent()
	if contentOk {
		w.Header().Set("Content-type", "video/mpeg")
		w.WriteHeader(http.StatusOK)
		for i := 0; i < 600; i++ {
			if _, err := w.Write(content); err != nil {
				ShowError(err, 0)
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

/*
ShouldCreateContent reports whether a new video file shall be created.
It removes existing files if necessary.
*/
func ShouldCreateContent(fileList []fs.DirEntry) bool {
	switch len(fileList) {
	case 0:
		return true
	case 1:
		return false
	default:
		for _, file := range fileList {
			os.Remove(System.Folder.Video + file.Name())
		}
		return true
	}
}

/*
GetTuner returns the maximum number of connections for a playlist.
It will check if the buffer type is matching the third party buffers
*/
func GetTuner(id, playlistType string) (tuner int) {
	switch Settings.Buffer {
	case "-":
		tuner = Settings.Tuner

	case "ffmpeg", "vlc", "threadfin":
		i, err := strconv.Atoi(getProviderParameter(id, playlistType, "tuner"))
		if err == nil {
			tuner = i
		} else {
			ShowError(err, 0)
			tuner = 1
		}
	}
	return
}

/*
GetPlaylistType returns the type of the playlist based on the playlist ID
*/
func GetPlaylistType(playlistID string) string {
	switch playlistID[0:1] {
	case "M":
		return "m3u"
	case "H":
		return "hdhr"
	default:
		return ""
	}
}

/*
CreateAlternativNoMoreStreamsVideo generates a new video file based on the image provided as path to it.
It will use the third party tool defined in the settings and starts a process for generating the video file
*/
func CreateAlternativNoMoreStreamsVideo(pathToFile string) error {
	cmd := new(exec.Cmd)
	path, arguments := prepareArguments(pathToFile)
	if len(arguments) == 0 {
		if _, err := os.Stat(Settings.FFmpegPath); err != nil {
			return fmt.Errorf("ffmpeg path is not valid. Can not convert custom image to video")
		}
	}

	cmd = exec.Command(path, arguments...)

	if len(cmd.Args) > 0 && path != "" {
		ShowInfo("Streaming Status:Creating video from uploaded image for a customized no more stream video")
		err := cmd.Run()
		if err != nil {
			return err
		}
		ShowInfo("Streaming Status:Successfully created video from custom image")
		return nil
	} else {
		return fmt.Errorf("path for third party tool ")
	}
}

// TODO: Add description
func prepareArguments(pathToFile string) (string, []string) {
	switch Settings.Buffer {
	case "ffmpeg", "threadfin", "-":
		return Settings.FFmpegPath, []string{"--no-audio", "--loop", "--sout", fmt.Sprintf("'#transcode{vcodec=h264,vb=1024,scale=1,width=1920,height=1080,acodec=none,venc=x264{preset=ultrafast}}:standard{access=file,mux=ts,dst=%sstream-limit.ts}'", System.Folder.Video), System.Folder.Video, pathToFile}
	case "vlc":
		return Settings.VLCPath, []string{"-loop", "1", "-i", pathToFile, "-c:v", "libx264", "-t", "1", "-pix_fmt", "yuv420p", "-vf", "scale=1920:1080", fmt.Sprintf("%sstream-limit.ts", System.Folder.Video)}
	default:
		return "", []string{}
	}
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
