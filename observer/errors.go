package observer

import "errors"

// errors
var (
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
	ErrUnknownFormulator          = errors.New("unknown formulator")
	ErrUnknownObserver            = errors.New("unknown observer")
	ErrDuplicatedVote             = errors.New("duplicated vote")
	ErrDuplicatedAckAndTimeout    = errors.New("duplicated akc and timeout")
	ErrNoFormulatorConnected      = errors.New("no formulator connected")
	ErrAlreadyVoted               = errors.New("already voted")
)
