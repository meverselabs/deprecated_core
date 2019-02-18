package observer

import (
	"net"
	"sync"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/util"
	"git.fleta.io/fleta/framework/message"
)

type ObserverPeer struct {
	sync.Mutex
	id      string
	netAddr string
	conn    net.Conn
	pubhash common.PublicHash
}

func NewObserverPeer(conn net.Conn, pubhash common.PublicHash) *ObserverPeer {
	p := &ObserverPeer{
		id:      pubhash.String(),
		netAddr: conn.RemoteAddr().String(),
		conn:    conn,
		pubhash: pubhash,
	}
	return p
}

func (p *ObserverPeer) ID() string {
	return p.id
}

func (p *ObserverPeer) NetAddr() string {
	return p.netAddr
}

func (p *ObserverPeer) Send(m message.Message) error {
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

func (p *ObserverPeer) SendRaw(bs []byte) error {
	p.Lock()
	defer p.Unlock()

	if _, err := p.conn.Write(bs); err != nil {
		return err
	}
	return nil
}