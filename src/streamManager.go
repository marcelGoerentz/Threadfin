package src

import (
	"bytes"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/avfs/avfs/vfs/osfs"
	"github.com/google/uuid"
)

// StreamManager verwaltet die Streams und ffmpeg-Prozesse
type StreamManager struct {
	Playlists             map[string]*Playlist
	errorChan             chan ErrorInfo
	stopChan              chan bool
	LockAgainstNewStreams bool
	FileSystem            avfs.VFS
	mu                    sync.Mutex
}

type Playlist struct {
	Name    string
	Streams map[string]*Stream
}

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
						if stream.DoAutoReconnect{
							if buffer, ok := stream.Buffer.(*ThirdPartyBuffer); ok {
								buffer.StartBuffer(stream)
								continue
							} 
						}
						stream.StopStream(streamID)
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
		if sm.IsNewStreamPossible(streamInfo, w) {
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
			if sm.IsNewStreamPossible(streamInfo, w) {
				// create a new buffer and add the stream to the map within the existing playlist
				sm.Playlists[playlistID].Streams[streamID] = CreateStream(streamInfo, sm.FileSystem, sm.errorChan)
				ShowInfo(fmt.Sprintf("Streaming:Started streaming for %s", streamID))
			} else {
				return "", ""
			}
		} else {
			if len(stream.Clients) == 0 {
				if stream.StopTimer != nil {
					stream.StopTimer.Stop()
					stream.StopTimer = nil
					stream.TimerCancel = nil
				}
				stream.Buffer.SetStopChan(make(chan struct{}))
				go stream.Buffer.addBufferedFilesToPipe()
			}
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
IsNewStreamPossible reports whether there is a new connection allowed
*/
func (sm *StreamManager) IsNewStreamPossible(streamInfo StreamInfo, w http.ResponseWriter) bool {
	playlistID := streamInfo.PlaylistID
	if len(sm.Playlists[playlistID].Streams) < GetTuner(playlistID, GetPlaylistType(playlistID)) {
		return true
	} else {
		HandleStreamLimit(w)
		return false
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
					stream.Buffer.StopBuffer()
					// Start a timer to stop the stream after a delay
                    cancel := func() {
						sm.mu.Lock()
						defer sm.mu.Unlock()
						stream.Cancel() // Tell everyone about the ending of the stream
						stream.Buffer.CloseBuffer()

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
					stream.TimerCancel = cancel
                    stream.StopTimer = time.AfterFunc(time.Duration(Settings.BufferTerminationTimeout) * time.Second, cancel)
				}
			}
		}
		if len(sm.Playlists[playlistID].Streams) == 0 {
			delete(sm.Playlists, playlistID)
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

	// Add a new client to the client map
	client := &Client{
		r: r,
		w: w,
		buffer: new(bytes.Buffer),
		flushChannel: make(chan struct{}, 1),
		doneChannel: make(chan struct{}, 1),
	}
	stream := sm.Playlists[playlistID].Streams[streamInfo.URLid]
	stream.Clients[clientID] = client

	// Start a goroutine to handle writing to the client
    go stream.handleClientWrites(client)

	// Make sure Broadcast is running only once
	if len(stream.Clients) == 1 {
		go stream.Broadcast()
	}

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
