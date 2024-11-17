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
	return &StreamManager{
		playlists: map[string]*NewPlaylist{},
		streams:   make(map[string]*NewStream),
	}
}

// StartStream startet den ffmpeg-Prozess für einen Stream
func (sm *StreamManager) StartStream(streamInfo StreamInfo, w http.ResponseWriter) (string, string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	playlistID := streamInfo.PlaylistID
	streamID := streamInfo.URLid
	_, exists := sm.playlists[playlistID]
	if !exists {
		playlist := &NewPlaylist{
			streams: make(map[string]*NewStream),
		}
		sm.playlists[playlistID] = playlist
		sm.playlists[playlistID].streams[streamID] = createStream(streamInfo)
		showInfo(fmt.Sprintf("Streaming:Started streaming for %s", streamID))
	} else {
		stream, exists := sm.streams[streamID]
		if !exists {
			if isNewStreamPossible(sm, streamInfo, w) {
				sm.playlists[playlistID].streams[streamID] = createStream(streamInfo)
				showInfo(fmt.Sprintf("Streaming:Started streaming for %s", streamID))
			} else {
				return "", ""
			}
		} else {
			if isNewStreamPossible(sm, streamInfo, w) {
				showInfo(fmt.Sprintf("Streaming:Client joined %s, total: %d", streamID, len(stream.clients)+1))
			} else {
				return "", ""
			}
		}
	}
	return uuid.New().String(), playlistID
}

func createStream(streamInfo StreamInfo) *NewStream {
	ctx, cancel := context.WithCancel(context.Background())
	folder := System.Folder.Temp + streamInfo.PlaylistID + string(os.PathSeparator) + streamInfo.URLid + string(os.PathSeparator)
	stream := &NewStream{
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
	cmd := buffer(stream, false, 0)
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

func getStreamLimitContent() ([]byte, bool) {
	var content []byte
	var contentOk bool
	imageFileList, err := bufferVFS.ReadDir(System.Folder.Custom)
	if err != nil {
		ShowError(err, 0)
	}
	fileList, err := bufferVFS.ReadDir(System.Folder.Video)
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
	}
	if value, ok := webUI["html/video/stream-limit.ts"]; ok && !contentOk {
		contentOk = true
		content = GetHTMLString(value.(string))
	}
	return content, contentOk
}

func handleStreamLimit(w http.ResponseWriter) {
	showInfo("Streaming Status: No new connections available. Tuner limit reached.")
	content, contentOk := getStreamLimitContent()
	if contentOk {
		w.Header().Set("Content-type", "video/mpeg")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
		w.WriteHeader(http.StatusOK)
		for i := 0; i < 60; i++ {
			if _, err := w.Write(content); err != nil {
				ShowError(err, 0)
				return
			}
			time.Sleep(500 * time.Millisecond)
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
func (sm *StreamManager) StopStream(streamID string, clientID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	stream, exists := sm.streams[streamID]
	if exists {
		delete(stream.clients, clientID)
		showInfo(fmt.Sprintf("Streaming:Client left %s, total: %d", streamID, len(stream.clients)))
		if len(stream.clients) == 0 {
			stream.cmd.Process.Signal(syscall.SIGKILL)
			stream.cmd.Wait()
			deletPIDfromDisc(fmt.Sprintf("%d", stream.cmd.Process.Pid))
			delete(sm.streams, streamID)
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
}

// ServeStream sendet die Daten an die Clients
func (sm *StreamManager) ServeStream(streamInfo StreamInfo, w http.ResponseWriter, r *http.Request) {
	clientID, playlistID := sm.StartStream(streamInfo, w)
	defer sm.StopStream(streamInfo.URLid, clientID)
	if clientID != "" {
		client := &Client{
			r: r,
			w: w,
		}
		stream := sm.playlists[playlistID].streams[streamInfo.URLid]
		stream.clients[clientID] = *client
		// Ab hier die Daten an den Client senden
		if len(stream.clients) == 1 {
			serveStream(stream, r)
		} else {
			<-stream.ctx.Done()
			delete(stream.clients, clientID)
		}
	}

}

func serveStream(stream *NewStream, r *http.Request) {
	var oldSegments []string
	for {
		if ctx := r.Context(); ctx.Err() != nil {
			return
		}

		tmpFiles := getBufTmpFiles(stream)
		for _, f := range tmpFiles {
			if !checkBufferFolder(stream) {
				return
			}
			oldSegments = append(oldSegments, f)
			if !sendFileToClient(stream, f) {
				return
			}
			if len(oldSegments) > 5 {
				deleteOldestSegment(stream, oldSegments[0])
				oldSegments = oldSegments[1:]
			}
		}
		if len(tmpFiles) == 0 {
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func getBufTmpFiles(stream *NewStream) (tmpFiles []string) {

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

func deleteOldestSegment(stream *NewStream, oldSegment string) {
	fileToRemove := stream.Folder + oldSegment
	if err := bufferVFS.RemoveAll(getPlatformFile(fileToRemove)); err != nil {
		ShowError(err, 4007)
	}
}

func checkBufferFolder(stream *NewStream) bool {
	if _, err := bufferVFS.Stat(stream.Folder); fsIsNotExistErr(err) {
		return false
	}
	return true
}

func sendFileToClient(stream *NewStream, fileName string) bool {
	file, err := bufferVFS.Open(stream.Folder + fileName)
	if err != nil {
		showDebug(fmt.Sprintf("Buffer Open (%s)", fileName), 2)
		return false
	}
	defer file.Close()
	l, err := file.Stat()
	if err != nil {
		return false
	}
	buffer := make([]byte, l.Size())
	if _, err := file.Read(buffer); err != nil {
		return false
	}
	for _, client := range stream.clients {
		if _, err := client.w.Write(buffer); err != nil {
			return false
		}
	}
	return true
}

func getCurrentlyUsedChannels(response *APIResponseStruct) error {
	// should be nil but its always better to check
	if response.ActiveStreams == nil {
		response.ActiveStreams = &ActiveStreamsStruct{
			Playlists: make(map[string]*PlaylistStruct),
		}
	} else if response.ActiveStreams.Playlists == nil {
		response.ActiveStreams.Playlists = make(map[string]*PlaylistStruct)
	}
	BufferInformation.Range(func(_, value interface{}) bool {
		playlist, ok := value.(Playlist)
		if !ok {
			return true // Skip if the type assertion fails
		}

		var playlistID = playlist.PlaylistID
		// should be nil but its always better to check
		if response.ActiveStreams == nil {
			response.ActiveStreams = &ActiveStreamsStruct{
				Playlists: make(map[string]*PlaylistStruct),
			}
		} else if response.ActiveStreams.Playlists == nil {
			response.ActiveStreams.Playlists = make(map[string]*PlaylistStruct)
		}
		response.ActiveStreams.Playlists[playlistID] = createPlaylistStruct(playlist.PlaylistName, playlistID, playlist.Streams)
		return true
	})
	return nil
}

/*
This function will extract the info from the ThisStrem Struct
*/
func createPlaylistStruct(name string, playlistID string, streams map[int]ThisStream) *PlaylistStruct {
	var playlist = &PlaylistStruct{}
	playlist.PlaylistName = name
	playlist.ActiveChannels = &[]string{}
	playlist.ClientConnections = 0

	for _, stream := range streams {
		*playlist.ActiveChannels = append(*playlist.ActiveChannels, stream.ChannelName)
		if c, ok := BufferClients.Load(playlistID + stream.MD5); ok {
			var clients = c.(ClientConnection)
			playlist.ClientConnections += clients.Connection
		}
	}
	return playlist
}
