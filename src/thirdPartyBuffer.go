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

/*
GetBufferConfig reutrns the the arguments from the buffer settings
*/
func GetBufferConfig() (bufferType, path, options string) {
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

/*
Buffer starts the third party tool and capture its output
*/
func Buffer(stream *Stream, useBackup bool, backupNumber int, errorChan chan ErrorInfo) *exec.Cmd {
	if useBackup {
		UpdateStreamURLForBackup(stream, backupNumber)
	}

	bufferType, path, options := GetBufferConfig()
	if bufferType == "" {
		return nil
	}

	if err := PrepareBufferFolder(stream.Folder); err != nil {
		ShowError(err, 4008)
		HandleBufferError(err, backupNumber, useBackup, stream, errorChan)
		return nil
	}

	showInfo(fmt.Sprintf("%s path:%s", bufferType, path))
	showInfo("Streaming URL:" + stream.URL)

	if cmd, err := RunBufferCommand(bufferType, path, options, stream, errorChan); err != nil {
		return HandleBufferError(err, backupNumber, useBackup, stream, errorChan)
	} else {
		return cmd
	}
}

/*
UpdateStreamURLForBackup will set the ther stream url when a backup will be used
*/
func UpdateStreamURLForBackup(stream *Stream, backupNumber int) {
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

/*
HandleBufferError will retry running the Buffer function with the next backup number
*/
func HandleBufferError(err error, backupNumber int, useBackup bool, stream *Stream, errorChan chan ErrorInfo) *exec.Cmd {
	ShowError(err, 4011)
	if !useBackup || (useBackup && backupNumber >= 0 && backupNumber <= 3) {
		backupNumber++
		if stream.BackupChannel1URL != "" || stream.BackupChannel2URL != "" || stream.BackupChannel3URL != "" {
			return Buffer(stream, true, backupNumber, errorChan)
		}
	}
	return nil
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

/*
RunBufferCommand starts the third party tool process
*/
func RunBufferCommand(bufferType string, path, options string, stream *Stream, errorChan chan ErrorInfo) (*exec.Cmd, error) {
	args := PrepareBufferArguments(options, stream.URL)

	cmd := exec.Command(path, args...)
	debug := fmt.Sprintf("%s:%s %s", strings.ToUpper(Settings.Buffer), path, args)
	showDebug(debug, 1)

	stdOut, logOut, err := GetCommandPipes(cmd)
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start buffer command: %w", err)
	}
	WritePIDtoDisk(fmt.Sprintf("%d", cmd.Process.Pid))

	var streamStatus = make(chan bool)
	go ShowCommandLogOutputInConsole(bufferType, logOut, streamStatus)
	go HandleCommandOutput(stdOut, stream, errorChan)

	return cmd, nil
}

/*
PrepareBufferArguments
*/
func PrepareBufferArguments(options, url string) []string {
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

/*
Get the output pipes of the given command
*/
func GetCommandPipes(cmd *exec.Cmd) (io.ReadCloser, io.ReadCloser, error) {
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

/*
ShowCommandLogOutputInConsole prints the log output of the given pipe
*/
func ShowCommandLogOutputInConsole(bufferType string, logOut io.ReadCloser, streamStatus chan bool) {
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

/*
HandleCommandOutput save the byte ouptut of the command as files
*/
func HandleCommandOutput(stdOut io.ReadCloser, stream *Stream, errorChan chan ErrorInfo) {
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
				ShowError(err, 4010)
				errorChan <- ErrorInfo{CreateFileError, stream, ""}
				return
			}
			init = false
		}
		n, err := reader.Read(buffer)
		if err == io.EOF {
			f.Close()
			showDebug("Buffer pipe reached EOF!", 3)
			errorChan <- ErrorInfo{EndOfFileError, stream, ""}
			return
		}
		if err != nil {
			ShowError(err, 4012)
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
WritePIDtoDisk saves the PID of the buffering process
*/
func WritePIDtoDisk(pid string) {
	// Open the file in append mode (create it if it doesn't exist)
	file, err := os.OpenFile(System.Folder.Temp+"PIDs", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0660)
	if err != nil {
		ShowError(err, 4040)
	}
	defer file.Close()

	// Write your text to the file
	_, err = file.WriteString(pid + "\n")
	if err != nil {
		ShowError(err, 4041)
	}
}

/*
DeletPIDfromDisc deletes the PID from the disk
*/
func DeletPIDfromDisc(delete_pid string) error {
	file, err := os.OpenFile(System.Folder.Temp+"PIDs", os.O_RDWR, 0660)
	if err != nil {
		ShowError(err, 4042)
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
		ShowError(err, 4043)
		return err
	}

	updatedPIDs := []string{}
	for index, pid := range pids {
		if pid != delete_pid {
			// Create a new slice by excluding the element at the specified index
			_, err = file.WriteString(pid + "\n")
			if err != nil {
				ShowError(err, 4044)
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
			ShowError(err, 4045)
			return err
		}
	}
	return nil
}
