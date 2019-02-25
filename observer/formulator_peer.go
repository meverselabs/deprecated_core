package observer

import (
	"bytes"
	"net"
	"sync"
	"time"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/util"
	"git.fleta.io/fleta/framework/message"
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
}

// NewFormulatorPeer returns a ormulatorPeer
func NewFormulatorPeer(conn net.Conn, pubhash common.PublicHash, address common.Address) *FormulatorPeer {
	p := &FormulatorPeer{
		id:      address.String(),
		netAddr: conn.RemoteAddr().String(),
		conn:    conn,
		pubhash: pubhash,
		address: address,
	}
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
		p.conn.Close()
		return ErrPeerTimeout
	case err := <-errCh:
		return err
	}
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
