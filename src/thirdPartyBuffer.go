package src

import (
	"bufio"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

type ThirdPartyBuffer struct {
	StreamBuffer
	Cmd    *exec.Cmd
	BufferType string
	Path       string
	Options    string
}


// StartThirdPartyBuffer starts the third party tool and capture its output for the given stream.
func (sb *ThirdPartyBuffer) StartBuffer(stream *Stream) error {   
	if err := sb.StreamBuffer.StartBuffer(stream); err != nil {
		return err
	}

	sb.SetBufferConfig()
	if sb.BufferType == "" {
		return fmt.Errorf("could not set buffer config")
	}

	ShowInfo(fmt.Sprintf("Streaming: Buffer:%s path:%s", sb.BufferType, sb.Path))
	ShowInfo("Streaming URL:" + stream.URL)

	err := sb.RunBufferCommand(stream)
	if err != nil {
		stream.handleBufferError(err)
	}
	return nil
}

// SetBufferConfig returns the the arguments from the buffer settings in the config file
func (sb *ThirdPartyBuffer) SetBufferConfig() {
	sb.BufferType = strings.ToUpper(Settings.Buffer)
	switch sb.BufferType {
	case "FFMPEG":
		sb.Options = Settings.FFmpegOptions
		sb.Path = Settings.FFmpegPath
	case "VLC":
		sb.Options = Settings.VLCOptions
		sb.Path = Settings.VLCPath
	default:
		sb.BufferType = ""
		sb.Options = ""
		sb.Path = ""
	}
}

// RunBufferCommand starts the third party tool process with the specified path and options, and captures its output.
//
// Parameters:
//   - bufferType: A string specifying the type of buffer (e.g., "FFMPEG", "VLC").
//   - path: A string specifying the path to the buffer executable.
//   - options: A string specifying the options to be passed to the buffer executable.
//   - stream: A pointer to a Stream struct containing the stream information.
//   - errorChan: A channel for sending error information.
//
// Returns:
//   - *Buffer: A pointer to a Buffer struct representing the buffer process.
//   - error: An error object if an error occurs, otherwise nil.
func (sb *ThirdPartyBuffer) RunBufferCommand(stream *Stream) error {
	sb.Stream = stream
	args := PrepareBufferArguments(sb.Options, stream.URL)

	cmd := exec.Command(sb.Path, args...)
	debug := fmt.Sprintf("%s:%s %s", strings.ToUpper(Settings.Buffer), sb.Path, args)
	ShowDebug(debug, 1)

	stdOut, stdErr, err := GetCommandPipes(cmd)
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start buffer command: %w", err)
	}
	WritePIDtoDisk(fmt.Sprintf("%d", cmd.Process.Pid))

	go ShowCommandStdErrInConsole(sb.BufferType, stdErr)
	go sb.HandleByteOutput(stdOut)

	sb.Cmd = cmd
	return nil
}

// PrepareBufferArguments replaces the [URL] placeholder in the buffer options with the actual stream URL
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

// GetCommandPipes retrieves the standard output and standard error pipes of the given command.
// It returns the stdout pipe, stderr pipe, and an error if any occurs.
//
// Parameters:
//   - cmd: A pointer to an exec.Cmd struct representing the command to be executed.
//
// Returns:
//   - io.ReadCloser: A ReadCloser for the standard output pipe.
//   - io.ReadCloser: A ReadCloser for the standard error pipe.
//   - error: An error object if an error occurs, otherwise nil.
func GetCommandPipes(cmd *exec.Cmd) (io.ReadCloser, io.ReadCloser, error) {
	stdOut, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	stdErr, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	return stdOut, stdErr, nil
}

// ShowCommandStdErrInConsole reads from the provided io.ReadCloser (stdErr) line by line,
// and logs each line with the specified bufferType prefix. If an error occurs during scanning,
// it logs the error with a specific error code.
//
// Parameters:
//   - bufferType: A string that specifies the type of buffer, used as a prefix in the log.
//   - stdErr: An io.ReadCloser from which the function reads the standard error output.
//
// The function uses a bufio.Scanner to read the standard error output line by line,
// trims any leading or trailing whitespace from each line, and logs it using the ShowInfo function.
// If an error occurs during scanning, it logs the error using the ShowError function with error code 4018.
func ShowCommandStdErrInConsole(bufferType string, stdErr io.ReadCloser) {
	scanner := bufio.NewScanner(stdErr)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		debug := fmt.Sprintf("%s log:%s", bufferType, strings.TrimSpace(scanner.Text()))
		ShowInfo(debug)
	}

	if scanner.Err() != nil {
		ShowError(scanner.Err(), 4018)
	}
}

// WritePIDtoDisk saves the PID of the buffering process to a file on disk
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

// DeletPIDfromDisc deletes the PID from the disk
// DeletPIDfromDisc removes a specified PID from a file on disk.
// The file is expected to be located in the system's temporary folder and named "PIDs".
// Each line in the file represents a PID.
//
// Parameters:
//
//	delete_pid (string): The PID to be removed from the file.
//
// Returns:
//
//	error: An error object if an error occurs, otherwise nil.
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

func (sb *ThirdPartyBuffer) HandleByteOutput(stdOut io.ReadCloser) {
    sb.StreamBuffer.HandleByteOutput(stdOut)
}

func (sb *ThirdPartyBuffer) PrepareBufferFolder(folder string) error {
    return sb.StreamBuffer.PrepareBufferFolder(folder)
}

func (sb *ThirdPartyBuffer) GetBufTmpFiles() []string {
    return sb.StreamBuffer.GetBufTmpFiles()
}

func (sb *ThirdPartyBuffer) GetBufferedSize() int {
    return sb.StreamBuffer.GetBufferedSize()
}

func (sb *ThirdPartyBuffer) addBufferedFilesToPipe() {
    sb.StreamBuffer.addBufferedFilesToPipe()
}

func (sb *ThirdPartyBuffer) DeleteOldestSegment() {
    sb.StreamBuffer.DeleteOldestSegment()
}

func (sb *ThirdPartyBuffer) CheckBufferFolder() (bool, error) {
    return sb.StreamBuffer.CheckBufferFolder()
}

func (sb *ThirdPartyBuffer) CheckBufferedFile(file string) (bool, error) {
    return sb.StreamBuffer.CheckBufferedFile(file)
}

func (sb *ThirdPartyBuffer) writeToPipe(file string) error {
    return sb.StreamBuffer.writeToPipe(file)
}
