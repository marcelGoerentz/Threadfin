package src

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
)

/*
NewStreamManager creates and returns a new StreamManager struct and will check permanently the errorChan of the struct
*/
func NewStreamManager() *StreamManager {
	sm := &StreamManager{
		playlists: map[string]*Playlist{},
		errorChan: make(chan ErrorInfo),
		stopChan:  make(chan bool),
	}

	// Start a go routine that will check for the error channel
	go func() {
		for {
			select {
			case errorInfo := <-sm.errorChan:
				if errorInfo.ErrorCode != 0 {
					playlistID, streamID := sm.GetPlaylistIDandStreamID(errorInfo.Stream)
					if errorInfo.ClientID != "" {
						// Client specifc errors
						sm.StopStream(playlistID, streamID, errorInfo.ClientID)
						/*} else {
						if streamID != "" {
							// Buffer errors
							if errorInfo.ErrorCode != EndOfFileError {
								ShowError(fmt.Errorf("stopping all clients because of error while buffering"), errorInfo.ErrorCode)
							} else {
								showInfo("Streaming:Stopping all clients as the stream has ended or was terminated!")
							}
							sm.StopStreamForAllClients(streamID)
						}*/
					} else {
						for clientID, client := range errorInfo.Stream.clients {
							sm.StopStream(playlistID, streamID, clientID)
							CloseClientConnection(client.w)
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
	_, exists := sm.playlists[playlistID]
	if !exists {
		// create a new one
		playlist := &Playlist{
			name:    getProviderParameter(playlistID, GetPlaylistType(playlistID), "name"),
			streams: make(map[string]*Stream),
		}
		// add the playlist to the map
		sm.playlists[playlistID] = playlist

		// check if a new stream is possible
		if IsNewStreamPossible(sm, streamInfo, w) {
			// create a new buffer and add the stream to the map within the new playlist
			sm.playlists[playlistID].streams[streamID] = CreateStream(streamInfo, sm.errorChan)
			if sm.playlists[playlistID].streams[streamID] == nil {
				return "", ""
			}
			showInfo(fmt.Sprintf("Streaming:Started streaming for %s", streamID))
		} else {
			return "", ""
		}
	} else {
		// check if the stream already exists
		stream, exists := sm.playlists[playlistID].streams[streamID]
		if !exists {
			// check if a new stream is possible
			if IsNewStreamPossible(sm, streamInfo, w) {
				// create a new buffer and add the stream to the map within the existing playlist
				sm.playlists[playlistID].streams[streamID] = CreateStream(streamInfo, sm.errorChan)
				showInfo(fmt.Sprintf("Streaming:Started streaming for %s", streamID))
			} else {
				return "", ""
			}
		} else {
			// Here we can check if multiple clients for one stream is allowed!
			showInfo(fmt.Sprintf("Streaming:Client joined %s, total: %d", streamID, len(stream.clients)+1))
		}
	}
	return
}

/*
CreateStream will create and return a new Stream struct, it will also start the new buffer.
*/
func CreateStream(streamInfo StreamInfo, errorChan chan ErrorInfo) *Stream {
	ctx, cancel := context.WithCancel(context.Background())
	folder := System.Folder.Temp + streamInfo.PlaylistID + string(os.PathSeparator) + streamInfo.URLid + string(os.PathSeparator)
	stream := &Stream{
		name:              streamInfo.Name,
		Buffer:            nil,
		ctx:               ctx,
		cancel:            cancel,
		URL:               streamInfo.URL,
		BackupChannel1URL: streamInfo.BackupChannel1URL,
		BackupChannel2URL: streamInfo.BackupChannel2URL,
		BackupChannel3URL: streamInfo.BackupChannel3URL,
		Folder:            folder,
		clients:           make(map[string]Client),
	}
	buffer := StartBuffer(stream, false, 0, errorChan)
	if buffer == nil {
		return nil
	}
	stream.Buffer = buffer
	return stream
}

/*
IsNewStreamPossible reports whether there is a new connection allowed
*/
func IsNewStreamPossible(sm *StreamManager, streamInfo StreamInfo, w http.ResponseWriter) bool {
	playlistID := streamInfo.PlaylistID
	if len(sm.playlists[playlistID].streams) < GetTuner(playlistID, GetPlaylistType(playlistID)) {
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
		}
		content, err = os.ReadFile(System.Folder.Video + fileList[0].Name())
		if err != nil {
			ShowError(err, 0)
		}
		contentOk = true
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
	showInfo("Streaming Status: No new connections available. Tuner limit reached.")
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
			bufferVFS.Remove(System.Folder.Video + file.Name())
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

	case "ffmpeg", "vlc":
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

	if len(cmd.Args) > 0  && path != "" {
		showInfo("Streaming Status:Creating video from uploaded image for a customized no more stream video")
		err := cmd.Run()
		if err != nil {
			return err
		}
		showInfo("Streaming Status:Successfully created video from custom image")
		return nil
	} else {
		return fmt.Errorf("path for third party tool ")
	}
}

// TODO: Add description
func prepareArguments(pathToFile string) (string, []string){
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

	playlist, exists := sm.playlists[playlistID]
	if exists {
		stream, exists := playlist.streams[streamID]
		if exists {
			//client := stream.clients[clientID]
			//client.w.Header().Set("Connection", "Close")
			//client.w.Write(nil)
			delete(stream.clients, clientID)
			showInfo(fmt.Sprintf("Streaming:Client left %s, total: %d", streamID, len(stream.clients)))
			if len(stream.clients) == 0 {
				stream.cancel() // Tell everyone about the ending of the stream
				if stream.Buffer.isThirdPartyBuffer {
					stream.Buffer.cmd.Process.Signal(syscall.SIGKILL) // Kill the third party tool process
					stream.Buffer.cmd.Wait()
					DeletPIDfromDisc(fmt.Sprintf("%d", stream.Buffer.cmd.Process.Pid)) // Delete the PID since the process has been terminated
				} else {
					close(stream.Buffer.stopChan)
				}
				delete(sm.playlists[playlistID].streams, streamID)
				showInfo(fmt.Sprintf("Streaming:Stopped streaming for %s", streamID))
				var debug = fmt.Sprintf("Streaming:Remove temporary files (%s)", stream.Folder)
				showDebug(debug, 1)

				debug = fmt.Sprintf("Streaming:Remove tmp folder %s", stream.Folder)
				showDebug(debug, 1)

				if err := bufferVFS.RemoveAll(stream.Folder); err != nil {
					ShowError(err, 4005)
				}
			}
		}
		if len(sm.playlists[playlistID].streams) == 0 {
			delete(sm.playlists, playlistID)
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

/*
StopStreamForAllClient stops the third paryt tool process and will delete all clients from the given stream
*/
/*func (sm *StreamManager) StopStreamForAllClients(streamID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	for playlistID, playlist := range sm.playlists {
		stream, exists := playlist.streams[streamID]
		if exists {
			stream.cancel() // Cancel the context to stop all clients
			stream.cmd.Process.Signal(syscall.SIGKILL)
			stream.cmd.Wait()
			DeletPIDfromDisc(fmt.Sprintf("%d", stream.cmd.Process.Pid))
			delete(playlist.streams, streamID)
			showInfo(fmt.Sprintf("Streaming:Stopped streaming for %s", streamID))
			var debug = fmt.Sprintf("Streaming:Remove temporary files (%s)", stream.Folder)
			showDebug(debug, 1)
			debug = fmt.Sprintf("Streaming:Remove tmp folder %s", stream.Folder)
			showDebug(debug, 1)
			if err := bufferVFS.RemoveAll(stream.Folder); err != nil {
				ShowError(err, 4005)
			}
		}
		if len(playlist.streams) == 0 {
			delete(sm.playlists, playlistID)
		}
	}
}

/*
ServeStream will ensure that the clients is getting the stream requested
*/
func (sm *StreamManager) ServeStream(streamInfo StreamInfo, w http.ResponseWriter, r *http.Request) {
	clientID, playlistID := sm.StartStream(streamInfo, w)
	if clientID == "" || playlistID == "" {
		if sm.playlists[streamInfo.PlaylistID].streams[streamInfo.URLid] == nil {
			delete(sm.playlists, streamInfo.PlaylistID)
		}
		return
	}
	defer sm.StopStream(playlistID, streamInfo.URLid, clientID)

	// Add a new client to the client map
	client := &Client{
		r: r,
		w: w,
	}
	stream := sm.playlists[playlistID].streams[streamInfo.URLid]
	stream.clients[clientID] = *client

	// If it was the first client start t
	if len(stream.clients) == 1 {
		// Send Data to the clients, this should run only once per stream
		go SendData(stream, sm.errorChan)
	}

	// Wait for the client context to get closed
	<-r.Context().Done()
}

/*
GetPlaylistIDandStreamID retrieves the playlist ID and the stream ID from the given stream
*/
func (sm *StreamManager) GetPlaylistIDandStreamID(stream *Stream) (string, string) {
	for playlistID, playlist := range sm.playlists {
		for streamID, tmpStream := range playlist.streams {
			if tmpStream.name == stream.name {
				return playlistID, streamID
			}
		}
	}
	showDebug("Streaming:Could not get playlist ID and stream ID", 3)
	return "", ""
}

/*
SendData sends Data to the clients connected to the stream
With errorChan it reports occuring errors to the StreamManager instance
*/
func SendData(stream *Stream, errorChan chan ErrorInfo) {
	var oldSegments []string

	for {
		tmpFiles := GetBufTmpFiles(stream)
		for _, f := range tmpFiles {
			if !CheckBufferFolder(stream) {
				errorChan <- ErrorInfo{BufferFolderError, stream, ""}
				return
			}
			oldSegments = append(oldSegments, f)
			showDebug(fmt.Sprintf("Streaming:Sending file %s to clients", f), 1)
			if !SendFileToClients(stream, f, errorChan) {
				return
			}
			if len(oldSegments) > 5 {
				DeleteOldestSegment(stream, oldSegments[0])
				oldSegments = oldSegments[1:]
			}
		}
		if len(tmpFiles) == 0 {
			time.Sleep(10 * time.Millisecond) // This will ensure that streams will synchronize over the time
		}
	}
}

/*
GetBufTmpFiles retrieves the files within the buffer folder
and returns a sorted list with the file names
*/
func GetBufTmpFiles(stream *Stream) (tmpFiles []string) {

	var tmpFolder = stream.Folder
	var fileIDs []float64

	if _, err := bufferVFS.Stat(tmpFolder); !fsIsNotExistErr(err) {

		files, err := bufferVFS.ReadDir(getPlatformPath(tmpFolder))
		if err != nil {
			ShowError(err, 000)
			return
		}

		// Check if more then one file is available
		if len(files) > 1 {
			// Iterate over the files and collect the IDs
			for _, file := range files {
				if !file.IsDir() && filepath.Ext(file.Name()) == ".ts" {
					fileID := strings.TrimSuffix(file.Name(), ".ts")
					if f, err := strconv.ParseFloat(fileID, 64); err == nil {
						fileIDs = append(fileIDs, f)
					}
				}
			}

			sort.Float64s(fileIDs)
			if len(fileIDs) > 0 {
				fileIDs = fileIDs[:len(fileIDs)-1]
			}

			// Create the return array with the sorted files
			for _, file := range fileIDs {
				fileName := fmt.Sprintf("%.0f.ts", file)
				// Check if the file is already within old segments array
				if !ContainsString(stream.OldSegments, fileName) {
					tmpFiles = append(tmpFiles, fileName)
					stream.OldSegments = append(stream.OldSegments, fileName)
				}
			}
		}
	}
	return
}

/*
DeleteOldesSegment will delete the file provided in the buffer
*/
func DeleteOldestSegment(stream *Stream, oldSegment string) {
	fileToRemove := stream.Folder + oldSegment
	if err := bufferVFS.RemoveAll(getPlatformFile(fileToRemove)); err != nil {
		ShowError(err, 4007)
	}
}

/*
CheckBufferFolder reports whether the buffer folder exists.
*/
func CheckBufferFolder(stream *Stream) bool {
	if _, err := bufferVFS.Stat(stream.Folder); fsIsNotExistErr(err) {
		return false
	}
	return true
}

/*
SendFileToClients reports whether sending the File to the clients was successful
It will also use the errorChan to report to the StreamManager if there is an error sending the file to a specifc client
*/
func SendFileToClients(stream *Stream, fileName string, errorChan chan ErrorInfo) bool {
	file, err := bufferVFS.Open(stream.Folder + fileName)
	if err != nil {
		ShowError(err, 4014)
		return false
	}
	defer file.Close()
	l, err := file.Stat()
	if err != nil {
		ShowError(err, 4015)
		return false
	}
	buffer := make([]byte, l.Size())
	if _, err := file.Read(buffer); err != nil {
		ShowError(err, 4016)
		return false
	}
	for clientID, client := range stream.clients {
		showDebug(fmt.Sprintf("Streaming:Sending file %s to client %s", fileName, clientID), 3)
		if _, err := client.w.Write(buffer); err != nil {
			showDebug(fmt.Sprintf("Streaming:Error when trying to send file to client %s", clientID), 1)
			errorChan <- ErrorInfo{SendFileError, stream, clientID}
		}
	}
	return true
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
	for playlistID, playlist := range sm.playlists {
		// create a new ActiveStreams struct if it doesn't exist right now
		if response.ActiveStreams == nil {
			response.ActiveStreams = &ActiveStreamsStruct{
				Playlists: make(map[string]*PlaylistStruct),
			}
		}
		// for every Playlist found Create a new Playlist struct and add it to the map
		response.ActiveStreams.Playlists[playlistID] = CreatePlaylistStruct(playlist.name, sm.playlists[playlistID].streams)
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
		*playlist.ActiveChannels = append(*playlist.ActiveChannels, stream.name)
		playlist.ClientConnections += len(stream.clients)
	}
	return playlist
}
