package observer

import (
	"bytes"
	crand "crypto/rand"
	"encoding/binary"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/common/util"
	"git.fleta.io/fleta/core/kernel"
	"git.fleta.io/fleta/core/key"
	"git.fleta.io/fleta/core/message_def"
	"git.fleta.io/fleta/framework/chain/mesh"
	"git.fleta.io/fleta/framework/message"
)

// FormulatorServiceEventHandler handles events that are ocurred from the formulator service
type FormulatorServiceEventHandler interface {
	OnFormulatorConnected(p *FormulatorPeer)
	OnFormulatorDisconnected(p *FormulatorPeer)
	OnRecv(p mesh.Peer, r io.Reader, t message.Type) error
}

// FormulatorService provides connectivity with formulators
type FormulatorService struct {
	sync.Mutex
	Key      key.Key
	peerHash map[common.Address]*FormulatorPeer
	handler  FormulatorServiceEventHandler
	kn       *kernel.Kernel
	manager  *message.Manager
}

// NewFormulatorService returns a FormulatorService
func NewFormulatorService(Key key.Key, kn *kernel.Kernel, handler FormulatorServiceEventHandler) *FormulatorService {
	ms := &FormulatorService{
		Key:      Key,
		kn:       kn,
		peerHash: map[common.Address]*FormulatorPeer{},
		handler:  handler,
		manager:  message.NewManager(),
	}
	return ms
}

// Run provides a server
func (ms *FormulatorService) Run(BindAddress string) error {
	for {
		if err := ms.server(BindAddress); err != nil {
			log.Println("[server]", err)
		}
		time.Sleep(1 * time.Second)
	}
}

// PeerCount returns a number of the peer
func (ms *FormulatorService) PeerCount() int {
	ms.Lock()
	defer ms.Unlock()

	return len(ms.peerHash)
}

// RemovePeer removes peers from the mesh
func (ms *FormulatorService) RemovePeer(addr common.Address) {
	ms.Lock()
	p, has := ms.peerHash[addr]
	if has {
		delete(ms.peerHash, addr)
	}
	ms.Unlock()

	if has {
		p.conn.Close()
		ms.handler.OnFormulatorDisconnected(p)
	}
}

// SendTo sends a message to the formulator
func (ms *FormulatorService) SendTo(Formulator common.Address, m message.Message) error {
	ms.Lock()
	p, has := ms.peerHash[Formulator]
	ms.Unlock()
	if !has {
		return ErrUnknownFormulator
	}

	if err := p.Send(m); err != nil {
		log.Println(err)
		ms.RemovePeer(p.address)
	}
	return nil
}

// BroadcastMessage sends a message to all peers
func (ms *FormulatorService) BroadcastMessage(m message.Message) error {
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
			ms.RemovePeer(p.address)
		}
	}
	return nil
}

func (ms *FormulatorService) server(BindAddress string) error {
	lstn, err := net.Listen("tcp", BindAddress)
	if err != nil {
		return err
	}
	log.Println("Observer", common.NewPublicHash(ms.Key.PublicKey()), "Start to Listen", BindAddress)

	for {
		conn, err := lstn.Accept()
		if err != nil {
			return err
		}
		go func() {
			defer conn.Close()

			pubhash, err := ms.sendHandshake(conn)
			if err != nil {
				log.Println("[sendHandshake]", err)
				return
			}
			Formulator, err := ms.recvHandshake(conn)
			if err != nil {
				log.Println("[recvHandshakeAck]", err)
				return
			}
			if !ms.kn.IsFormulator(Formulator, pubhash) {
				log.Println("[IsFormulator]")
				return
			}

			p := NewFormulatorPeer(conn, pubhash, Formulator)
			ms.Lock()
			old, has := ms.peerHash[Formulator]
			ms.peerHash[Formulator] = p
			ms.Unlock()
			if has {
				ms.RemovePeer(old.address)
			}
			defer ms.RemovePeer(p.address)

			if err := ms.handleConnection(p); err != nil {
				log.Println("[handleConnection]", err)
			}
		}()
	}
}

func (ms *FormulatorService) handleConnection(p *FormulatorPeer) error {
	log.Println("Observer", common.NewPublicHash(ms.Key.PublicKey()).String(), "Fromulator Connected", p.address.String())

	ms.handler.OnFormulatorConnected(p)

	pingTimer := time.NewTimer(10 * time.Second)
	deadTimer := time.NewTimer(30 * time.Second)
	go func() {
		for {
			select {
			case <-pingTimer.C:
				if err := p.Send(&message_def.PingMessage{Timestamp: uint64(time.Now().UnixNano())}); err != nil {
					p.conn.Close()
					return
				}
			case <-deadTimer.C:
				p.conn.Close()
				return
			}
		}
	}()
	for {
		var t message.Type
		if v, _, err := util.ReadUint64(p.conn); err != nil {
			return err
		} else {
			t = message.Type(v)
		}
		deadTimer.Reset(30 * time.Second)
		if t == message_def.PingMessageType {
			continue
		}

		if err := ms.handler.OnRecv(p, p.conn, t); err != nil {
			return err
		}
	}
}

func (ms *FormulatorService) recvHandshake(conn net.Conn) (common.Address, error) {
	//log.Println("recvHandshake")
	req := make([]byte, 60)
	if _, err := util.FillBytes(conn, req); err != nil {
		return common.Address{}, err
	}
	var Formulator common.Address
	copy(Formulator[:], req[32:])
	timestamp := binary.LittleEndian.Uint64(req[52:])
	diff := time.Duration(uint64(time.Now().UnixNano()) - timestamp)
	if diff < 0 {
		diff = -diff
	}
	if diff > time.Second*30 {
		return common.Address{}, ErrInvalidTimestamp
	}
	//log.Println("sendHandshakeAck")
	h := hash.Hash(req)
	if sig, err := ms.Key.Sign(h); err != nil {
		return common.Address{}, err
	} else if _, err := conn.Write(sig[:]); err != nil {
		return common.Address{}, err
	}
	return Formulator, nil
}

func (ms *FormulatorService) sendHandshake(conn net.Conn) (common.PublicHash, error) {
	//log.Println("sendHandshake")
	req := make([]byte, 40)
	if _, err := crand.Read(req[:32]); err != nil {
		return common.PublicHash{}, err
	}
	binary.LittleEndian.PutUint64(req[32:], uint64(time.Now().UnixNano()))
	if _, err := conn.Write(req); err != nil {
		return common.PublicHash{}, err
	}
	//log.Println("recvHandshakeAsk")
	h := hash.Hash(req)
	var sig common.Signature
	if _, err := sig.ReadFrom(conn); err != nil {
		return common.PublicHash{}, err
	}
	pubkey, err := common.RecoverPubkey(h, sig)
	if err != nil {
		return common.PublicHash{}, err
	}
	pubhash := common.NewPublicHash(pubkey)
	return pubhash, nil
}

// FormulatorMap returns a formulator list as a map
func (ms *FormulatorService) FormulatorMap() map[common.Address]bool {
	ms.Lock()
	defer ms.Unlock()

	FormulatorMap := map[common.Address]bool{}
	for _, p := range ms.peerHash {
		FormulatorMap[p.address] = true
	}
	return FormulatorMap
}
