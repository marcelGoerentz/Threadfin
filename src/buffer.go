package src

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/avfs/avfs/vfs/osfs"
)

/*
InitBufferVFS will set the bufferVFS variable
*/
func InitBufferVFS(virtual bool) {

	if virtual {
		bufferVFS = memfs.New()
	} else {
		bufferVFS = osfs.New()
	}

}

func (b *Buffer) StartBuffer(stream *Stream, errorChan chan ErrorInfo) {
	if stream.UseBackup {
		UpdateStreamURLForBackup(stream)
	}

	var err error = nil
	if err = PrepareBufferFolder(stream.Folder); err != nil {
		ShowError(err, 4008)
		b.HandleBufferError(err, stream, errorChan)
		return
	}

	switch Settings.Buffer {
	case "ffmpeg", "vlc":
		err = StartThirdPartyBuffer(stream, errorChan)
	case "threadfin":
		err = StartThreadfinBuffer(stream, errorChan)
	default:
		return
	}
	if err != nil {
		b.HandleBufferError(err, stream, errorChan)
	}
}

/*
HandleBufferError will retry running the Buffer function with the next backup number
*/
func (b *Buffer) HandleBufferError(err error, stream *Stream, errorChan chan ErrorInfo) {
	ShowError(err, 4011)
	if !stream.UseBackup || (stream.UseBackup && stream.BackupNumber >= 0 && stream.BackupNumber <= 3) {
		stream.BackupNumber++
		if stream.BackupChannel1URL != "" || stream.BackupChannel2URL != "" || stream.BackupChannel3URL != "" {
			stream.UseBackup = true
			b.StartBuffer(stream, errorChan)
		}
	}
}

/*
HandleByteOutput save the byte ouptut of the command or http request as files
*/
func HandleByteOutput(stdOut io.ReadCloser, stream *Stream, errorChan chan ErrorInfo) {
	bufferSize := Settings.BufferSize * 1024 // Puffergröße in Bytes
	buffer := make([]byte, bufferSize)
	var fileSize int
	init := true
	tmpFolder := stream.Folder + string(os.PathSeparator)
	tmpSegment := 1

	var f avfs.File
	var err error
	var tmpFile string
	reader := bufio.NewReader(stdOut)
	for {
		if init {
			tmpFile = fmt.Sprintf("%s%d.ts", tmpFolder, tmpSegment)
			f, err = bufferVFS.Create(tmpFile)
			if err != nil {
				f.Close()
				ShowError(err, CreateFileError)
				errorChan <- ErrorInfo{CreateFileError, stream, ""}
				return
			}
			init = false
		}
		n, err := reader.Read(buffer)
		if n == 0 && err == nil {
			continue
		}
		if err == io.EOF {
			f.Close()
			ShowDebug("Buffer reached EOF!", 3)
			errorChan <- ErrorInfo{EndOfFileError, stream, ""}
			return
		}
		if err != nil {
			if _, ok := err.(*net.OpError); !ok || stream.Buffer.IsThirdPartyBuffer {
				ShowError(err, ReadIntoBufferError)
			}
			f.Close()
			errorChan <- ErrorInfo{ReadIntoBufferError, stream, ""}
			return
		}
		if _, err := f.Write(buffer[:n]); err != nil {
			ShowError(err, WriteToBufferError)
			f.Close()
			errorChan <- ErrorInfo{WriteToBufferError, stream, ""}
			return
		}
		fileSize += n
		// Prüfen, ob Dateigröße den Puffer überschreitet
		if fileSize >= bufferSize {
			tmpSegment++
			tmpFile = fmt.Sprintf("%s%d.ts", tmpFolder, tmpSegment)
			// Datei schließen und neue Datei öffnen
			f.Close()
			f, err = bufferVFS.Create(tmpFile)
			if err != nil {
				f.Close()
				ShowError(err, CreateFileError)
				errorChan <- ErrorInfo{CreateFileError, stream, ""}
				return
			}
			fileSize = 0
		}
	}
}

/*
UpdateStreamURLForBackup will set the ther stream url when a backup will be used
*/
func UpdateStreamURLForBackup(stream *Stream) {
	switch stream.BackupNumber {
	case 1:
		stream.URL = stream.BackupChannel1URL
		ShowHighlight("START OF BACKUP 1 STREAM")
		ShowInfo("Backup Channel 1 URL: " + stream.URL)
	case 2:
		stream.URL = stream.BackupChannel2URL
		ShowHighlight("START OF BACKUP 2 STREAM")
		ShowInfo("Backup Channel 2 URL: " + stream.URL)
	case 3:
		stream.URL = stream.BackupChannel3URL
		ShowHighlight("START OF BACKUP 3 STREAM")
		ShowInfo("Backup Channel 3 URL: " + stream.URL)
	}
}

/*
PrepareBufferFolder will clean the buffer folder and check if the folder exists
*/
func PrepareBufferFolder(folder string) error {
	if err := bufferVFS.RemoveAll(getPlatformPath(folder)); err != nil {
		return fmt.Errorf("failed to remove buffer folder: %w", err)
	}

	if err := checkVFSFolder(folder, bufferVFS); err != nil {
		return fmt.Errorf("failed to check buffer folder: %w", err)
	}

	return nil
}
