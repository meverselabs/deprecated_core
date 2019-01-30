package observer

import (
	"encoding/hex"
	"net"
	"sync"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/core/block"
	"git.fleta.io/fleta/core/data"
	"git.fleta.io/fleta/core/key"
	"git.fleta.io/fleta/framework/chain"
	"git.fleta.io/fleta/framework/message"
)

// Connector TODO
type Connector struct {
	Config            *Config
	mm                *message.Manager
	tran              *data.Transactor
	observerConnHash  map[int]net.Conn
	waitBlockHash     map[hash.Hash256]chan *block.ObserverSigned
	RequestTimeout    int
	connHashLock      sync.Mutex
	waitBlockHashLock sync.Mutex
}

/*
// NewConnector TODO
func NewConnector(Config *Config, RequestTimeout int, tran *data.Transactor) (*Connector, error) {
	if len(Config.Addresses) == 0 {
		return nil, ErrRequiredObserverAddresses
	}
	if RequestTimeout == 0 {
		return nil, ErrRequiredRequestTimeout
	}
	mm := message.NewManager()
	ob := &Connector{
		Config:           Config,
		mm:               mm,
		tran:             tran,
		RequestTimeout:   RequestTimeout,
		observerConnHash: map[int]net.Conn{},
		waitBlockHash:    map[hash.Hash256]chan *block.ObserverSigned{},
	}
	mm.ApplyMessage(message_def.BlockObSignMessageType, ob.messageCreator, ob.blockObSignMessageHandler)
	return ob, nil
}

// Start TODO
func (ob *Connector) Start() {
	go ob.run()
}

func (ob *Connector) run() {
	for i, addr := range ob.Config.Addresses {
		go func(i int, addr string) {
		DialLoop:
			for {
				Conn, err := network.Dial(ob.Config.Network, addr)
				if err != nil {
					log.Println(err)
					time.Sleep(10 * time.Second) //TEMP
					continue DialLoop
				}
				ob.connHashLock.Lock()
				if ob.observerConnHash[i] != nil {
					ob.observerConnHash[i].Close()
				}
				ob.observerConnHash[i] = Conn
				ob.connHashLock.Unlock()

			ConnLoop:
				for {
					closeFunc := func() {
						ob.connHashLock.Lock()
						Conn.Close()
						delete(ob.observerConnHash, i)
						ob.connHashLock.Unlock()
					}
					if v, _, err := util.ReadUint64(Conn); err != nil {
						closeFunc()
						log.Println(err)
						break ConnLoop
					} else {
						m, handler, err := ob.mm.ParseMessage(Conn, message.Type(v))
						if err != nil {
							closeFunc()
							log.Println(err)
							break ConnLoop
						}
						if err := handler(m); err != nil {
							closeFunc()
							log.Println(err)
							break ConnLoop
						}
						// TODO
					}
				}
			}
		}(i, addr)
	}
}

// AddMessageHandler TODO
func (ob *Connector) AddMessageHandler(t message.Type, creator message.Creator, handler message.Handler) error {
	return ob.mm.ApplyMessage(t, creator, handler)
}

// RequestSign TODO
func (ob *Connector) RequestSign(b *block.Block, s *block.Signed) (*block.ObserverSigned, error) {
	msg := &message_def.BlockGenMessage{
		Block:  b,
		Signed: s,
		Tran:   ob.tran,
	}
	var buffer bytes.Buffer
	if _, err := util.WriteUint64(&buffer, uint64(msg.Type())); err != nil {
		return nil, err
	}
	if _, err := msg.WriteTo(&buffer); err != nil {
		return nil, err
	}
	ob.connHashLock.Lock()
	var Conn net.Conn
	for _, c := range ob.observerConnHash {
		Conn = c
		break
	}
	ob.connHashLock.Unlock()
	if Conn != nil {
		h := b.Header.Hash()
		ob.waitBlockHashLock.Lock()
		if _, has := ob.waitBlockHash[h]; has {
			ob.waitBlockHashLock.Unlock()
			return nil, ErrAlreadyRequested
		} else {
			if _, err := Conn.Write(buffer.Bytes()); err != nil {
				return nil, err
			}
			ch := make(chan *block.ObserverSigned)
			ob.waitBlockHash[h] = ch
			ob.waitBlockHashLock.Unlock()
			timer := time.NewTimer(time.Duration(ob.RequestTimeout) * time.Second)
			select {
			case <-timer.C:
				return nil, ErrRequestTimeout
			case nos := <-ch:
				return nos, nil
			}
		}
	} else {
		return nil, ErrNotConnected
	}
}

func (ob *Connector) messageCreator(r io.Reader, mt message.Type) (message.Message, error) {
	switch mt {
	case message_def.BlockObSignMessageType:
		p := message_def.NewBlockObSignMessage()
		if _, err := p.ReadFrom(r); err != nil {
			return nil, err
		}
		return p, nil
	default:
		return nil, message.ErrUnknownMessage
	}
}

func (ob *Connector) blockObSignMessageHandler(m message.Message) error {
	msg := m.(*message_def.BlockObSignMessage)
	log.Println(unsafe.Pointer(ob), "[Oc] BlockObSignMessage", msg.ObserverSigned.BlockHash)
	ob.waitBlockHashLock.Lock()
	if ch, has := ob.waitBlockHash[msg.ObserverSigned.BlockHash]; !has {
		return ErrInvalidResponse
	} else {
		delete(ob.waitBlockHash, msg.ObserverSigned.BlockHash)
		ch <- msg.ObserverSigned
	}
	ob.waitBlockHashLock.Unlock()
	return nil
}
*/
// RequestSign TODO
func (ob *Connector) RequestSign(cd *chain.Data, s *block.Signed) (*block.ObserverSigned, error) {

	obstrs := []string{
		"cca49818f6c49cf57b6c420cdcd98fcae08850f56d2ff5b8d287fddc7f9ede08",
		"39f1a02bed5eff3f6247bb25564cdaef20d410d77ef7fc2c0181b1d5b31ce877",
		"2b97bc8f21215b7ed085cbbaa2ea020ded95463deef6cbf31bb1eadf826d4694",
		"3b43d728deaa62d7c8790636bdabbe7148a6641e291fd1f94b157673c0172425",
		"e6cf2724019000a3f703db92829ecbd646501c0fd6a5e97ad6774d4ad621f949",
	}

	obkeys := make([]key.Key, 0, len(obstrs))
	ObserverKeys := make([]string, 0, len(obstrs))
	for _, v := range obstrs {
		if bs, err := hex.DecodeString(v); err != nil {
			panic(err)
		} else if Key, err := key.NewMemoryKeyFromBytes(bs); err != nil {
			panic(err)
		} else {
			obkeys = append(obkeys, Key)
			ObserverKeys = append(ObserverKeys, common.NewPublicHash(Key.PublicKey()).String())
		}
	}

	ns := &block.ObserverSigned{
		Signed:             *s,
		ObserverSignatures: []common.Signature{},
	}

	SignedHash := s.Hash()
	for i := 0; i < len(ObserverKeys)/2+1; i++ {
		Key := obkeys[i]
		if sig, err := Key.Sign(SignedHash); err != nil {
			panic(err)
		} else {
			ns.ObserverSignatures = append(ns.ObserverSignatures, sig)
		}
	}
	return ns, nil
}
