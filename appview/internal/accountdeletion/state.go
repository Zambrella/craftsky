package accountdeletion

import "errors"

type Status string

const (
	StatusIntent   Status = "intent"
	StatusActive   Status = "active"
	StatusRetrying Status = "retrying"
)

var (
	ErrPointOfNoReturn        = errors.New("account deletion is past the point of no return")
	ErrBoundOAuthUnauthorized = errors.New("bound OAuth session unauthorized")
)
