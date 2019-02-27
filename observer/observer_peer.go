package observer

import (
	"bytes"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/util"
	"git.fleta.io/fleta/framework/message"
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
}

// NewPeer returns a Peer
func NewPeer(conn net.Conn, pubhash common.PublicHash) *Peer {
	p := &Peer{
		id:        pubhash.String(),
		netAddr:   conn.RemoteAddr().String(),
		conn:      conn,
		pubhash:   pubhash,
		startTime: uint64(time.Now().UnixNano()),
	}
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

// Send sends a message to the peer
func (p *Peer) Read(bs []byte) (int, error) {
	n, err := p.conn.Read(bs)
	atomic.AddUint64(&p.readTotal, uint64(n))
	return n, err
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
		atomic.AddUint64(&p.writeTotal, uint64(len(bs)))
		errCh <- err
	}()
	wg.Wait()
	deadTimer := time.NewTimer(5 * time.Second)
	select {
	case <-deadTimer.C:
		p.conn.Close()
		return <-errCh
	case err := <-errCh:
		deadTimer.Stop()
		return err
	}
}
