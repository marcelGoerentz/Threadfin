package src

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
)

func NewStreamManager() *StreamManager {
	sm := &StreamManager{
		playlists: map[string]*Playlist{},
		errorChan: make(chan ErrorInfo),
		stopChan: make(chan bool),
	}

	// Start a go routine that will check for the error channel
	go func() {
		for {
			select {
				case errorInfo := <- sm.errorChan:
					if errorInfo.ErrorCode != 0 {
						playlistID, streamID := sm.getPlaylistIDandStreamID(errorInfo.Stream)
						if errorInfo.ClientID != "" {
							// Client specifc errors
							sm.StopStream(playlistID, streamID, errorInfo.ClientID)
						} else {
							// Buffer errors
							if errorInfo.ErrorCode != EndOfFileError{
								ShowError(fmt.Errorf("stopping all clients because of error while buffering"), errorInfo.ErrorCode)
							}
							sm.stopStreamForAllClients(streamID)
						}
					}
				case <- sm.stopChan:
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
			name:    getProviderParameter(playlistID, getPlaylistType(playlistID), "name"),
			streams: make(map[string]*Stream),
		}
		// add the playlist to the map
		sm.playlists[playlistID] = playlist

		// check if a new stream is possible
		if isNewStreamPossible(sm, streamInfo, w) {
			// create a new buffer and add the stream to the map within the new playlist
			sm.playlists[playlistID].streams[streamID] = createStream(streamInfo, sm.errorChan)
			showInfo(fmt.Sprintf("Streaming:Started streaming for %s", streamID))
		} else {
			return "", ""
		}
	} else {
		// check if the stream already exists
		stream, exists := sm.playlists[playlistID].streams[streamID]
		if !exists {
			// check if a new stream is possible
			if isNewStreamPossible(sm, streamInfo, w) {
				// create a new buffer and add the stream to the map within the existing playlist
				sm.playlists[playlistID].streams[streamID] = createStream(streamInfo, sm.errorChan)
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

func createStream(streamInfo StreamInfo, errorChan chan ErrorInfo) *Stream {
	ctx, cancel := context.WithCancel(context.Background())
	folder := System.Folder.Temp + streamInfo.PlaylistID + string(os.PathSeparator) + streamInfo.URLid + string(os.PathSeparator)
	stream := &Stream{
		name:              streamInfo.Name,
		cmd:               nil,
		ctx:               ctx,
		cancel:            cancel,
		URL:               streamInfo.URL,
		BackupChannel1URL: streamInfo.BackupChannel1URL,
		BackupChannel2URL: streamInfo.BackupChannel2URL,
		BackupChannel3URL: streamInfo.BackupChannel3URL,
		Folder:            folder,
		clients:           make(map[string]Client),
	}
	cmd := buffer(stream, false, 0, errorChan)
	stream.cmd = cmd
	return stream
}

func isNewStreamPossible(sm *StreamManager, streamInfo StreamInfo, w http.ResponseWriter) bool {
	playlistID := streamInfo.PlaylistID
	if len(sm.playlists[playlistID].streams) < getTuner(playlistID, getPlaylistType(playlistID)) {
		return true
	} else {
		handleStreamLimit(w)
		return false
	}
}

/* This function will check if there is already a custuom video that will be provided to client
 * Otherwise it will check if there has been uploaded a image that will be converted into an video
 * Finally it will provide either the default content or the new content
 */
func getStreamLimitContent() ([]byte, bool) {
	var content []byte
	var contentOk bool
	imageFileList, err := os.ReadDir(System.Folder.Custom)
	if err != nil {
		ShowError(err, 0)
	}
	fileList, err := os.ReadDir(System.Folder.Video)
	if err == nil {
		createContent := shouldCreateContent(fileList)
		if createContent && len(imageFileList) > 0 {
			err := createAlternativNoMoreStreamsVideo(System.Folder.Custom + imageFileList[0].Name())
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

// Sends an info to the client that the stream limit has been reached. The content that will provided to client will be fetched with getStreamLimitContent() function
func handleStreamLimit(w http.ResponseWriter) {
	showInfo("Streaming Status: No new connections available. Tuner limit reached.")
	content, contentOk := getStreamLimitContent()
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

func shouldCreateContent(fileList []fs.DirEntry) bool {
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

func getTuner(id, playlistType string) (tuner int) {

	switch Settings.Buffer {

	case "-":
		tuner = Settings.Tuner

	case "threadfin", "ffmpeg", "vlc":

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

func getPlaylistType(playlistID string) string {
	switch playlistID[0:1] {
	case "M":
		return "m3u"
	case "H":
		return "hdhr"
	default:
		return ""
	}
}

func createAlternativNoMoreStreamsVideo(pathToFile string) error {
	cmd := new(exec.Cmd)
	switch Settings.Buffer {
	case "ffmpeg":
		cmd = exec.Command(Settings.FFmpegPath, "-loop", "1", "-i", pathToFile, "-c:v", "libx264", "-t", "1", "-pix_fmt", "yuv420p", "-vf", "scale=1920:1080", fmt.Sprintf("%sstream-limit.ts", System.Folder.Video))
	case "vlc":
		cmd = exec.Command(Settings.VLCPath, "--no-audio", "--loop", "--sout", fmt.Sprintf("'#transcode{vcodec=h264,vb=1024,scale=1,width=1920,height=1080,acodec=none,venc=x264{preset=ultrafast}}:standard{access=file,mux=ts,dst=%sstream-limit.ts}'", System.Folder.Video), System.Folder.Video, pathToFile)

	}
	if len(cmd.Args) > 0 {
		showInfo("Streaming Status:Creating video from uploaded image for a customized no more stream video")
		err := cmd.Run()
		if err != nil {
			return err
		}
		showInfo("Streaming Status:Successfully created video from custom image")
	}
	return nil
}

// StopStream stoppt den ffmpeg-Prozess, wenn keine Clients mehr verbunden sind
func (sm *StreamManager) StopStream(playlistID string, streamID string, clientID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	playlist, exists := sm.playlists[playlistID]
	if exists{
		stream, exists := playlist.streams[streamID]
		if exists {
			delete(stream.clients, clientID)
			showInfo(fmt.Sprintf("Streaming:Client left %s, total: %d", streamID, len(stream.clients)))
			if len(stream.clients) == 0 {
				stream.cmd.Process.Signal(syscall.SIGKILL)
				stream.cmd.Wait()
				deletPIDfromDisc(fmt.Sprintf("%d", stream.cmd.Process.Pid))
				delete(sm.playlists[playlistID].streams, streamID)
				showInfo(fmt.Sprintf("Streaming:Stopped streaming for %s", streamID))
				var debug = fmt.Sprintf("Streaming Status:Remove temporary files (%s)", stream.Folder)
				showDebug(debug, 1)

				debug = fmt.Sprintf("Remove tmp folder:%s", stream.Folder)
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

/*
func closeClientConnection(w http.ResponseWriter) {
	var once sync.Once
    // Set the header
    w.Header().Set("Connection", "close")
    once.Do(func() {
        w.WriteHeader(http.StatusNotFound) // Einmaliger Aufruf von WriteHeader
    })
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
} */

func (sm *StreamManager) stopStreamForAllClients(streamID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	for playlistID, playlist := range sm.playlists {
		stream, exists := playlist.streams[streamID]
		if exists {
			stream.cancel() // Cancel the context to stop all clients
			stream.cmd.Process.Signal(syscall.SIGKILL)
			stream.cmd.Wait()
			deletPIDfromDisc(fmt.Sprintf("%d", stream.cmd.Process.Pid))
			delete(playlist.streams, streamID)
			showInfo(fmt.Sprintf("Streaming:Stopped streaming for %s", streamID))
			var debug = fmt.Sprintf("Streaming Status:Remove temporary files (%s)", stream.Folder)
			showDebug(debug, 1)
			debug = fmt.Sprintf("Remove tmp folder:%s", stream.Folder)
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

// ServeStream sendet die Daten an die Clients
func (sm *StreamManager) ServeStream(streamInfo StreamInfo, w http.ResponseWriter, r *http.Request) {
	clientID, playlistID := sm.StartStream(streamInfo, w)
	if clientID == "" || playlistID == "" {
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
		go serveStream(stream, sm.errorChan)
	}

	// Wait for the stream to close the context
	<-stream.ctx.Done()
}

/*
	This function retrieves the playlist ID and the stream ID from the given stream
*/
func (sm *StreamManager) getPlaylistIDandStreamID(stream *Stream) (string, string){
	for playlistID, playlist  := range sm.playlists {
		for streamID, tmpStream := range playlist.streams {
			if tmpStream.name == stream.name {
				return playlistID, streamID
			}
		}
	}
	showDebug("Could not get playlist ID and stream ID", 3)
	return "", ""
}

func serveStream(stream *Stream, errorChan chan ErrorInfo) {
	var oldSegments []string

	for {
		tmpFiles := getBufTmpFiles(stream)
		for _, f := range tmpFiles {
			if !checkBufferFolder(stream) {
				errorChan <- ErrorInfo{BufferFolderError, stream, ""}
				return
			}
			oldSegments = append(oldSegments, f)
			if !sendFileToClients(stream, f, errorChan) {
				return
			}
			if len(oldSegments) > 10 {
				deleteOldestSegment(stream, oldSegments[0])
				oldSegments = oldSegments[1:]
			}
		}
		if len(tmpFiles) == 0 {
			time.Sleep(10 * time.Millisecond) // This will ensure that streams will synchronize over the time
		}
	}
}

/*
	This function retrieves the files within the buffer folder
	and returns a sorted list with the file names
*/
func getBufTmpFiles(stream *Stream) (tmpFiles []string) {

	var tmpFolder = stream.Folder
	var fileIDs []float64

	if _, err := bufferVFS.Stat(tmpFolder); !fsIsNotExistErr(err) {

		files, err := bufferVFS.ReadDir(getPlatformPath(tmpFolder))
		if err != nil {
			ShowError(err, 000)
			return
		}

		if len(files) > 2 {

			for _, file := range files {

				var fileID = strings.Replace(file.Name(), ".ts", "", -1)
				var f, err = strconv.ParseFloat(fileID, 64)

				if err == nil {
					fileIDs = append(fileIDs, f)
				}

			}

			sort.Float64s(fileIDs)
			fileIDs = fileIDs[:len(fileIDs)-1]

			for _, file := range fileIDs {

				var fileName = fmt.Sprintf("%d.ts", int64(file))

				if indexOfString(fileName, stream.OldSegments) == -1 {
					tmpFiles = append(tmpFiles, fileName)
					stream.OldSegments = append(stream.OldSegments, fileName)
				}

			}

		}

	}

	return
}

func deleteOldestSegment(stream *Stream, oldSegment string) {
	fileToRemove := stream.Folder + oldSegment
	if err := bufferVFS.RemoveAll(getPlatformFile(fileToRemove)); err != nil {
		ShowError(err, 4007)
	}
}

/*
	Check if the buffer folder exists
*/
func checkBufferFolder(stream *Stream) bool {
	if _, err := bufferVFS.Stat(stream.Folder); fsIsNotExistErr(err) {
		return false
	}
	return true
}


/*
	This functions sends the buffered files to the clients
*/
func sendFileToClients(stream *Stream, fileName string, errorChan chan ErrorInfo) bool {
	file, err := bufferVFS.Open(stream.Folder + fileName)
	if err != nil {
		showInfo("DEBUG: Could not open file!")
		ShowError(err, 0) // TODO: Add error code!
		return false
	}
	defer file.Close()
	l, err := file.Stat()
	if err != nil {
		showInfo("DEBUG: Could not get file statisitcs!")
		ShowError(err, 0) // TODO: Add error code!
		return false
	}
	buffer := make([]byte, l.Size())
	if _, err := file.Read(buffer); err != nil {
		showInfo("DEBUG: Read Buffer is not working!")
		ShowError(err, 0) // TODO: Add error code!
		return false
	}
    for clientID, client := range stream.clients {
		showDebug(fmt.Sprintf("Sending file to client %s", fileName), 3)
		if _, err := client.w.Write(buffer); err != nil {
			errorChan <- ErrorInfo{SendFileError, stream, clientID}
        }
    }
	return true
}

/*
	This function is getting used by the API call for retrieving the currently used channels
*/
func getCurrentlyUsedChannels(sm *StreamManager, response *APIResponseStruct) error {
	// should be nil but its always better to check
	if response.ActiveStreams == nil {
		response.ActiveStreams = &ActiveStreamsStruct{
			Playlists: make(map[string]*PlaylistStruct),
		}
	} else if response.ActiveStreams.Playlists == nil {
		response.ActiveStreams.Playlists = make(map[string]*PlaylistStruct)
	}
	for playlistID, playlist := range sm.playlists {
		if response.ActiveStreams == nil {
			response.ActiveStreams = &ActiveStreamsStruct{
				Playlists: make(map[string]*PlaylistStruct),
			}
		}
		response.ActiveStreams.Playlists[playlistID] = createPlaylistStruct(playlist.name, sm.playlists[playlistID].streams)
	}
	return nil
}

/*
This function will extract the info from the ThisStrem Struct
*/
func createPlaylistStruct(name string, streams map[string]*Stream) *PlaylistStruct {
	var playlist = &PlaylistStruct{}
	playlist.PlaylistName = name
	playlist.ActiveChannels = &[]string{}
	playlist.ClientConnections = 0

	for _, stream := range streams {
		*playlist.ActiveChannels = append(*playlist.ActiveChannels, stream.name)
		playlist.ClientConnections += len(stream.clients)
	}
	return playlist
}
