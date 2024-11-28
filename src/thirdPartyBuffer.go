package src

import (
	"bufio"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"
)

/*
StartThirdPartyBuffer starts the third party tool and capture its output
*/
func StartThirdPartyBuffer(stream *Stream, useBackup bool, backupNumber int, errorChan chan ErrorInfo) (*Buffer, error) {
	if useBackup {
		UpdateStreamURLForBackup(stream, backupNumber)
	}

	bufferType, path, options := GetBufferConfig()
	if bufferType == "" {
		return nil, fmt.Errorf("could not get buffer config")
	}

	showInfo(fmt.Sprintf("Streaming: Buffer:%s path:%s", bufferType, path))
	showInfo("Streaming URL:" + stream.URL)

	if buffer, err := RunBufferCommand(bufferType, path, options, stream, errorChan); err != nil {
		return HandleBufferError(err, backupNumber, useBackup, stream, errorChan), err
	} else {
		return buffer, nil
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
		return bufferType, Settings.VLCPath, Settings.VLCOptions
	default:
		return "", "", ""
	}
}

/*
RunBufferCommand starts the third party tool process
*/
func RunBufferCommand(bufferType string, path, options string, stream *Stream, errorChan chan ErrorInfo) (*Buffer, error) {
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
	go HandleByteOutput(stdOut, stream, errorChan)

	buffer := &Buffer{
		isThirdPartyBuffer: true,
		cmd:                cmd,
	}

	return buffer, nil
}

/*
PrepareBufferArguments
*/
func PrepareBufferArguments(options, streamURL string) []string {
	args := []string{}
	u, err := url.Parse(streamURL)
	if err != nil {
		return []string{}
	}
	for i, a := range strings.Split(options, " ") {
		a = strings.Replace(a, "[URL]", streamURL, 1)
		if i == 0 && len(Settings.UserAgent) != 0 && Settings.Buffer == "ffmpeg" && u.Scheme != "rtp" {
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
