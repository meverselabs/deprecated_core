package observer

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"io/ioutil"
	"net"
	"sync/atomic"
	"time"

	"github.com/fletaio/common"
	"github.com/fletaio/common/util"
	"github.com/fletaio/framework/message"
)

// Peer is a observer peer
type Peer struct {
	id         string
	netAddr    string
	conn       net.Conn
	pubhash    common.PublicHash
	startTime  uint64
	readTotal  uint64
	writeTotal uint64
	writeChan  chan []byte
}

// NewPeer returns a Peer
func NewPeer(conn net.Conn, pubhash common.PublicHash) *Peer {
	p := &Peer{
		id:        pubhash.String(),
		netAddr:   conn.RemoteAddr().String(),
		conn:      conn,
		pubhash:   pubhash,
		startTime: uint64(time.Now().UnixNano()),
		writeChan: make(chan []byte, 10),
	}
	go func() {
		defer p.conn.Close()

		for {
			select {
			case bs := <-p.writeChan:
				var buffer bytes.Buffer
				buffer.Write(bs[:8])          // message type
				buffer.Write(make([]byte, 4)) //size of gzip
				if len(bs) > 8 {
					zw := gzip.NewWriter(&buffer)
					zw.Write(bs[8:])
					zw.Flush()
					zw.Close()
				}
				wbs := buffer.Bytes()
				binary.LittleEndian.PutUint32(wbs[8:], uint32(len(wbs)-12))
				if err := p.conn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
					return
				}
				_, err := p.conn.Write(wbs)
				if err != nil {
					return
				}
				atomic.AddUint64(&p.writeTotal, uint64(len(wbs)))
			}
		}
	}()
	return p
}

// ID returns the id of the peer
func (p *Peer) ID() string {
	return p.id
}

// NetAddr returns the network address of the peer
func (p *Peer) NetAddr() string {
	return p.netAddr
}

// ReadMessageData returns a message data
func (p *Peer) ReadMessageData() (message.Type, []byte, error) {
	var t message.Type
	if v, _, err := util.ReadUint64(p.conn); err != nil {
		return 0, nil, err
	} else {
		t = message.Type(v)
	}

	if Len, _, err := util.ReadUint32(p.conn); err != nil {
		return 0, nil, err
	} else if Len == 0 {
		return t, nil, nil
	} else {
		zbs := make([]byte, Len)
		if _, err := util.FillBytes(p.conn, zbs); err != nil {
			return 0, nil, err
		}
		zr, err := gzip.NewReader(bytes.NewReader(zbs))
		if err != nil {
			return 0, nil, err
		}
		defer zr.Close()

		bs, err := ioutil.ReadAll(zr)
		if err != nil {
			return 0, nil, err
		}
		return t, bs, nil
	}
}

// Send sends a message to the peer
func (p *Peer) Send(m message.Message) error {
	var buffer bytes.Buffer
	if _, err := util.WriteUint64(&buffer, uint64(m.Type())); err != nil {
		return err
	}
	if _, err := m.WriteTo(&buffer); err != nil {
		return err
	}
	return p.SendRaw(buffer.Bytes())
}

// SendRaw sends bytes to the peer
func (p *Peer) SendRaw(bs []byte) error {
	p.writeChan <- bs
	return nil
}
