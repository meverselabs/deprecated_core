package formulator

import (
	"bytes"
	"net"
	"sync"
	"time"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/util"
	"git.fleta.io/fleta/framework/message"
)

// Peer is a peer of the formulator
type Peer struct {
	id      string
	netAddr string
	conn    net.Conn
	pubhash common.PublicHash
}

// NewPeer returns a Peer
func NewPeer(conn net.Conn, pubhash common.PublicHash) *Peer {
	p := &Peer{
		id:      pubhash.String(),
		netAddr: conn.RemoteAddr().String(),
		conn:    conn,
		pubhash: pubhash,
	}
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
	errCh := make(chan error)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		wg.Done()
		_, err := p.conn.Write(bs)
		if err != nil {
			p.conn.Close()
		}
		errCh <- err
	}()
	wg.Wait()
	deadTimer := time.NewTimer(5 * time.Second)
	select {
	case <-deadTimer.C:
		return ErrPeerTimeout
	case err := <-errCh:
		return err
	}
}
