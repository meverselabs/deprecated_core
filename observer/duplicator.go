package observer

import (
	"bytes"
	"io"
)

// Duplicator duplicates bytes when reader was read
type Duplicator struct {
	reader io.Reader
	buffer bytes.Buffer
}

// NewDuplicator returns a Duplicator
func NewDuplicator(reader io.Reader) *Duplicator {
	return &Duplicator{
		reader: reader,
	}
}

// Read reads bytes from the reader and stores same bytes to the buffer
func (r *Duplicator) Read(bs []byte) (int, error) {
	n, err := r.reader.Read(bs)
	if err != nil {
		return n, err
	}
	r.buffer.Write(bs[:n])
	return n, err
}
