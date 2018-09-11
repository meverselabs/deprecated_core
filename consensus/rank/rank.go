package rank

import (
	"bytes"
	"encoding/binary"
	"io"
	"strconv"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/common/util"
)

// Rank TODO
type Rank struct {
	PublicKey common.PublicKey
	phase     uint32
	hashSpace hash.Hash256
	score     uint64
}

// NewRank TODO
func NewRank(PublicKey common.PublicKey, phase uint32, hashSpace hash.Hash256) *Rank {
	m := &Rank{
		phase:     phase,
		hashSpace: hashSpace,
	}
	copy(m.PublicKey[:], PublicKey[:])
	m.update()
	return m
}

// Hash TODO
func (rank *Rank) Hash() (hash.Hash256, error) {
	var buffer bytes.Buffer
	if _, err := rank.WriteTo(&buffer); err != nil {
		return hash.Hash256{}, err
	}
	return hash.Hash(buffer.Bytes()), nil
}

// WriteTo TODO
func (rank *Rank) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	if n, err := rank.PublicKey.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := util.WriteUint32(w, rank.phase); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := rank.hashSpace.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	return wrote, nil
}

// ReadFrom TODO
func (rank *Rank) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if n, err := rank.PublicKey.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if v, n, err := util.ReadUint32(r); err != nil {
		return read, err
	} else {
		read += n
		rank.phase = v
	}
	if n, err := rank.hashSpace.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	rank.update()
	return read, nil
}

// Clone TODO
func (rank *Rank) Clone() *Rank {
	return NewRank(rank.PublicKey, rank.phase, rank.hashSpace)
}

// Score TODO
func (rank *Rank) Score() uint64 {
	return rank.score
}

// Phase TODO
func (rank *Rank) Phase() uint32 {
	return rank.phase
}

// HashSpace TODO
func (rank *Rank) HashSpace() hash.Hash256 {
	return rank.hashSpace
}

// Less TODO
func (rank *Rank) Less(b *Rank) bool {
	return rank.score < b.score || (rank.score == b.score && bytes.Compare(rank.PublicKey[:], b.PublicKey[:]) < 0)
}

// Equal TODO
func (rank *Rank) Equal(b *Rank) bool {
	return rank.score == b.score && bytes.Equal(rank.PublicKey[:], b.PublicKey[:])
}

// IsZero TODO
func (rank *Rank) IsZero() bool {
	var emptyKey common.PublicKey
	return rank.score == 0 && bytes.Compare(rank.PublicKey[:], emptyKey[:]) == 0
}

// Set TODO
func (rank *Rank) Set(phase uint32, hashSpace hash.Hash256) {
	rank.phase = phase
	rank.hashSpace = hashSpace
	rank.update()
}

// SetPhase TODO
func (rank *Rank) SetPhase(phase uint32) {
	rank.phase = phase
	rank.update()
}

// SetHashSpace TODO
func (rank *Rank) SetHashSpace(hashSpace hash.Hash256) {
	rank.hashSpace = hashSpace
	rank.update()
}

func (rank *Rank) update() {
	rank.score = uint64(rank.phase)<<32 + uint64(binary.LittleEndian.Uint32(rank.hashSpace[:4]))
}

// Key TODO
func (rank *Rank) Key() string {
	bs := make([]byte, 8)
	binary.LittleEndian.PutUint64(bs, rank.score)
	return string(rank.PublicKey[:]) + "," + string(bs)
}

// String TODO
func (rank *Rank) String() string {
	/*
		bs := make([]byte, 8)
		binary.LittleEndian.PutUint64(bs, rank.score)
		return string(rank.PublicKey[:]) + "," + string(bs)
	*/
	key := rank.PublicKey.String()
	return "{" + key[:2] + key[len(key)-2:] + "," + strconv.FormatUint(uint64(rank.phase), 10) + "," + strconv.FormatUint(rank.score, 10) + "}"
}
