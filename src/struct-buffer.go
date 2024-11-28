package src

import (
	"context"
	"net/http"
	"os/exec"
	"sync"
)

// StreamManager verwaltet die Streams und ffmpeg-Prozesse
type StreamManager struct {
	playlists map[string]*Playlist
	errorChan chan ErrorInfo
	stopChan  chan bool
	mu        sync.Mutex
}

type Playlist struct {
	name    string
	streams map[string]*Stream
}

// Stream repräsentiert einen einzelnen Stream
type Stream struct {
	name    string
	clients map[string]Client
	Buffer  *Buffer
	ctx     context.Context
	cancel  context.CancelFunc

	Folder            string
	OldSegments       []string
	URL               string
	BackupChannel1URL string
	BackupChannel2URL string
	BackupChannel3URL string
}

type Buffer struct {
	isThirdPartyBuffer bool
	cmd                *exec.Cmd
	stopChan           chan struct{}
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
