package txpool

import "git.fleta.io/fleta/common"

// SeqCache TODO
type SeqCache interface {
	Seq(addr common.Address) uint64
}
