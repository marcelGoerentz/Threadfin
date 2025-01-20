package src

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/avfs/avfs/vfs/osfs"
	"github.com/google/uuid"
)

/*
NewStreamManager creates and returns a new StreamManager struct and will check permanently the errorChan of the struct
*/
func NewStreamManager() *StreamManager {
	sm := &StreamManager{
		Playlists:  map[string]*Playlist{},
		errorChan:  make(chan ErrorInfo),
		stopChan:   make(chan bool),
		FileSystem: nil,
	}

	// Start a go routine that will check for the error channel
	go func() {
		for {
			select {
			case errorInfo := <-sm.errorChan:
				stream := errorInfo.Stream
				if errorInfo.BufferClosed && !stream.DoAutoReconnect {
					ShowError(errorInfo.Error, errorInfo.ErrorCode)
				} else {
					ShowDebug(errorInfo.Error.Error(), 3)
				}
				_, streamID := sm.GetPlaylistIDandStreamID(stream)
				if errorInfo.ClientID != "" {
					// Client specifc errors
					errorInfo.Stream.RemoveClientFromStream(streamID, errorInfo.ClientID)
				} else {
					// Buffer disconnect error
					clients := stream.Clients
					if len(clients) > 0 && errorInfo.BufferClosed {
						if stream.DoAutoReconnect && stream.Buffer.IsThirdPartyBuffer {
							stream.Buffer.StartBuffer(stream)
						} else {
							stream.StopStream(streamID)
						}
					}
				}
			case <-sm.stopChan:
				return
			}
		}
	}()
	return sm
}

/*
StartStream starts the ffmpeg process for buffering a stream
It will check if the stream already exists
*/
func (sm *StreamManager) StartStream(streamInfo StreamInfo, w http.ResponseWriter) (clientID string, playlistID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// get the playlist ID from stream info
	playlistID = streamInfo.PlaylistID
	// generate new client ID
	clientID = uuid.New().String()
	// set URL ID as stream ID
	streamID := streamInfo.URLid

	// check if playlist already exists
	_, exists := sm.Playlists[playlistID]
	if !exists {
		// create a new one
		playlist := &Playlist{
			Name:    getProviderParameter(playlistID, GetPlaylistType(playlistID), "name"),
			Streams: make(map[string]*Stream),
		}
		// add the playlist to the map
		sm.Playlists[playlistID] = playlist

		// check if a new stream is possible
		if IsNewStreamPossible(sm, streamInfo, w) {
			// create a new buffer and add the stream to the map within the new playlist
			sm.Playlists[playlistID].Streams[streamID] = CreateStream(streamInfo, sm.FileSystem, sm.errorChan)
			if sm.Playlists[playlistID].Streams[streamID] == nil {
				return "", ""
			}
			ShowInfo(fmt.Sprintf("Streaming:Started streaming for %s", streamID))
		} else {
			return "", ""
		}
	} else {
		// check if the stream already exists
		stream, exists := sm.Playlists[playlistID].Streams[streamID]
		if !exists {
			// check if a new stream is possible
			if IsNewStreamPossible(sm, streamInfo, w) {
				// create a new buffer and add the stream to the map within the existing playlist
				sm.Playlists[playlistID].Streams[streamID] = CreateStream(streamInfo, sm.FileSystem, sm.errorChan)
				ShowInfo(fmt.Sprintf("Streaming:Started streaming for %s", streamID))
			} else {
				return "", ""
			}
		} else {
			// Here we can check if multiple clients for one stream is allowed!
			ShowInfo(fmt.Sprintf("Streaming:Client joined %s, total: %d", streamID, len(stream.Clients)+1))
		}
	}
	return
}

func InitBufferVFS(virtual bool) avfs.VFS {
	if virtual {
		return memfs.New()
	} else {
		return osfs.New()
	}
}

/*
CreateStream will create and return a new Stream struct, it will also start the new buffer.
*/
func CreateStream(streamInfo StreamInfo, fileSystem avfs.VFS, errorChan chan ErrorInfo) *Stream {
	ctx, cancel := context.WithCancel(context.Background())
	folder := System.Folder.Temp + streamInfo.PlaylistID + string(os.PathSeparator) + streamInfo.URLid
	stream := &Stream{
		Name:              streamInfo.Name,
		Buffer:            &StreamBuffer{Config: &BufferConfig{}, FileSystem: fileSystem},
		ErrorChan:         errorChan,
		Ctx:               ctx,
		Cancel:            cancel,
		URL:               streamInfo.URL,
		BackupChannel1URL: streamInfo.BackupChannel1URL,
		BackupChannel2URL: streamInfo.BackupChannel2URL,
		BackupChannel3URL: streamInfo.BackupChannel3URL,
		Folder:            folder,
		Clients:           make(map[string]Client),
		BackupNumber:      0,
		UseBackup:         false,
		DoAutoReconnect:   Settings.BufferAutoReconnect,
	}
	stream.Buffer.StartBuffer(stream)
	go stream.Buffer.writeToPipes()
	go stream.Buffer.findBufferedFiles()
	return stream
}

/*
IsNewStreamPossible reports whether there is a new connection allowed
*/
func IsNewStreamPossible(sm *StreamManager, streamInfo StreamInfo, w http.ResponseWriter) bool {
	playlistID := streamInfo.PlaylistID
	if len(sm.Playlists[playlistID].Streams) < GetTuner(playlistID, GetPlaylistType(playlistID)) {
		return true
	} else {
		HandleStreamLimit(w)
		return false
	}
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

/*
StopStream stops the third party tool process when there are no more clients receiving the stream
*/
func (sm *StreamManager) StopStream(playlistID string, streamID string, clientID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	playlist, exists := sm.Playlists[playlistID]
	if exists {
		if stream, exists := playlist.Streams[streamID]; exists {
			if client, exists := stream.Clients[clientID]; exists {
				CloseClientConnection(client.w)
				delete(stream.Clients, clientID)
				ShowInfo(fmt.Sprintf("Streaming:Client left %s, total: %d", streamID, len(stream.Clients)))
				if len(stream.Clients) == 0 {
					stream.Cancel() // Tell everyone about the ending of the stream
					if stream.Buffer.IsThirdPartyBuffer {
						stream.Buffer.Cmd.Process.Signal(syscall.SIGKILL) // Kill the third party tool process
						stream.Buffer.Cmd.Wait()
						DeletPIDfromDisc(fmt.Sprintf("%d", stream.Buffer.Cmd.Process.Pid)) // Delete the PID since the process has been terminated
					} else {
						close(stream.Buffer.StopChan)
					}

					ShowInfo(fmt.Sprintf("Streaming:Stopped streaming for %s", streamID))
					var debug = fmt.Sprintf("Streaming:Remove temporary files (%s)", stream.Folder)
					ShowDebug(debug, 1)

					debug = fmt.Sprintf("Streaming:Remove tmp folder %s", stream.Folder)
					ShowDebug(debug, 1)

					if err := sm.FileSystem.RemoveAll(stream.Folder); err != nil {
						ShowError(err, 4005)
					}
					delete(sm.Playlists[playlistID].Streams, streamID)
				}
			}
		}
		if len(sm.Playlists[playlistID].Streams) == 0 {
			delete(sm.Playlists, playlistID)
		}
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

func (sm *StreamManager) StopAllStreams() {
	for _, playlist := range sm.Playlists {
		for streamID, stream := range playlist.Streams {
			stream.StopStream(streamID)
		}
	}
}

/*
ServeStream will ensure that the clients is getting the stream requested
*/
func (sm *StreamManager) ServeStream(streamInfo StreamInfo, w http.ResponseWriter, r *http.Request) {

	if sm.LockAgainstNewStreams {
		return
	}

	// Initialize buffer file system
	if sm.FileSystem == nil {
		sm.FileSystem = InitBufferVFS(Settings.StoreBufferInRAM)
	}

	clientID, playlistID := sm.StartStream(streamInfo, w)
	if clientID == "" || playlistID == "" {
		if sm.Playlists[streamInfo.PlaylistID].Streams[streamInfo.URLid] == nil {
			delete(sm.Playlists, streamInfo.PlaylistID)
		}
		return
	}
	defer sm.StopStream(playlistID, streamInfo.URLid, clientID)

	pipeReader, pipeWriter := io.Pipe()
	// Add a new client to the client map
	client := &Client{
		r: r,
		w: w,
		pipeWriter: pipeWriter,
		pipeReader: pipeReader,
	}
	stream := sm.Playlists[playlistID].Streams[streamInfo.URLid]
	stream.Clients[clientID] = *client

	go stream.sendDataToClient(clientID, client)

	// Wait for the client context to get closed
	<-r.Context().Done()
}

/*
GetPlaylistIDandStreamID retrieves the playlist ID and the stream ID from the given stream
*/
func (sm *StreamManager) GetPlaylistIDandStreamID(stream *Stream) (string, string) {
	for playlistID, playlist := range sm.Playlists {
		for streamID, tmpStream := range playlist.Streams {
			if tmpStream.Name == stream.Name {
				return playlistID, streamID
			}
		}
	}
	ShowDebug("Streaming:Could not get playlist ID and stream ID", 3)
	return "", ""
}

func (s * Stream) sendDataToClient(clientID string, client *Client) {
	defer client.pipeReader.Close() // Ensure the pipe reader is closed when the function exits
	buffer := make([]byte, 4096)
	for {
		n, err := client.pipeReader.Read(buffer)
		if err != nil {
			if err != io.EOF {
				s.ReportError(fmt.Errorf("Streaming:Error when trying to send file to client %s", clientID), SendFileError, clientID, false)
			}
			break
		}
		if _, err := client.w.Write(buffer[:n]); err != nil {
			s.ReportError(fmt.Errorf("Streaming:Error when trying to send file to client %s", clientID), SendFileError, clientID, false)
			break
		}
	}
}

func (s *Stream) ReportError(err error, errCode int, clientID string, closed bool) {
	s.ErrorChan <- ErrorInfo{err, errCode, s, clientID, closed}
}

func (s *Stream) StopStream(streamID string) {
	for clientID, client := range s.Clients {
		client.pipeWriter.Close()
		CloseClientConnection(client.w)
		delete(s.Clients, clientID)
		ShowInfo(fmt.Sprintf("Streaming:Client kicked %s, total: %d", streamID, len(s.Clients)))
		if len(s.Clients) == 0 {
			s.Cancel() // Tell everyone about the ending of the stream
			if s.Buffer.IsThirdPartyBuffer {
				s.Buffer.Cmd.Process.Signal(syscall.SIGKILL) // Kill the third party tool process
				s.Buffer.Cmd.Wait()
				DeletPIDfromDisc(fmt.Sprintf("%d", s.Buffer.Cmd.Process.Pid)) // Delete the PID since the process has been terminated
			} else {
				close(s.Buffer.StopChan)
			}
		}
	}
}

func (s *Stream) RemoveClientFromStream(streamID, clientID string) {
	if client, exists := s.Clients[clientID]; exists {
		client.pipeWriter.Close()
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

/*
GetCurrentlyUsedChannels will extract and fill the data into the response struct about the currently active playlists and channels
*/
func GetCurrentlyUsedChannels(sm *StreamManager, response *APIResponseStruct) error {
	// should be nil but its always better to check
	if response.ActiveStreams == nil {
		response.ActiveStreams = &ActiveStreamsStruct{
			Playlists: make(map[string]*PlaylistStruct),
		}
	} else if response.ActiveStreams.Playlists == nil {
		response.ActiveStreams.Playlists = make(map[string]*PlaylistStruct)
	}
	// iterate over the playlists within the StreamManager and extract the data
	for playlistID, playlist := range sm.Playlists {
		// create a new ActiveStreams struct if it doesn't exist right now
		if response.ActiveStreams == nil {
			response.ActiveStreams = &ActiveStreamsStruct{
				Playlists: make(map[string]*PlaylistStruct),
			}
		}
		// for every Playlist found Create a new Playlist struct and add it to the map
		response.ActiveStreams.Playlists[playlistID] = CreatePlaylistStruct(playlist.Name, sm.Playlists[playlistID].Streams)
	}
	return nil
}

/*
CreatePlaylistSruct will extract the info from the given Stream struct
*/
func CreatePlaylistStruct(name string, streams map[string]*Stream) *PlaylistStruct {
	var playlist = &PlaylistStruct{
		PlaylistName:      name,
		ActiveChannels:    &[]string{},
		ClientConnections: 0,
	}

	// Iterate over every stream within the map
	for _, stream := range streams {
		*playlist.ActiveChannels = append(*playlist.ActiveChannels, stream.Name)
		playlist.ClientConnections += len(stream.Clients)
	}
	return playlist
}
