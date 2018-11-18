package observer

import (
	"bytes"
	"io"
	"log"
	"net"
	"sync"
	"time"
	"unsafe"

	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/common/util"
	"git.fleta.io/fleta/core/block"
	"git.fleta.io/fleta/core/data"
	"git.fleta.io/fleta/core/message_def"
	"git.fleta.io/fleta/framework/message"
	"git.fleta.io/fleta/mocknet"
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
	mm.ApplyMessage(message_def.BlockObSignMessageType, ob.blockObSignMessageCreator, ob.blockObSignMessageHandler)
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
				Conn, err := mocknet.Dial(ob.Config.Network, addr)
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
func (ob *Connector) AddMessageHandler(t message.Type, creator func(r io.Reader) message.Message, handler func(m message.Message) error) error {
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

func (ob *Connector) blockObSignMessageCreator(r io.Reader) message.Message {
	p := message_def.NewBlockObSignMessage()
	p.ReadFrom(r)
	return p
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
