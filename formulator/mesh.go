package formulator

import (
	"bytes"
	crand "crypto/rand"
	"encoding/binary"
	"log"
	"net"
	"sync"
	"time"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/common/util"
	"git.fleta.io/fleta/core/key"
	"git.fleta.io/fleta/framework/chain/mesh"
	"git.fleta.io/fleta/framework/message"
)

type FormulatorMesh struct {
	sync.Mutex
	Key           key.Key
	Formulator    common.Address
	NetAddressMap map[common.PublicHash]string
	peerHash      map[string]*FormulatorPeer
	handler       mesh.EventHandler
}

func NewFormulatorMesh(Key key.Key, Formulator common.Address, NetAddressMap map[common.PublicHash]string, handler mesh.EventHandler) *FormulatorMesh {
	ms := &FormulatorMesh{
		Key:           Key,
		Formulator:    Formulator,
		NetAddressMap: NetAddressMap,
		peerHash:      map[string]*FormulatorPeer{},
		handler:       handler,
	}
	return ms
}

func (ms *FormulatorMesh) Add(netAddr string, doForce bool) {
	log.Println("FormulatorMesh", "Add", netAddr, doForce)
}
func (ms *FormulatorMesh) Remove(netAddr string) {
	log.Println("FormulatorMesh", "Remove", netAddr)
}
func (ms *FormulatorMesh) RemoveByID(ID string) {
	log.Println("FormulatorMesh", "RemoveByID", ID)
}
func (ms *FormulatorMesh) Ban(netAddr string, Seconds uint32) {
	log.Println("FormulatorMesh", "Ban", netAddr, Seconds)
}
func (ms *FormulatorMesh) BanByID(ID string, Seconds uint32) {
	log.Println("FormulatorMesh", "BanByID", ID, Seconds)
}
func (ms *FormulatorMesh) Unban(netAddr string) {
	log.Println("FormulatorMesh", "Unban", netAddr)
}
func (ms *FormulatorMesh) Peers() []mesh.Peer {
	peers := []mesh.Peer{}
	for _, p := range ms.peerHash {
		peers = append(peers, p)
	}
	return peers
}

func (ms *FormulatorMesh) Run() error {
	ObPubHash := common.NewPublicHash(ms.Key.PublicKey())
	for PubHash, v := range ms.NetAddressMap {
		if !PubHash.Equal(ObPubHash) {
			go func(pubhash common.PublicHash, NetAddr string) {
				time.Sleep(1 * time.Second)
				for {
					ms.Lock()
					_, has := ms.peerHash[pubhash.String()]
					ms.Unlock()
					if !has {
						if err := ms.client(NetAddr, pubhash); err != nil {
							log.Println("[client]", err, NetAddr)
						}
					}
					time.Sleep(1 * time.Second)
				}
			}(PubHash, v)
		}
	}
	for {
		time.Sleep(time.Hour) //TEMP
	}
	return nil
}

func (ms *FormulatorMesh) removePeer(p *FormulatorPeer) {
	delete(ms.peerHash, p.ID())
	p.conn.Close()
}

func (ms *FormulatorMesh) client(Address string, TargetPubHash common.PublicHash) error {
	conn, err := net.DialTimeout("tcp", Address, 10*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()

	if err := ms.recvHandshake(conn); err != nil {
		log.Println("[recvHandshake]", err)
		return err
	}
	pubhash, err := ms.sendHandshake(conn)
	if err != nil {
		log.Println("[sendHandshake]", err)
		return err
	}
	if !pubhash.Equal(TargetPubHash) {
		return common.ErrInvalidPublicHash
	}
	if _, has := ms.NetAddressMap[pubhash]; !has {
		return ErrNotAllowedPublicHash
	}

	ms.Lock()
	p, has := ms.peerHash[pubhash.String()]
	if !has {
		p = NewFormulatorPeer(conn, pubhash)
		ms.peerHash[pubhash.String()] = p

		defer func() {
			ms.Lock()
			defer ms.Unlock()

			ms.removePeer(p)
			ms.handler.OnClosed(p)
		}()
	}
	ms.Unlock()

	if !has {
		if err := ms.handler.BeforeConnect(p); err != nil {
			return err
		}
		if err := ms.handleConnection(p); err != nil {
			return err
		}
	}
	return nil
}

func (ms *FormulatorMesh) handleConnection(p *FormulatorPeer) error {
	log.Println(common.NewPublicHash(ms.Key.PublicKey()).String(), "Connected", p.pubhash.String())

	ms.handler.AfterConnect(p)

	for {
		var t message.Type
		if v, _, err := util.ReadUint64(p.conn); err != nil {
			return err
		} else {
			t = message.Type(v)
		}

		if err := ms.handler.OnRecv(p, p.conn, t); err != nil {
			return err
		}
	}
}

func (ms *FormulatorMesh) recvHandshake(conn net.Conn) error {
	//log.Println("recvHandshake")
	req := make([]byte, 40)
	if err := util.FillBytes(conn, req); err != nil {
		return err
	}
	timestamp := binary.LittleEndian.Uint64(req[32:])
	diff := time.Duration(uint64(time.Now().UnixNano()) - timestamp)
	if diff < 0 {
		diff = -diff
	}
	if diff > time.Second*30 {
		return ErrInvalidTimestamp
	}
	//log.Println("sendHandshakeAck")
	h := hash.Hash(req)
	if sig, err := ms.Key.Sign(h); err != nil {
		return err
	} else if _, err := conn.Write(sig[:]); err != nil {
		return err
	}
	return nil
}

func (ms *FormulatorMesh) sendHandshake(conn net.Conn) (common.PublicHash, error) {
	//log.Println("sendHandshake")
	req := make([]byte, 60)
	if _, err := crand.Read(req[:32]); err != nil {
		return common.PublicHash{}, err
	}
	copy(req[32:], ms.Formulator[:])
	binary.LittleEndian.PutUint64(req[52:], uint64(time.Now().UnixNano()))
	if _, err := conn.Write(req); err != nil {
		return common.PublicHash{}, err
	}
	//log.Println("recvHandshakeAsk")
	h := hash.Hash(req)
	var sig common.Signature
	if err := util.FillBytes(conn, sig[:]); err != nil {
		return common.PublicHash{}, err
	}
	pubkey, err := common.RecoverPubkey(h, sig)
	if err != nil {
		return common.PublicHash{}, err
	}
	pubhash := common.NewPublicHash(pubkey)
	return pubhash, nil
}

// SendTo sends a message to the target peer
func (ms *FormulatorMesh) SendTo(id string, m message.Message) error {
	ms.Lock()
	p, has := ms.peerHash[id]
	ms.Unlock()
	if !has {
		return ErrUnknownPeer
	}

	if err := p.Send(m); err != nil {
		ms.Lock()
		ms.removePeer(p)
		ms.Unlock()
		return err
	}
	return nil
}

func (ms *FormulatorMesh) BroadcastMessage(m message.Message) error {
	var buffer bytes.Buffer
	if _, err := util.WriteUint64(&buffer, uint64(m.Type())); err != nil {
		return err
	}
	if _, err := m.WriteTo(&buffer); err != nil {
		return err
	}
	data := buffer.Bytes()

	peers := []*FormulatorPeer{}
	ms.Lock()
	for _, p := range ms.peerHash {
		peers = append(peers, p)
	}
	ms.Unlock()
	for _, p := range peers {
		if err := p.SendRaw(data); err != nil {
			log.Println(err)
			ms.Lock()
			ms.removePeer(p)
			ms.Unlock()
		}
	}
	return nil
}

type FormulatorPeer struct {
	sync.Mutex
	id      string
	netAddr string
	conn    net.Conn
	pubhash common.PublicHash
}

func NewFormulatorPeer(conn net.Conn, pubhash common.PublicHash) *FormulatorPeer {
	p := &FormulatorPeer{
		id:      pubhash.String(),
		netAddr: conn.RemoteAddr().String(),
		conn:    conn,
		pubhash: pubhash,
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
