package domain

import (
	"context"
	"syscall"
)

type TokenProvider interface {
	GetActiveCharacterToken(ctx context.Context) string
}

type ServerProvider interface {
	IsTestingServer(handle syscall.Handle) bool
}
