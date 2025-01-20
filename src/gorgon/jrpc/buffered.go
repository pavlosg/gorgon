package jrpc

import (
	"errors"
	"io"
)

const bufferSize = 4096

type BufferedStream struct {
	underlying io.ReadWriteCloser
	begin      int
	end        int
	buffer     [bufferSize]byte
}

func NewBufferedStream(underlying io.ReadWriteCloser) *BufferedStream {
	return &BufferedStream{underlying: underlying}
}

var (
	errNegativeRead = errors.New("underlying returned negative count from Read")
	errLineTooLong  = errors.New("line too long")
)

func (b *BufferedStream) readLine() (line []byte, err error) {
	line = make([]byte, 0, 200)
	for {
		if b.begin >= b.end {
			b.begin = 0
			b.end, err = b.underlying.Read(b.buffer[:])
			if err != nil {
				return
			}
			if b.end < 0 {
				panic(errNegativeRead)
			}
		}
		for b.begin < b.end {
			if len(line) >= bufferSize {
				err = errLineTooLong
				return
			}
			i := b.begin
			line = append(line, b.buffer[i])
			b.begin++
			if b.buffer[i] == '\n' {
				return
			}
		}
	}
}

func (b *BufferedStream) Read(p []byte) (n int, err error) {
	if b.begin >= b.end {
		if len(p) >= bufferSize {
			return b.underlying.Read(p)
		}
		b.begin = 0
		b.end, err = b.underlying.Read(b.buffer[:])
		if err != nil {
			return
		}
		if b.end < 0 {
			panic(errNegativeRead)
		}
	}
	n = minInt(b.end-b.begin, len(p))
	copy(p, b.buffer[b.begin:b.begin+n])
	b.begin += n
	return
}

func (b *BufferedStream) Write(p []byte) (int, error) {
	return b.underlying.Write(p)
}

func (b *BufferedStream) Close() error {
	return b.underlying.Close()
}

func minInt(a, b int) int {
	if b < a {
		return b
	}
	return a
}
