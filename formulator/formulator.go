package formulator

import (
	"bytes"
	"errors"
	"io"
	"sync"

	"git.fleta.io/fleta/framework/router"

	"git.fleta.io/fleta/core/data"
	"git.fleta.io/fleta/core/kernel"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/common/util"
	"git.fleta.io/fleta/core/block"
	"git.fleta.io/fleta/core/key"
	"git.fleta.io/fleta/core/message_def"
	"git.fleta.io/fleta/framework/chain"
	"git.fleta.io/fleta/framework/chain/mesh"
	"git.fleta.io/fleta/framework/message"
	"git.fleta.io/fleta/framework/peer"
)

// errors
var (
	ErrInvalidRequest = errors.New("invalid request")
	/*
		ErrInvalidTimestamp           = errors.New("invalid timestamp")
		ErrNotAllowedPublicHash       = errors.New("not allowed public hash")
		ErrInvalidModerator           = errors.New("invalid moderator")
		ErrInvalidRoundState          = errors.New("invalid round state")
		ErrInvalidRoundHash           = errors.New("invalid round hash")
		ErrInvalidVote                = errors.New("invalid vote")
		ErrInvalidVoteType            = errors.New("invalid vote type")
		ErrInvalidVoteSignature       = errors.New("invalid vote signature")
		ErrInvalidFormulatorSignature = errors.New("invalid formulator signature")
		ErrInvalidBlockGen            = errors.New("invalid block gen")
		ErrInitialTimeout             = errors.New("initial timeout")
		ErrDuplicatedVote             = errors.New("duplicated vote")
		ErrDuplicatedAckAndTimeout    = errors.New("duplicated akc and timeout")
		ErrAlreadyVoted               = errors.New("already voted")
	*/
)

type Formulator struct {
	sync.Mutex
	Config         *Config
	ObserverKeyMap map[common.PublicHash]bool
	ms             *FormulatorMesh
	cm             *chain.Manager
	kn             *kernel.Kernel
	pm             peer.Manager
	manager        *message.Manager
	lastGenMessage *message_def.BlockGenMessage
	lastReqMessage *message_def.BlockReqMessage
	lastContext    *data.Context
	isRunning      bool
}

type Config struct {
	ChainCoord    *common.Coordinate
	ObserverKeys  []string
	NetAddressMap map[common.PublicHash]string
	Key           key.Key
	Formulator    common.Address
	Router        router.Config
	Peer          peer.Config
}

func NewFormulator(Config *Config, kn *kernel.Kernel) (*Formulator, error) {
	ObserverKeyMap := map[common.PublicHash]bool{}
	for _, str := range Config.ObserverKeys {
		if pubhash, err := common.ParsePublicHash(str); err != nil {
			return nil, err
		} else {
			ObserverKeyMap[pubhash] = true
		}
	}

	r, err := router.NewRouter(&Config.Router)
	if err != nil {
		return nil, err
	}

	pm, err := peer.NewManager(kn.ChainCoord(), r, &Config.Peer)
	if err != nil {
		return nil, err
	}

	fr := &Formulator{
		Config:         Config,
		ObserverKeyMap: ObserverKeyMap,
		cm:             chain.NewManager(kn),
		pm:             pm,
		kn:             kn,
		manager:        message.NewManager(),
	}
	fr.manager.SetCreator(message_def.BlockReqMessageType, fr.messageCreator)
	fr.manager.SetCreator(message_def.BlockObSignMessageType, fr.messageCreator)
	fr.manager.SetCreator(chain.DataMessageType, fr.messageCreator)
	fr.manager.SetCreator(chain.StatusMessageType, fr.messageCreator)

	fr.ms = NewFormulatorMesh(Config.Key, Config.Formulator, Config.NetAddressMap, fr)
	fr.cm.Mesh = pm

	if err := fr.cm.Init(); err != nil {
		return nil, err
	}
	return fr, nil
}

func (fr *Formulator) Run() {
	fr.Lock()
	if fr.isRunning {
		fr.Unlock()
		return
	}
	fr.isRunning = true
	fr.Unlock()

	go fr.cm.Run()
	fr.ms.Run()
}

// OnRecv is called when a message is received from the peer
func (fr *Formulator) OnRecv(p mesh.Peer, r io.Reader, t message.Type) error {
	m, err := fr.manager.ParseMessage(r, t)
	if err != nil {
		return err
	}
	if err := fr.handleMessage(p, m); err != nil {
		//log.Println(err)
		return nil
	}
	return nil
}

func (fr *Formulator) handleMessage(p mesh.Peer, m message.Message) error {
	fr.Lock()
	defer fr.Unlock()

	switch msg := m.(type) {
	case *message_def.BlockReqMessage:
		//log.Println(fr.Config.Formulator, fr.kn.Provider().Height(), msg.TargetHeight, "BlockReqMessage")
		if msg.TargetHeight <= fr.kn.Provider().Height() {
			return nil
		}
		if fr.lastGenMessage != nil {
			if fr.lastGenMessage.RoundHash.Equal(msg.RoundHash) {
				if fr.lastGenMessage.Block.Header.TimeoutCount == msg.TimeoutCount {
					if err := p.Send(fr.lastGenMessage); err != nil {
						return err
					}
				}
				return nil
			}
		}

		NextRoundHash := fr.NextRoundHash()
		if !msg.RoundHash.Equal(NextRoundHash) {
			return ErrInvalidRequest
		}
		Top, err := fr.kn.TopRank(int(msg.TimeoutCount)) //TEMP
		if err != nil {
			return err
		}
		if !msg.Formulator.Equal(Top.Address) {
			return ErrInvalidRequest
		}
		if !msg.Formulator.Equal(fr.Config.Formulator) {
			return ErrInvalidRequest
		}
		if !msg.FormulatorPublicHash.Equal(common.NewPublicHash(fr.Config.Key.PublicKey())) {
			return ErrInvalidRequest
		}

		cp := fr.kn.Provider()
		if !msg.PrevHash.Equal(cp.PrevHash()) {
			return ErrInvalidRequest
		}
		if msg.TargetHeight != cp.Height()+1 {
			return ErrInvalidRequest
		}

		ctx, b, err := fr.kn.GenerateBlock(msg.TimeoutCount, fr.Config.Formulator)
		if err != nil {
			return err
		}

		nm := &message_def.BlockGenMessage{
			RoundHash: msg.RoundHash,
			Block:     b,
			Tran:      fr.kn.Transactor(),
		}

		if sig, err := fr.Config.Key.Sign(b.Header.Hash()); err != nil {
			return err
		} else {
			nm.GeneratorSignature = sig
		}

		if err := p.Send(nm); err != nil {
			return err
		}

		fr.lastGenMessage = nm
		fr.lastReqMessage = msg
		fr.lastContext = ctx

		return nil
	case *message_def.BlockObSignMessage:
		//log.Println(fr.Config.Formulator, fr.kn.Provider().Height(), "BlockObSignMessage")
		if fr.lastGenMessage == nil {
			return nil
		}
		if msg.TargetHeight <= fr.kn.Provider().Height() {
			return nil
		}
		if !msg.ObserverSigned.HeaderHash.Equal(fr.lastGenMessage.Block.Header.Hash()) {
			return ErrInvalidRequest
		}

		cd := &chain.Data{
			Header:     fr.lastGenMessage.Block.Header,
			Body:       fr.lastGenMessage.Block.Body,
			Signatures: append([]common.Signature{msg.ObserverSigned.GeneratorSignature}, msg.ObserverSigned.ObserverSignatures...),
		}
		if err := fr.cm.Process(cd, fr.lastContext); err != nil {
			return err
		}
		fr.cm.BroadcastHeader(cd.Header)
		return nil
	case *chain.DataMessage:
		//log.Println(fr.Config.Formulator, fr.kn.Provider().Height(), "chain.DataMessage")
		if msg.Data.Header.Height() <= fr.kn.Provider().Height() {
			return nil
		}
		if err := fr.cm.AddData(msg.Data); err != nil {
			return err
		}
		return nil
	case *chain.StatusMessage:
		//log.Println(fr.Config.Formulator, fr.kn.Provider().Height(), "chain.DataMessage")
		Height := fr.kn.Provider().Height()
		if Height < msg.Height && msg.Height <= Height+10 {
			Limit := msg.Height
			if Limit > Height+10 {
				Limit = Height + 10
			}
			for i := Height + 1; i <= Limit; i++ {
				sm := &chain.RequestMessage{
					Height: i,
				}
				if err := p.Send(sm); err != nil {
					return err
				}
			}
		}
		return nil
	default:
		return message.ErrUnhandledMessage
	}
}

// NextRoundHash provides next round hash value
func (fr *Formulator) NextRoundHash() hash.Hash256 {
	cp := fr.kn.Provider()
	var buffer bytes.Buffer
	if _, err := fr.kn.ChainCoord().WriteTo(&buffer); err != nil {
		panic(err)
	}
	buffer.WriteString(",")
	PrevHash := cp.PrevHash()
	if _, err := PrevHash.WriteTo(&buffer); err != nil {
		panic(err)
	}
	buffer.WriteString(",")
	if _, err := util.WriteUint32(&buffer, cp.Height()+1); err != nil {
		panic(err)
	}
	return hash.DoubleHash(buffer.Bytes())
}

func (fr *Formulator) messageCreator(r io.Reader, t message.Type) (message.Message, error) {
	switch t {
	case message_def.BlockReqMessageType:
		p := &message_def.BlockReqMessage{}
		if _, err := p.ReadFrom(r); err != nil {
			return nil, err
		}
		return p, nil
	case message_def.BlockObSignMessageType:
		p := &message_def.BlockObSignMessage{
			ObserverSigned: &block.ObserverSigned{},
		}
		if _, err := p.ReadFrom(r); err != nil {
			return nil, err
		}
		return p, nil
	case chain.DataMessageType:
		p := &chain.DataMessage{
			Data: &chain.Data{
				Header: fr.kn.Provider().CreateHeader(),
				Body:   fr.kn.Provider().CreateBody(),
			},
		}
		if _, err := p.ReadFrom(r); err != nil {
			return nil, err
		}
		return p, nil
	case chain.StatusMessageType:
		p := &chain.StatusMessage{}
		if _, err := p.ReadFrom(r); err != nil {
			return nil, err
		}
		return p, nil
	default:
		return nil, message.ErrUnknownMessage
	}
}
