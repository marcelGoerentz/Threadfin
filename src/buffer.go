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

func StartBuffer(stream *Stream, useBackup bool, backupNumber int, errorChan chan ErrorInfo) *Buffer {
	if useBackup {
		UpdateStreamURLForBackup(stream, backupNumber)
	}

	if err := PrepareBufferFolder(stream.Folder); err != nil {
		ShowError(err, 4008)
		HandleBufferError(err, backupNumber, useBackup, stream, errorChan)
		return nil
	}

	switch Settings.Buffer {
	case "ffmpeg", "vlc":
		if buffer, err := StartThirdPartyBuffer(stream, useBackup, backupNumber, errorChan); err != nil {
			return HandleBufferError(err, backupNumber, useBackup, stream, errorChan)
		} else {
			return buffer
		}
	case "threadfin":
		if buffer, err := StartThreadfinBuffer(stream, useBackup, backupNumber, errorChan); err != nil {
			return HandleBufferError(err, backupNumber, useBackup, stream, errorChan)
		} else {
			return buffer
		}
	default:
		return nil
	}
}

/*
HandleBufferError will retry running the Buffer function with the next backup number
*/
func HandleBufferError(err error, backupNumber int, useBackup bool, stream *Stream, errorChan chan ErrorInfo) *Buffer {
	ShowError(err, 4011)
	if !useBackup || (useBackup && backupNumber >= 0 && backupNumber <= 3) {
		backupNumber++
		if stream.BackupChannel1URL != "" || stream.BackupChannel2URL != "" || stream.BackupChannel3URL != "" {
			return StartBuffer(stream, true, backupNumber, errorChan)
		}
	}
	return nil
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
				ShowError(err, 4010)
				errorChan <- ErrorInfo{CreateFileError, stream, ""}
				return
			}
			init = false
		}
		n, err := reader.Read(buffer)
		if n == 0 {
			continue
		}
		if err == io.EOF {
			f.Close()
			ShowDebug("Buffer reached EOF!", 3)
			errorChan <- ErrorInfo{EndOfFileError, stream, ""}
			return
		}
		if err != nil {
			if _, ok := err.(*net.OpError); !ok || stream.Buffer.isThirdPartyBuffer {
				ShowError(err, 4012)
			}
			f.Close()
			errorChan <- ErrorInfo{ReadIntoBufferError, stream, ""}
			return
		}
		if _, err := f.Write(buffer[:n]); err != nil {
			ShowError(err, 4013)
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
				ShowError(err, 4010)
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
func UpdateStreamURLForBackup(stream *Stream, backupNumber int) {
	switch backupNumber {
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
