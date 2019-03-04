package formulator

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"io/ioutil"
	"net"
	"sync"
	"time"

	"github.com/fletaio/common"
	"github.com/fletaio/common/util"
	"github.com/fletaio/framework/message"
)

// Peer is a peer of the formulator
type Peer struct {
	id        string
	netAddr   string
	conn      net.Conn
	pubhash   common.PublicHash
	writeChan chan []byte
}

// NewPeer returns a Peer
func NewPeer(conn net.Conn, pubhash common.PublicHash) *Peer {
	p := &Peer{
		id:        pubhash.String(),
		netAddr:   conn.RemoteAddr().String(),
		conn:      conn,
		pubhash:   pubhash,
		writeChan: make(chan []byte, 10),
	}
	go func() {
		for {
			select {
			case bs := <-p.writeChan:
				errCh := make(chan error)
				var wg sync.WaitGroup
				wg.Add(1)
				go func() {
					wg.Done()
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
					_, err := p.conn.Write(wbs)
					if err != nil {
						p.conn.Close()
					}
					errCh <- err
				}()
				wg.Wait()
				deadTimer := time.NewTimer(5 * time.Second)
				select {
				case <-deadTimer.C:
					p.conn.Close()
					return
				case err := <-errCh:
					deadTimer.Stop()
					if err != nil {
						return
					}
				}
			}
		}
	}()
	return p
}

// ID is the public hash of the peer formulator
func (p *Peer) ID() string {
	return p.id
}

// NetAddr is the network address of the peer
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
