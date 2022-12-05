package app

import "context"

type TokenProvider interface {
	GetActiveCharacterToken(ctx context.Context) string
}
