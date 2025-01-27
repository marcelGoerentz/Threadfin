package src

import "io"

type BufferInterface interface {
	StartBuffer(stream *Stream) error
    StopBuffer()
    CloseBuffer()
    HandleByteOutput(stdOut io.ReadCloser)
    PrepareBufferFolder(folder string) error
    GetBufTmpFiles() []string
    GetBufferedSize() int
    addBufferedFilesToPipe()
    DeleteOldestSegment()
    CheckBufferFolder() (bool, error)
    CheckBufferedFile(file string) (bool, error)
    writeToPipe(file string) error
    writeBytesToPipe(data []byte) error
    GetPipeReader() *io.PipeReader
    GetStopChan() chan struct{}
    SetStopChan(chan struct{})
}