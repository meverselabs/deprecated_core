package data

import (
	"io"

	"git.fleta.io/fleta/common/util"
)

// Data TODO
type Data interface {
	Version() uint16
	WriteTo(w io.Writer) (int64, error)
	ReadFrom(r io.Reader) (int64, error)
}

// Base TODO
type Base struct {
	Version_ uint16
}

// Version TODO
func (d *Base) Version() uint16 {
	return d.Version_
}

// NewBase TODO
func NewBase(version uint16) *Base {
	return &Base{
		Version_: version,
	}
}

// WriteTo TODO
func (d *Base) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	if n, err := util.WriteUint16(w, d.Version_); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	return wrote, nil
}

// ReadFrom TODO
func (d *Base) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if v, n, err := util.ReadUint16(r); err != nil {
		return read, err
	} else {
		read += n
		d.Version_ = v
	}
	return read, nil
}
