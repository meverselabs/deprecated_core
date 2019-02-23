package observer

import (
	"bytes"
	"net"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/util"
	"git.fleta.io/fleta/framework/message"
)

// FormulatorPeer is a formulator peer
type FormulatorPeer struct {
	id      string
	netAddr string
	conn    net.Conn
	pubhash common.PublicHash
	address common.Address
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

// Send sends a message to the peer
func (p *FormulatorPeer) Send(m message.Message) error {
	var buffer bytes.Buffer
	if _, err := util.WriteUint64(&buffer, uint64(m.Type())); err != nil {
		return err
	}
	if _, err := m.WriteTo(&buffer); err != nil {
		return err
	}
	if _, err := p.conn.Write(buffer.Bytes()); err != nil {
		return err
	}
	return nil
}

// SendRaw sends bytes to the peer
func (p *FormulatorPeer) SendRaw(bs []byte) error {
	if _, err := p.conn.Write(bs); err != nil {
		return err
	}
	return nil
}
