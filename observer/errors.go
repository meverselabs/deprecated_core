package observer

import "errors"

// observer errors
var (
	ErrRequiredObserverAddresses = errors.New("required observer addresses")
	ErrRequiredRequestTimeout    = errors.New("required request timeout")
	ErrNotConnected              = errors.New("not connected")
	ErrRequestTimeout            = errors.New("request timeout")
	ErrAlreadyRequested          = errors.New("already requested")
	ErrInvalidResponse           = errors.New("invalid response")
)
