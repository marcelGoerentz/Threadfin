package src

import (
	"context"
	"net/http"
	"os/exec"
	"sync"

	"github.com/avfs/avfs"
)

// StreamManager verwaltet die Streams und ffmpeg-Prozesse
type StreamManager struct {
	Playlists map[string]*Playlist
	errorChan chan ErrorInfo
	stopChan  chan bool
	LockAgainstNewStreams bool
	FileSystem avfs.VFS
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
	ErrorChan chan ErrorInfo
	Ctx     context.Context
	Cancel  context.CancelFunc

	Folder            string
	LatestSegment	  int
	OldSegments       []string
	URL               string
	BackupChannel1URL string
	BackupChannel2URL string
	BackupChannel3URL string
	UseBackup         bool
	BackupNumber      int
	DoAutoReconnect   bool
}

type Buffer struct {
	FileSystem		   avfs.VFS
	IsThirdPartyBuffer bool
	Cmd                *exec.Cmd
	Config             *BufferConfig
	Stream			   *Stream // Reference to the parents struct
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
	Error     error
	ErrorCode int
	Stream    *Stream
	ClientID  string
	BufferClosed bool
}

const (
	BufferFolderError     = 4008
	SendFileError         = 4009
	CreateFileError       = 4010
	EndOfFileError        = 4011
	ReadIntoBufferError   = 4012
	WriteToBufferError    = 4013
	OpenFileError         = 4014  //errMsg = "Not able to open buffered file"
	FileStatError         = 4015	//errMsg = "Could not get file statics of buffered file"
	ReadFileError         = 4016  //errMsg = "Could not read buffered file before sending to clients"
	FileDoesNotExistError = 4019 //errMsg = "Buffered file does not exist anymore"
)
