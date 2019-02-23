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

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/common/util"
	"git.fleta.io/fleta/core/key"
	"git.fleta.io/fleta/core/message_def"
	"git.fleta.io/fleta/framework/chain/mesh"
	"git.fleta.io/fleta/framework/message"
)

// ObserverMeshDeligator deligates unhandled messages of the observer mesh
type ObserverMeshDeligator interface {
	OnRecv(p mesh.Peer, r io.Reader, t message.Type) error
}

type ObserverMesh struct {
	sync.Mutex
	Key            key.Key
	NetAddressMap  map[common.PublicHash]string
	clientPeerHash map[common.PublicHash]*Peer
	serverPeerHash map[common.PublicHash]*Peer
	deligator      ObserverMeshDeligator
	handler        mesh.EventHandler
}

func NewObserverMesh(Key key.Key, NetAddressMap map[common.PublicHash]string, Deligator ObserverMeshDeligator, handler mesh.EventHandler) *ObserverMesh {
	ms := &ObserverMesh{
		Key:            Key,
		NetAddressMap:  NetAddressMap,
		clientPeerHash: map[common.PublicHash]*Peer{},
		serverPeerHash: map[common.PublicHash]*Peer{},
		deligator:      Deligator,
		handler:        handler,
	}
	return ms
}

func (ms *ObserverMesh) Add(netAddr string, doForce bool) {
	log.Println("ObserverMesh", "Add", netAddr, doForce)
}
func (ms *ObserverMesh) Remove(netAddr string) {
	log.Println("ObserverMesh", "Remove", netAddr)
}
func (ms *ObserverMesh) RemoveByID(ID string) {
	log.Println("ObserverMesh", "RemoveByID", ID)
}
func (ms *ObserverMesh) Ban(netAddr string, Seconds uint32) {
	log.Println("ObserverMesh", "Ban", netAddr, Seconds)
}
func (ms *ObserverMesh) BanByID(ID string, Seconds uint32) {
	log.Println("ObserverMesh", "BanByID", ID, Seconds)
}
func (ms *ObserverMesh) Unban(netAddr string) {
	log.Println("ObserverMesh", "Unban", netAddr)
}
func (ms *ObserverMesh) Peers() []mesh.Peer {
	peerMap := map[common.PublicHash]*Peer{}
	ms.Lock()
	for _, p := range ms.clientPeerHash {
		peerMap[p.pubhash] = p
	}
	for _, p := range ms.serverPeerHash {
		peerMap[p.pubhash] = p
	}
	ms.Unlock()

	peers := []mesh.Peer{}
	for _, p := range peerMap {
		peers = append(peers, p)
	}
	return peers
}

func (ms *ObserverMesh) Run(BindAddress string) error {
	ObPubHash := common.NewPublicHash(ms.Key.PublicKey())
	for PubHash, v := range ms.NetAddressMap {
		if !PubHash.Equal(ObPubHash) {
			go func(pubhash common.PublicHash, NetAddr string) {
				time.Sleep(1 * time.Second)
				for {
					ms.Lock()
					_, hasC := ms.clientPeerHash[pubhash]
					_, hasS := ms.serverPeerHash[pubhash]
					ms.Unlock()
					if !hasC && !hasS {
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
		if err := ms.server(BindAddress); err != nil {
			log.Println("[server]", err)
		}
		time.Sleep(1 * time.Second)
	}
}

// RemovePeer removes peers from the mesh
func (ms *ObserverMesh) RemovePeer(p *Peer, peerHash map[common.PublicHash]*Peer) {
	ms.Lock()
	delete(peerHash, p.pubhash)
	ms.Unlock()

	p.conn.Close()
	ms.handler.OnDisconnected(p)
}

// BroadcastMessage sends a message to all peers
func (ms *ObserverMesh) BroadcastMessage(m message.Message) error {
	var buffer bytes.Buffer
	if _, err := util.WriteUint64(&buffer, uint64(m.Type())); err != nil {
		return err
	}
	if _, err := m.WriteTo(&buffer); err != nil {
		return err
	}
	data := buffer.Bytes()

	peerMap := map[common.PublicHash]*Peer{}
	targetMap := map[common.PublicHash]map[common.PublicHash]*Peer{}
	ms.Lock()
	for _, p := range ms.clientPeerHash {
		peerMap[p.pubhash] = p
		targetMap[p.pubhash] = ms.clientPeerHash
	}
	for _, p := range ms.serverPeerHash {
		peerMap[p.pubhash] = p
		targetMap[p.pubhash] = ms.serverPeerHash
	}
	ms.Unlock()

	for pubhash, p := range peerMap {
		if err := p.SendRaw(data); err != nil {
			log.Println(err)
			ms.RemovePeer(p, targetMap[pubhash])
		}
	}
	return nil
}

func (ms *ObserverMesh) client(Address string, TargetPubHash common.PublicHash) error {
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

	p := NewPeer(conn, pubhash)
	ms.Lock()
	old, has := ms.clientPeerHash[pubhash]
	ms.clientPeerHash[pubhash] = p
	ms.Unlock()
	if has {
		ms.RemovePeer(old, ms.clientPeerHash)
	}
	defer ms.RemovePeer(p, ms.clientPeerHash)

	if err := ms.handleConnection(p); err != nil {
		log.Println("[handleConnection]", err)
	}
	return nil
}

func (ms *ObserverMesh) server(BindAddress string) error {
	lstn, err := net.Listen("tcp", BindAddress)
	if err != nil {
		return err
	}
	log.Println(common.NewPublicHash(ms.Key.PublicKey()), "Start to Listen", BindAddress)
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
			if _, has := ms.NetAddressMap[pubhash]; !has {
				log.Println("ErrInvalidPublicHash")
				return
			}
			if err := ms.recvHandshake(conn); err != nil {
				log.Println("[recvHandshakeAck]", err)
				return
			}

			p := NewPeer(conn, pubhash)
			ms.Lock()
			old, has := ms.serverPeerHash[pubhash]
			ms.serverPeerHash[pubhash] = p
			ms.Unlock()
			if has {
				ms.RemovePeer(old, ms.serverPeerHash)
			}
			defer ms.RemovePeer(p, ms.serverPeerHash)

			if err := ms.handleConnection(p); err != nil {
				log.Println("[handleConnection]", err)
			}
		}()
	}
}

func (ms *ObserverMesh) handleConnection(p *Peer) error {
	log.Println(common.NewPublicHash(ms.Key.PublicKey()).String(), "Connected", p.pubhash.String())

	ms.handler.OnConnected(p)

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
		var t message.Type
		if v, _, err := util.ReadUint64(p.conn); err != nil {
			return err
		} else {
			t = message.Type(v)
		}
		atomic.SwapUint64(&pingCount, 0)
		if t == message_def.PingMessageType {
			// Because a PingMessage is zero size, so do not need to consume the body
			continue
		}

		if err := ms.deligator.OnRecv(p, p.conn, t); err != nil {
			return err
		}
	}
}

func (ms *ObserverMesh) recvHandshake(conn net.Conn) error {
	//log.Println("recvHandshake")
	req := make([]byte, 40)
	if _, err := util.FillBytes(conn, req); err != nil {
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

func (ms *ObserverMesh) sendHandshake(conn net.Conn) (common.PublicHash, error) {
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
