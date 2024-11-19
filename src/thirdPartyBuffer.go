package src

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/avfs/avfs/vfs/osfs"
)

func initBufferVFS(virtual bool) {

	if virtual {
		bufferVFS = memfs.New()
	} else {
		bufferVFS = osfs.New()
	}

}

func getBufferConfig() (bufferType, path, options string) {
	bufferType = strings.ToUpper(Settings.Buffer)
	switch bufferType {
	case "FFMPEG":
		return bufferType, Settings.FFmpegPath, Settings.FFmpegOptions
	case "VLC":
		return bufferType, Settings.VLCPath, fmt.Sprintf("%s meta-title=Threadfin", Settings.VLCOptions)
	default:
		return "", "", ""
	}
}

func buffer(stream *Stream, useBackup bool, backupNumber int, errorChan chan ErrorInfo) *exec.Cmd {
	if useBackup {
		updateStreamURLForBackup(stream, backupNumber)
	}

	bufferType, path, options := getBufferConfig()
	if bufferType == "" {
		return nil
	}

	if err := prepareBufferFolder(stream.Folder); err != nil {
		newHandleBufferError(err, backupNumber, useBackup, stream, errorChan)
		return nil
	}

	showInfo(fmt.Sprintf("%s path:%s", bufferType, path))
	showInfo("Streaming URL:" + stream.URL)

	if cmd, err := runBufferCommand(bufferType, path, options, stream, errorChan); err != nil {
		return newHandleBufferError(err, backupNumber, useBackup, stream, errorChan)
	} else {
		return cmd
	}
}

func updateStreamURLForBackup(stream *Stream, backupNumber int) {
	switch backupNumber {
	case 1:
		stream.URL = stream.BackupChannel1URL
		showHighlight("START OF BACKUP 1 STREAM")
		showInfo("Backup Channel 1 URL: " + stream.URL)
	case 2:
		stream.URL = stream.BackupChannel2URL
		showHighlight("START OF BACKUP 2 STREAM")
		showInfo("Backup Channel 2 URL: " + stream.URL)
	case 3:
		stream.URL = stream.BackupChannel3URL
		showHighlight("START OF BACKUP 3 STREAM")
		showInfo("Backup Channel 3 URL: " + stream.URL)
	}
}

func newHandleBufferError(err error, backupNumber int, useBackup bool, stream *Stream, errorChan chan ErrorInfo) *exec.Cmd {
	ShowError(err, 0)
	if !useBackup || (useBackup && backupNumber >= 0 && backupNumber <= 3) {
		backupNumber++
		if stream.BackupChannel1URL != "" || stream.BackupChannel2URL != "" || stream.BackupChannel3URL != "" {
			return buffer(stream, true, backupNumber, errorChan)
		}
	}
	return nil
}

func prepareBufferFolder(folder string) error {
	if err := bufferVFS.RemoveAll(getPlatformPath(folder)); err != nil {
		return fmt.Errorf("failed to remove buffer folder: %w", err)
	}

	if err := checkVFSFolder(folder, bufferVFS); err != nil {
		return fmt.Errorf("failed to check buffer folder: %w", err)
	}

	return nil
}

func runBufferCommand(bufferType string, path, options string, stream *Stream, errorChan chan ErrorInfo) (*exec.Cmd, error) {
	args := prepareBufferArguments(options, stream.URL)

	cmd := exec.Command(path, args...)
	debug := fmt.Sprintf("%s:%s %s", strings.ToUpper(Settings.Buffer), path, args)
	showDebug(debug, 1)

	stdOut, logOut, err := getCommandPipes(cmd)
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start buffer command: %w", err)
	}
	writePIDtoDisk(fmt.Sprintf("%d", cmd.Process.Pid))

	var streamStatus = make(chan bool)
	go showCommandLogOutputInConsole(bufferType, logOut, streamStatus)
	go handleCommandOutput(stdOut, stream, errorChan)

	return cmd, nil
}

func prepareBufferArguments(options, url string) []string {
	args := []string{}
	for i, a := range strings.Split(options, " ") {
		a = strings.Replace(a, "[URL]", url, -1)
		if i == 0 && len(Settings.UserAgent) != 0 {
			args = append(args, "-user_agent", Settings.UserAgent)
		}
		args = append(args, a)
	}
	return args
}

func getCommandPipes(cmd *exec.Cmd) (io.ReadCloser, io.ReadCloser, error) {
	stdOut, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	logOut, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	return stdOut, logOut, nil
}

func showCommandLogOutputInConsole(bufferType string, logOut io.ReadCloser, streamStatus chan bool) {
	// Log Daten vom Prozess im Debug Mode 1 anzeigen.
	scanner := bufio.NewScanner(logOut)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {

		debug := fmt.Sprintf("%s log:%s", bufferType, strings.TrimSpace(scanner.Text()))

		select {
		case <-streamStatus:
			showDebug(debug, 1)
		default:
			showInfo(debug)
		}

		time.Sleep(time.Duration(10) * time.Millisecond)

	}
}

func handleCommandOutput(stdOut io.ReadCloser, stream *Stream, errorChan chan ErrorInfo) {
	bufferSize := Settings.BufferSize * 1024 // Puffergröße in Bytes
	buffer := make([]byte, bufferSize)
	var fileSize int
	init := true
	tmpFolder := stream.Folder
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
				showInfo("DEBUG: Creating File did not work!")
				ShowError(err, 0)
				errorChan <- ErrorInfo{CreateFileError, stream, ""}
				return
			}
			init = false
		}
		n, err := reader.Read(buffer)
		if err == io.EOF {
			f.Close()
			showInfo("DEBUG: Reached EOF!")
			errorChan <- ErrorInfo{EndOfFileError, stream, ""}
			return
		}
		if err != nil {
			showInfo("DEBUG: Other error when trying to read into buffer!")
			ShowError(err, 0)
			f.Close()
			errorChan <- ErrorInfo{ReadIntoBufferError, stream, ""}
			return
		}
		if _, err := f.Write(buffer[:n]); err != nil {
			showInfo("DEBUG: Write file not working!")
			ShowError(err, 0)
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
				ShowError(err, 0)
				errorChan <- ErrorInfo{CreateFileError, stream, ""}
				return
			}
			fileSize = 0
		}
	}
}

func writePIDtoDisk(pid string) {
	// Open the file in append mode (create it if it doesn't exist)
	file, err := os.OpenFile(System.Folder.Temp+"PIDs", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0660)
	if err != nil {
		ShowError(err, 0) // TODO: Add new error code
	}
	defer file.Close()

	// Write your text to the file
	_, err = file.WriteString(pid + "\n")
	if err != nil {
		ShowError(err, 0) // TODO: Add new error code
	}
}

func deletPIDfromDisc(delete_pid string) error {
	file, err := os.OpenFile(System.Folder.Temp+"PIDs", os.O_RDWR, 0660)
	if err != nil {
		return err
	}
	// Create a scanner
	scanner := bufio.NewScanner(file)

	// Read line by line
	pids := []string{}
	for scanner.Scan() {
		line := scanner.Text()
		pids = append(pids, line)
	}

	// Rewind the file to the beginning
	_, err = file.Seek(0, 0)
	if err != nil {
		return err
	}

	updatedPIDs := []string{}
	for index, pid := range pids {
		if pid != delete_pid {
			// Create a new slice by excluding the element at the specified index
			_, err = file.WriteString(pid + "\n")
			if err != nil {
				return err
			}
		} else {
			updatedPIDs = append(pids[:index], pids[index+1:]...)
		}
	}

	// Truncate any remaining content (if the new slice is shorter)
	if len(updatedPIDs) < len(pids) {
		err = file.Truncate(int64(len(updatedPIDs)))
		if err != nil {
			return err
		}
	}
	return nil
}
