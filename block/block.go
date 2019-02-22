package block

import (
	"bytes"
	"io"
)

// Block includes a block header and a block body
type Block struct {
	Header *Header
	Body   *Body
}

// WriteTo is a serialization function
func (b *Block) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	if n, err := b.Header.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := b.Body.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	return wrote, nil
}

// ReadFrom is a deserialization function
func (b *Block) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if n, err := b.Header.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if n, err := b.Body.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	return read, nil
}

// MarshalJSON is a marshaler function
func (b *Block) MarshalJSON() ([]byte, error) {
	var buffer bytes.Buffer
	buffer.WriteString(`{`)
	buffer.WriteString(`"hash":`)
	h := b.Header.Hash()
	if bs, err := h.MarshalJSON(); err != nil {
		return nil, err
	} else {
		buffer.Write(bs)
	}
	buffer.WriteString(`,`)
	buffer.WriteString(`"header":`)
	if bs, err := b.Header.MarshalJSON(); err != nil {
		return nil, err
	} else {
		buffer.Write(bs)
	}
	buffer.WriteString(`,`)
	buffer.WriteString(`"body":`)
	if bs, err := b.Body.MarshalJSON(); err != nil {
		return nil, err
	} else {
		buffer.Write(bs)
	}
	buffer.WriteString(`}`)
	return buffer.Bytes(), nil
}
