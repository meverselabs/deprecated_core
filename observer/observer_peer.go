package observer

import (
	"bytes"
	"net"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/util"
	"git.fleta.io/fleta/framework/message"
)

type Peer struct {
	id      string
	netAddr string
	conn    net.Conn
	pubhash common.PublicHash
}

func NewPeer(conn net.Conn, pubhash common.PublicHash) *Peer {
	p := &Peer{
		id:      pubhash.String(),
		netAddr: conn.RemoteAddr().String(),
		conn:    conn,
		pubhash: pubhash,
	}
	return p
}

func (p *Peer) ID() string {
	return p.id
}

func (p *Peer) NetAddr() string {
	return p.netAddr
}

func (p *Peer) Send(m message.Message) error {
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

func (p *Peer) SendRaw(bs []byte) error {
	if _, err := p.conn.Write(bs); err != nil {
		return err
	}
	return nil
}
