package domain

import "context"

type TokenProvider interface {
	GetActiveCharacterToken(ctx context.Context) string
}
