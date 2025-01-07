package src

import (
	"context"
	"net/http"
	"os/exec"
	"sync"
)

// StreamManager verwaltet die Streams und ffmpeg-Prozesse
type StreamManager struct {
	Playlists map[string]*Playlist
	errorChan chan ErrorInfo
	stopChan  chan bool
	mu        sync.Mutex
}

type Playlist struct {
	Name    string
	Streams map[string]*Stream
}

// Stream repräsentiert einen einzelnen Stream
type Stream struct {
	Name    string
	Clients map[string]Client
	Buffer  *Buffer
	Ctx     context.Context
	Cancel  context.CancelFunc

	Folder            string
	OldSegments       []string
	URL               string
	BackupChannel1URL string
	BackupChannel2URL string
	BackupChannel3URL string
	UseBackup         bool
	BackupNumber      int
}

type Buffer struct {
	IsThirdPartyBuffer bool
	Cmd                *exec.Cmd
	Config             *BufferConfig
	StopChan           chan struct{}
}

type BufferConfig struct {
	BufferType string
	Path       string
	Options    string
}

type Client struct {
	w http.ResponseWriter
	r *http.Request
}

type ErrorInfo struct {
	ErrorCode int
	Stream    *Stream
	ClientID  string
}

const (
	NoError             = 0
	BufferFolderError   = 4008
	SendFileError       = 4009
	CreateFileError     = 4010
	EndOfFileError      = 4011
	ReadIntoBufferError = 4012
	WriteToBufferError  = 4013
)
