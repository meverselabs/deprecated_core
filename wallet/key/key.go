package key

import (
	"io"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
)

// Key TODO
type Key interface {
	io.ReaderFrom
	io.WriterTo
	IsLock() bool
	Sign(h hash.Hash256) (common.Signature, error)
	SignWithPassphrase(h hash.Hash256, passphrase []byte) (common.Signature, error)
	Verify(h hash.Hash256, sig common.Signature) bool
	PublicKey() common.PublicKey
}
