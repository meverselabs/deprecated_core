package observer

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"io/ioutil"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fletaio/common"
	"github.com/fletaio/common/util"
	"github.com/fletaio/framework/message"
)

// FormulatorPeer is a formulator peer
type FormulatorPeer struct {
	sync.Mutex
	id          string
	netAddr     string
	conn        net.Conn
	pubhash     common.PublicHash
	address     common.Address
	guessHeight uint32
	startTime   uint64
	readTotal   uint64
	writeTotal  uint64
	writeChan   chan []byte
}

// NewFormulatorPeer returns a ormulatorPeer
func NewFormulatorPeer(conn net.Conn, pubhash common.PublicHash, address common.Address) *FormulatorPeer {
	p := &FormulatorPeer{
		id:        address.String(),
		netAddr:   conn.RemoteAddr().String(),
		conn:      conn,
		pubhash:   pubhash,
		address:   address,
		startTime: uint64(time.Now().UnixNano()),
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
					atomic.AddUint64(&p.writeTotal, uint64(len(wbs)))
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

// ID returns the id of the peer
func (p *FormulatorPeer) ID() string {
	return p.id
}

// NetAddr returns the network address of the peer
func (p *FormulatorPeer) NetAddr() string {
	return p.netAddr
}

// Address returns the formulator address of the peer
func (p *FormulatorPeer) Address() common.Address {
	return p.address
}

// ReadMessageData returns a message data
func (p *FormulatorPeer) ReadMessageData() (message.Type, []byte, error) {
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
func (p *FormulatorPeer) Send(m message.Message) error {
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
func (p *FormulatorPeer) SendRaw(bs []byte) error {
	p.writeChan <- bs
	return nil
}

// UpdateGuessHeight updates the guess height of the peer
func (p *FormulatorPeer) UpdateGuessHeight(height uint32) {
	p.Lock()
	defer p.Unlock()

	p.guessHeight = height
}

// GuessHeight updates the guess height of the peer
func (p *FormulatorPeer) GuessHeight() uint32 {
	return p.guessHeight
}
