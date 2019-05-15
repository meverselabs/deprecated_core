package event

import (
	"encoding/json"
	"io"

	"github.com/fletaio/common"
	"github.com/fletaio/common/util"
)

// Type is event type
type Type uint8

// Event is a interface that defines common event functions
type Event interface {
	io.WriterTo
	io.ReaderFrom
	json.Marshaler
	Coord() *common.Coordinate
	Index() uint16
	Type() Type
	SetIndex(index uint16)
}

// Base is the parts of event functions that are not changed by derived one
type Base struct {
	Coord_ *common.Coordinate
	Index_ uint16
	Type_  Type
}

// Coord returns the coord of the event
func (e *Base) Coord() *common.Coordinate {
	return e.Coord_
}

// Index returns the index of the event
func (e *Base) Index() uint16 {
	return e.Index_
}

// Type returns the type of the event
func (e *Base) Type() Type {
	return e.Type_
}

// SetIndex changes the index of the event
func (e *Base) SetIndex(index uint16) {
	e.Index_ = index
}

// WriteTo is a serialization function
func (e *Base) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	if n, err := e.Coord_.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := util.WriteUint16(w, e.Index_); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := util.WriteUint64(w, uint64(e.Type_)); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	return wrote, nil
}

// ReadFrom is a deserialization function
func (e *Base) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if n, err := e.Coord_.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if v, n, err := util.ReadUint16(r); err != nil {
		return read, err
	} else {
		read += n
		e.Index_ = v
	}
	if v, n, err := util.ReadUint64(r); err != nil {
		return read, err
	} else {
		read += n
		e.Type_ = Type(v)
	}
	return read, nil
}
