package src

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/avfs/avfs"
)

func (sb *StreamBuffer) StartBuffer(stream *Stream) {
	sb.Stream = stream
	var err error = nil
	if err = sb.PrepareBufferFolder(stream.Folder); err != nil {
		// If something went wrong when setting up the buffer storage don't run at all
		stream.ReportError(err, BufferFolderError, "", true)
		return
	}

	switch Settings.Buffer {
	case "ffmpeg", "vlc":
		sb.IsThirdPartyBuffer = true
		err = StartThirdPartyBuffer(stream)
	case "threadfin":
		err = StartThreadfinBuffer(stream)
	default:
		return
	}
	if err != nil {
		sb.Stream.handleBufferError(err)
	}
}

/*
HandleByteOutput save the byte ouptut of the command or http request as files
*/
func (sb *StreamBuffer) HandleByteOutput(stdOut io.ReadCloser) {
	TS_PACKAGE_MIN_SIZE := 188
	bufferSize := Settings.BufferSize * 1024 // in bytes
	buffer := make([]byte, bufferSize)
	var fileSize int
	init := true
	tmpFolder := sb.Stream.Folder + string(os.PathSeparator)
	tmpSegment := sb.LatestSegment

	var bufferVFS = sb.FileSystem
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
				sb.Stream.ReportError(err, CreateFileError, "", true)
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
			sb.Stream.ReportError(err, EndOfFileError, "", true)
			return
		}
		if err != nil {
			f.Close()
			bufferVFS.Remove(tmpFile)
			sb.Stream.ReportError(err, ReadIntoBufferError, "", true)
			return
		}
		if _, err := f.Write(buffer[:n]); err != nil {
			f.Close()
			bufferVFS.Remove(tmpFile)
			sb.Stream.ReportError(err, WriteToBufferError, "", true)
			return
		}
		fileSize += n
		// Check if the file size exceeds the threshold
		if fileSize >= TS_PACKAGE_MIN_SIZE*1024 {
			tmpSegment++
			tmpFile = fmt.Sprintf("%s%d.ts", tmpFolder, tmpSegment)
			// Close the current file and create a new one
			f.Close()
			sb.LatestSegment = tmpSegment
			f, err = bufferVFS.Create(tmpFile)
			if err != nil {
				f.Close()
				sb.Stream.ReportError(err, CreateFileError, "", true)
				return
			}
			fileSize = 0
		}
	}
}

/*
PrepareBufferFolder will clean the buffer folder and check if the folder exists
*/
func (sb *StreamBuffer) PrepareBufferFolder(folder string) error {
	if err := sb.FileSystem.RemoveAll(getPlatformPath(folder)); err != nil {
		return fmt.Errorf("failed to remove buffer folder: %w", err)
	}

	if err := checkVFSFolder(folder, sb.FileSystem); err != nil {
		return fmt.Errorf("failed to check buffer folder: %w", err)
	}

	return nil
}

/*
GetBufTmpFiles retrieves the files within the buffer folder
and returns a sorted list with the file names
*/
func (sb *StreamBuffer) GetBufTmpFiles() (tmpFiles []string) {

	var tmpFolder = sb.Stream.Folder + string(os.PathSeparator)
	var fileIDs []float64

	if _, err := sb.FileSystem.Stat(tmpFolder); !fsIsNotExistErr(err) {

		files, err := sb.FileSystem.ReadDir(getPlatformPath(tmpFolder))
		if err != nil {
			ShowError(err, 000)
			return
		}

		// Check if more then one file is available
		if len(files) >= 1 {
			// Iterate over the files and collect the IDs
			for _, file := range files {
				if !file.IsDir() && filepath.Ext(file.Name()) == ".ts" {
					fileID := strings.TrimSuffix(file.Name(), ".ts")
					if f, err := strconv.ParseFloat(fileID, 64); err == nil {
						fileIDs = append(fileIDs, f)
					}
				}
			}

			sort.Float64s(fileIDs)
			if len(fileIDs) > 0 {
				fileIDs = fileIDs[:len(fileIDs)-1]
			}

			// Create the return array with the sorted files
			for _, file := range fileIDs {
				fileName := fmt.Sprintf("%.0f.ts", file)
				// Check if the file is already within old segments array
				if !ContainsString(sb.OldSegments, fileName) {
					tmpFiles = append(tmpFiles, fileName)
					sb.OldSegments = append(sb.OldSegments, fileName)
				}
			}
		}
	}
	return
}

func (sb *StreamBuffer) GetBufferedSize() (size int) {
	size = 0
	var tmpFolder = sb.Stream.Folder + string(os.PathSeparator)
	if _, err := sb.FileSystem.Stat(tmpFolder); !fsIsNotExistErr(err) {

		files, err := sb.FileSystem.ReadDir(getPlatformPath(tmpFolder))
		if err != nil {
			ShowError(err, 000)
			return
		}
		for _, file := range files {
			if !file.IsDir() && filepath.Ext(file.Name()) == ".ts" {
				file_info, err := sb.FileSystem.Stat(getPlatformFile(tmpFolder + file.Name()))
				if err == nil {
					size += int(file_info.Size())
				}
			}
		}
	}
	return size
}

func (sb *StreamBuffer) addBufferedFilesToPipe() {
	for {
		select {
		case <-sb.StopChan:
			return
		default:
			if sb.GetBufferedSize() < Settings.BufferSize * 1024 {
				time.Sleep(25 * time.Millisecond) // Wait for new files
				continue
			}
			tmpFiles := sb.GetBufTmpFiles()
			for _, f := range tmpFiles {
				if ok, err := sb.CheckBufferFolder(); !ok {
					sb.Stream.ReportError(err, BufferFolderError, "", true)
					return
				}
				sb.OldSegments = append(sb.OldSegments, f)
				ShowDebug(fmt.Sprintf("Streaming:Broadcasting file %s to clients", f), 1)
				err := sb.writeToPipe(f) // Add file so it will be copied to the pipes
				if err != nil {
					sb.Stream.ReportError(err, 0, "", false)

				}
				sb.DeleteOldestSegment()
			}
		}
	}
}

/*
DeleteOldesSegment will delete the file provided in the buffer
*/
func (sb *StreamBuffer) DeleteOldestSegment() {
	fileToRemove := sb.Stream.Folder + string(os.PathSeparator) + sb.OldSegments[0]
	if err := sb.FileSystem.RemoveAll(getPlatformFile(fileToRemove)); err != nil {
		ShowError(err, 4007)
	}
}

/*
CheckBufferFolder reports whether the buffer folder exists.
*/
func (sb *StreamBuffer) CheckBufferFolder() (bool, error) {
	if _, err := sb.FileSystem.Stat(sb.Stream.Folder); fsIsNotExistErr(err) {
		return false, err
	}
	return true, nil
}

// CheckBufferedFile check for the existance of the given file (file path is needed)
func (sb *StreamBuffer) CheckBufferedFile(file string) (bool, error) {
	if _, err := sb.FileSystem.Stat(file); fsIsNotExistErr(err) {
		return false, err
	}
	return true, nil
}

func (sb *StreamBuffer) writeToPipe(file string) error {
	f, err := sb.FileSystem.Open(filepath.Join(sb.Stream.Folder, file))
	if err != nil {
		return err
	}
	_, err = io.Copy(sb.PipeWriter, f)
	if err != nil {
		f.Close()
		sb.Stream.ReportError(err, 0, "", true)
	}
	f.Close()
	return nil
}
