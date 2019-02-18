package observer

import (
	"net"
	"sync"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/util"
	"git.fleta.io/fleta/framework/message"
)

type FormulatorPeer struct {
	sync.Mutex
	id      string
	netAddr string
	conn    net.Conn
	pubhash common.PublicHash
	address common.Address
}

func NewFormulatorPeer(conn net.Conn, pubhash common.PublicHash, address common.Address) *FormulatorPeer {
	p := &FormulatorPeer{
		id:      pubhash.String(),
		netAddr: conn.RemoteAddr().String(),
		conn:    conn,
		pubhash: pubhash,
		address: address,
	}
	return p
}

func (p *FormulatorPeer) ID() string {
	return p.id
}

func (p *FormulatorPeer) NetAddr() string {
	return p.netAddr
}

func (p *FormulatorPeer) Send(m message.Message) error {
	p.Lock()
	defer p.Unlock()

	if _, err := util.WriteUint64(p.conn, uint64(m.Type())); err != nil {
		return err
	}
	if _, err := m.WriteTo(p.conn); err != nil {
		return err
	}
	return nil
}

func (p *FormulatorPeer) SendRaw(bs []byte) error {
	p.Lock()
	defer p.Unlock()

	if _, err := p.conn.Write(bs); err != nil {
		return err
	}
	return nil
}