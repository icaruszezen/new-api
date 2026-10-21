//go:build windows

package service

import (
	"errors"
	"syscall"
)

// isPeerResetError reports a write that failed because the peer reset or
// closed the connection.
func isPeerResetError(err error) bool {
	return errors.Is(err, syscall.WSAECONNRESET) || errors.Is(err, syscall.WSAECONNABORTED)
}
