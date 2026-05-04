package ai

import "errors"

var (
	ErrOutOfScope   = errors.New("out of scope")
	ErrQueryTimeout = errors.New("query timeout")
)
