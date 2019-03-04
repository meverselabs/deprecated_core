package observer

import (
	"bytes"
	crand "crypto/rand"
	"encoding/binary"
	"io"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fletaio/common"
	"github.com/fletaio/common/hash"
	"github.com/fletaio/common/util"
	"github.com/fletaio/core/kernel"
	"github.com/fletaio/core/key"
	"github.com/fletaio/core/message_def"
	"github.com/fletaio/framework/chain/mesh"
	"github.com/fletaio/framework/message"
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
	Key     key.Key
	peerMap map[common.Address]*FormulatorPeer
	handler FormulatorServiceEventHandler
	kn      *kernel.Kernel
	manager *message.Manager
}

// NewFormulatorService returns a FormulatorService
func NewFormulatorService(Key key.Key, kn *kernel.Kernel, handler FormulatorServiceEventHandler) *FormulatorService {
	ms := &FormulatorService{
		Key:     Key,
		kn:      kn,
		peerMap: map[common.Address]*FormulatorPeer{},
		handler: handler,
		manager: message.NewManager(),
	}
	return ms
}

// Run provides a server
func (ms *FormulatorService) Run(BindAddress string) {
	if err := ms.server(BindAddress); err != nil {
		panic(err)
	}
}

// PeerCount returns a number of the peer
func (ms *FormulatorService) PeerCount() int {
	ms.Lock()
	defer ms.Unlock()

	return len(ms.peerMap)
}

// RemovePeer removes peers from the mesh
func (ms *FormulatorService) RemovePeer(addr common.Address) {
	ms.Lock()
	p, has := ms.peerMap[addr]
	if has {
		delete(ms.peerMap, addr)
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
	p, has := ms.peerMap[Formulator]
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
	for _, p := range ms.peerMap {
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

// GuessHeightCountMap returns a number of the guess height from all peers
func (ms *FormulatorService) GuessHeightCountMap() map[uint32]int {
	CountMap := map[uint32]int{}
	ms.Lock()
	for _, p := range ms.peerMap {
		CountMap[p.GuessHeight()]++
	}
	ms.Unlock()
	return CountMap
}

// UpdateGuessHeight updates the guess height of the fomrulator
func (ms *FormulatorService) UpdateGuessHeight(Formulator common.Address, height uint32) {
	ms.Lock()
	p, has := ms.peerMap[Formulator]
	ms.Unlock()
	if has {
		p.UpdateGuessHeight(height)
	}
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
				log.Println("[IsFormulator]", Formulator.String(), pubhash.String())
				return
			}

			p := NewFormulatorPeer(conn, pubhash, Formulator)
			ms.Lock()
			old, has := ms.peerMap[Formulator]
			ms.peerMap[Formulator] = p
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

	var pingCount uint64
	pingCountLimit := uint64(3)
	pingTimer := time.NewTimer(10 * time.Second)
	go func() {
		for {
			select {
			case <-pingTimer.C:
				if err := p.Send(&message_def.PingMessage{}); err != nil {
					p.conn.Close()
					return
				}
				if atomic.AddUint64(&pingCount, 1) > pingCountLimit {
					p.conn.Close()
					return
				}
			}
		}
	}()
	for {
		t, bs, err := p.ReadMessageData()
		if err != nil {
			return err
		}
		atomic.SwapUint64(&pingCount, 0)
		if bs == nil {
			// Because a Message is zero size, so do not need to consume the body
			continue
		}

		if err := ms.handler.OnRecv(p, bytes.NewReader(bs), t); err != nil {
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
	for _, p := range ms.peerMap {
		FormulatorMap[p.address] = true
	}
	return FormulatorMap
}
