package src

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/avfs/avfs"
)

func (b *Buffer) StartBuffer(stream *Stream,) {
	b.Stream = stream
	var err error = nil
	if err = b.PrepareBufferFolder(stream.Folder); err != nil {
		// If something went wrong when setting up the buffer storage don't run at all
		stream.ReportError(err, BufferFolderError, "", true)
		return
	}

	switch Settings.Buffer {
	case "ffmpeg", "vlc":
		err = StartThirdPartyBuffer(stream)
	case "threadfin":
		err = StartThreadfinBuffer(stream)
	default:
		return
	}
	if err != nil {
		b.Stream.handleBufferError(err)
	}
}

/*
HandleByteOutput save the byte ouptut of the command or http request as files
*/
func (b *Buffer) HandleByteOutput(stdOut io.ReadCloser) {
	TS_PACKAGE_MIN_SIZE := 188
	bufferSize := Settings.BufferSize * 1024 // in bytes
	buffer := make([]byte, bufferSize)
	var fileSize int
	init := true
	tmpFolder := b.Stream.Folder + string(os.PathSeparator)
	tmpSegment := b.Stream.LatestSegment

	var bufferVFS = b.FileSystem
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
				b.Stream.ReportError(err, CreateFileError, "", true)
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
			b.Stream.ReportError(err, EndOfFileError, "", true)
			return
		}
		if err != nil {
			f.Close()
			bufferVFS.Remove(tmpFile)
			b.Stream.ReportError(err, ReadIntoBufferError, "", true)
			return
		}
		if _, err := f.Write(buffer[:n]); err != nil {
			f.Close()
			bufferVFS.Remove(tmpFile)
			b.Stream.ReportError(err, WriteToBufferError, "", true)
			return
		}
		fileSize += n
		// Check if the file size exceeds the threshold
		if fileSize >= TS_PACKAGE_MIN_SIZE * 1024 {
			tmpSegment++
			tmpFile = fmt.Sprintf("%s%d.ts", tmpFolder, tmpSegment)
			// Close the current file and create a new one
			f.Close()
			b.Stream.LatestSegment = tmpSegment
			f, err = bufferVFS.Create(tmpFile)
			if err != nil {
				f.Close()
				b.Stream.ReportError(err, CreateFileError, "", true)
				return
			}
			fileSize = 0
		}
	}
}

/*
PrepareBufferFolder will clean the buffer folder and check if the folder exists
*/
func (b *Buffer) PrepareBufferFolder(folder string) error {
	if err := b.FileSystem.RemoveAll(getPlatformPath(folder)); err != nil {
		return fmt.Errorf("failed to remove buffer folder: %w", err)
	}

	if err := checkVFSFolder(folder, b.FileSystem); err != nil {
		return fmt.Errorf("failed to check buffer folder: %w", err)
	}

	return nil
}
