package src

import (
	"context"
	"net/http"
	"os/exec"
	"sync"
	"time"
)

// Playlist : Enthält allen Playlistinformationen, die der Buffer benötigr
type Playlist struct {
	Folder        string
	PlaylistID    string
	PlaylistName  string
	Tuner         int
	HttpProxyIP   string
	HttpProxyPort string

	Clients map[int]ThisClient
	Streams map[int]ThisStream
}

// ThisClient : Clientinfos
type ThisClient struct {
	Connection int
}

// ThisStream : Enthält Informationen zu dem abzuspielenden Stream einer Playlist
type ThisStream struct {
	StreamID		  int
	ClientCounter	  int
	ChannelName       string
	Error             string
	Folder            string
	MD5               string
	NetworkBandwidth  int
	PlaylistID        string
	PlaylistName      string
	Status            bool
	URL               string
	BackupChannel1URL string
	BackupChannel2URL string
	BackupChannel3URL string

	Segment []Segment
	// Lokale Temp Datein
	OldSegments []string
}

// Segment : URL Segmente (HLS / M3U8)
type Segment struct {
	Duration     float64
	Info         bool
	PlaylistType string
	Sequence     int64
	URL          string
	Version      int
	Wait         float64

	StreamInf struct {
		AverageBandwidth int
		Bandwidth        int
		Framerate        float64
		Resolution       string
		SegmentURL       string
	}
}

// DynamicStream : Streaminformationen bei dynamischer Bandbreite
type DynamicStream struct {
	AverageBandwidth int
	Bandwidth        int
	Framerate        float64
	Resolution       string
	URL              string
}

// ClientConnection : Client Verbindungen
type ClientConnection struct {
	Connection int
	Error      error
}

// BandwidthCalculation : Bandbreitenberechnung für den Stream
type BandwidthCalculation struct {
	NetworkBandwidth int
	Size             int
	Start            time.Time
	Stop             time.Time
	TimeDiff         float64
}


// StreamManager verwaltet die Streams und ffmpeg-Prozesse
type StreamManager struct {
	playlists map[string]*NewPlaylist
	errorChan chan int
    mu      sync.Mutex
}

type NewPlaylist struct {
	name string
	streams map[string]*NewStream
}

// Stream repräsentiert einen einzelnen Stream
type NewStream struct {
	name 	string
	clients map[string]Client
    cmd     *exec.Cmd
    ctx     context.Context
    cancel  context.CancelFunc

	Status bool
	Folder string
	OldSegments []string
	URL string
	BackupChannel1URL string
	BackupChannel2URL string
	BackupChannel3URL string
}

type Client struct{
	w http.ResponseWriter
	r *http.Request
}

const (
	NoError = 0
    BufferError = 1
    SendFileError   = 2
	CreateFileError = 3
	EndOfFileError = 4
	ReadIntoBufferError = 5
	WriteToBufferError = 6
)
