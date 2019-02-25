package formulator

import "errors"

// errors
var (
	ErrInvalidTimestamp     = errors.New("invalid timestamp")
	ErrNotAllowedPublicHash = errors.New("not allowed public hash")
	ErrInvalidRequest       = errors.New("invalid request")
	ErrUnknownPeer          = errors.New("unknown peer")
	ErrPeerTimeout          = errors.New("peer timeout")
)
