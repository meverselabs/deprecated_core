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
	"git.fleta.io/fleta/framework/chain"
	"git.fleta.io/fleta/framework/chain/mesh"
	"git.fleta.io/fleta/framework/message"
)

// FormulatorServiceDeligator deligates unhandled messages of the formulator mesh
type FormulatorServiceDeligator interface {
	OnFormulatorConnected(p *FormulatorPeer)
	OnRecv(p mesh.Peer, r io.Reader, t message.Type) error
}

// FormulatorService provides connectivity with formulators
type FormulatorService struct {
	sync.Mutex
	Key       key.Key
	peerHash  map[common.Address]*FormulatorPeer
	deligator FormulatorServiceDeligator
	kn        *kernel.Kernel
	manager   *message.Manager
}

// NewFormulatorService returns a FormulatorService
func NewFormulatorService(Key key.Key, kn *kernel.Kernel, Deligator FormulatorServiceDeligator) *FormulatorService {
	ms := &FormulatorService{
		Key:       Key,
		kn:        kn,
		peerHash:  map[common.Address]*FormulatorPeer{},
		deligator: Deligator,
		manager:   message.NewManager(),
	}
	ms.manager.SetCreator(chain.RequestMessageType, ms.messageCreator)
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

func (ms *FormulatorService) removePeer(p *FormulatorPeer) {
	delete(ms.peerHash, p.address)
	p.conn.Close()
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

			ms.Lock()
			p, has := ms.peerHash[Formulator]
			if !has {
				p = NewFormulatorPeer(conn, pubhash, Formulator)
				ms.peerHash[Formulator] = p

				defer func() {
					ms.Lock()
					defer ms.Unlock()

					ms.removePeer(p)
				}()
			}
			ms.Unlock()

			if !has {
				if err := ms.handleConnection(p); err != nil {
					log.Println("[handleConnection]", err)
				}
			}
		}()
	}
}

func (ms *FormulatorService) handleConnection(p *FormulatorPeer) error {
	log.Println("Observer", common.NewPublicHash(ms.Key.PublicKey()).String(), "Fromulator Connected", p.address.String())

	ms.deligator.OnFormulatorConnected(p)

	cp := ms.kn.Provider()
	if err := p.Send(&chain.StatusMessage{
		Version:  cp.Version(),
		Height:   cp.Height(),
		PrevHash: cp.PrevHash(),
	}); err != nil {
		return err
	}

	for {
		var t message.Type
		if v, _, err := util.ReadUint64(p.conn); err != nil {
			return err
		} else {
			t = message.Type(v)
		}

		m, err := ms.manager.ParseMessage(p.conn, t)
		if err != nil {
			if err != message.ErrUnknownMessage {
				return err
			}
			if err := ms.deligator.OnRecv(p, p.conn, t); err != nil {
				return err
			}
		}

		if msg, is := m.(*chain.RequestMessage); is {
			cp := ms.kn.Provider()
			cd, err := cp.Data(msg.Height)
			if err != nil {
				return err
			}
			sm := &chain.DataMessage{
				Data: cd,
			}
			if err := p.Send(sm); err != nil {
				return err
			}
		}
	}
}

func (ms *FormulatorService) recvHandshake(conn net.Conn) (common.Address, error) {
	//log.Println("recvHandshake")
	req := make([]byte, 60)
	if err := util.FillBytes(conn, req); err != nil {
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

// FormulatorMap returns a formulator list as a map
func (ms *FormulatorService) FormulatorMap() map[common.Address]bool {
	FormulatorMap := map[common.Address]bool{}
	for _, p := range ms.peerHash {
		FormulatorMap[p.address] = true
	}
	return FormulatorMap
}

// SendTo sends a message to the formulator
func (ms *FormulatorService) SendTo(Formulator common.Address, m message.Message) error {
	ms.Lock()
	defer ms.Unlock()

	p, has := ms.peerHash[Formulator]
	if !has {
		return ErrUnknownFormulator
	}

	if err := p.Send(m); err != nil {
		log.Println(err)
		ms.removePeer(p)
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
			ms.Lock()
			ms.removePeer(p)
			ms.Unlock()
		}
	}
	return nil
}

func (ms *FormulatorService) messageCreator(r io.Reader, t message.Type) (message.Message, error) {
	switch t {
	case chain.RequestMessageType:
		p := &chain.RequestMessage{}
		if _, err := p.ReadFrom(r); err != nil {
			return nil, err
		}
		return p, nil
	default:
		return nil, message.ErrUnknownMessage
	}
}
