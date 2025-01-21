package src

import "io"

type BufferInterface interface {
	StartBuffer(stream *Stream) error
    HandleByteOutput(stdOut io.ReadCloser)
    PrepareBufferFolder(folder string) error
    GetBufTmpFiles() []string
    GetBufferedSize() int
    addBufferedFilesToPipe()
    DeleteOldestSegment()
    CheckBufferFolder() (bool, error)
    CheckBufferedFile(file string) (bool, error)
    writeToPipe(file string) error
}