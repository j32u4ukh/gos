package base

type ConnErrorType int

const (
	ErrTypeUnknown ConnErrorType = iota
	ErrTypeTimeout
	ErrTypeEOF
	ErrTypeNetwork
	ErrTypeDeadline
)

type ConnError struct {
	Type ConnErrorType
	Err  error
}
