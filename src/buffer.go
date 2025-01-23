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

type StreamBuffer struct {
	FileSystem         avfs.VFS
	Stream             *Stream // Reference to the parents struct
	StopChan           chan struct{}
	Stopped			   bool
	CloseChan		   chan struct{}
	Closed			   bool
	LatestSegment      int
	OldSegments        []string
	PipeWriter         *io.PipeWriter
	PipeReader         *io.PipeReader
}

const (
	BufferFolderError     = 4008
	SendFileError         = 4009
	CreateFileError       = 4010
	EndOfFileError        = 4011
	ReadIntoBufferError   = 4012
	WriteToBufferError    = 4013
	OpenFileError         = 4014 //errMsg = "Not able to open buffered file"
	FileStatError         = 4015 //errMsg = "Could not get file statics of buffered file"
	ReadFileError         = 4016 //errMsg = "Could not read buffered file before sending to clients"
	FileDoesNotExistError = 4019 //errMsg = "Buffered file does not exist anymore"
)

func (sb *StreamBuffer) StartBuffer(stream *Stream) error {
	sb.Stream = stream
	if err := sb.PrepareBufferFolder(filepath.Join(stream.Folder, "0.ts")); err != nil {
		// If something went wrong when setting up the buffer storage don't run at all
		stream.ReportError(err, BufferFolderError, "", true)
		return err
	}
	return nil
}

func (sb *StreamBuffer) StopBuffer() {
	if !sb.Stopped {
		sb.Stopped = true
		close(sb.StopChan)
	}
}

func (sb *StreamBuffer) CloseBuffer() {
	sb.StopBuffer()
	if ! sb.Closed {
		close(sb.CloseChan)
		sb.Closed = true
	}
	sb.RemoveBufferedFiles(filepath.Join(sb.Stream.Folder, "0.ts"))
}

func (sb *StreamBuffer) GetPipeReader() *io.PipeReader{
	return sb.PipeReader
}

func (sb *StreamBuffer) GetStopChan() chan struct{} {
	return sb.StopChan
}

func (sb *StreamBuffer) SetStopChan(stopChan chan struct{}) {
	sb.StopChan = stopChan
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
		select {
		case <- sb.CloseChan:
			// If the stream got stopped, stop the output
			return
		default:
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
}

func (sb *StreamBuffer) RemoveBufferedFiles(folder string) error {
	test := filepath.Dir(folder)
	if err := sb.FileSystem.RemoveAll(test); err != nil {
		return fmt.Errorf("failed to remove buffer folder: %w", err)
	}
	return nil
}

/*
PrepareBufferFolder will clean the buffer folder and check if the folder exists
*/
func (sb *StreamBuffer) PrepareBufferFolder(folder string) error {
	if err := sb.RemoveBufferedFiles(folder); err != nil {
		return err
	}

	if err := sb.createBufferFolder(filepath.Dir(folder)); err != nil {
		return fmt.Errorf("failed to create buffer folder: %w", err)
	}

	return nil
}

func (sb *StreamBuffer) createBufferFolder(path string) (err error) {

	var debug string
	_, err = sb.FileSystem.Stat(path)

	if fsIsNotExistErr(err) {
		// Folder does not exist, will now be created

		// If we are on Windows and the cache location path is NOT on C:\ we need to create the volume it is located on
		// Failure to do so here will result in a panic error and the stream not playing
		if Settings.StoreBufferInRAM {
			if sb.FileSystem.OSType() == avfs.OsWindows {
				vm := sb.FileSystem.(avfs.VolumeManager)
				pathIterator := avfs.NewPathIterator(sb.FileSystem, path)
				if pathIterator.VolumeName() != "C:" {
					vm.VolumeAdd(path)
				}
			}

			err = sb.FileSystem.MkdirAll(path, 0755)
			if err == nil {
				debug = fmt.Sprintf("Create virtual filesystem Folder: %s", path)
				ShowDebug(debug, 1)
			} else {
				return err
			}

		} else {
			err = sb.FileSystem.MkdirAll(path, 0755)
			if err == nil {
				debug = fmt.Sprintf("Created folder on disk: %s", path)
				ShowDebug(debug, 1)
			} else {
				return err
			}
		}

		return nil
	}

	return nil
}

/*
GetBufTmpFiles retrieves the files within the buffer folder
and returns a sorted list with the file names
*/
func (sb *StreamBuffer) GetBufTmpFiles() (tmpFiles []string) {

	var tmpFolder = sb.Stream.Folder
	var fileIDs []float64

	if _, err := sb.FileSystem.Stat(tmpFolder); !fsIsNotExistErr(err) {
		files, err := sb.FileSystem.ReadDir(tmpFolder)
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
	var tmpFolder = sb.Stream.Folder
	if _, err := sb.FileSystem.Stat(tmpFolder); !fsIsNotExistErr(err) {

		files, err := sb.FileSystem.ReadDir(tmpFolder)
		if err != nil {
			ShowError(err, 000)
			return
		}
		for _, file := range files {
			if !file.IsDir() && filepath.Ext(file.Name()) == ".ts" {
				file_info, err := sb.FileSystem.Stat(filepath.Join(tmpFolder, file.Name()))
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
	fileToRemove := filepath.Join(sb.Stream.Folder, sb.OldSegments[0])
	if err := sb.FileSystem.Remove(fileToRemove); err != nil {
		ShowError(err, 4007)
	}
	sb.OldSegments = sb.OldSegments[1:]
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
    defer f.Close()


	buf := make([]byte, 4096) // 4KB buffer
	for {
		select {
		case <- sb.StopChan:
			// Pipe was closed quit writing to it
			return nil
		default:
			n, err := f.Read(buf)
			if err != nil && err != io.EOF {
				sb.Stream.ReportError(err, 0, "", true) // TODO: Add error code
				return err
			} else if err == io.EOF {
				return nil
			}
			if n == 0 {
				break
			}

			_, err = sb.PipeWriter.Write(buf[:n])
			if err != nil {
				sb.Stream.ReportError(err, 0, "", true) // TODO: Add error code
				return err
			}
		}
	}
}
