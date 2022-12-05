package client

import (
	"fmt"
	"net/http"

	"github.com/shivas/abyss-blackbox/internal/app"
	"github.com/shivas/abyss-blackbox/internal/version"
	"golang.org/x/exp/slog"
)

const agent = "abyssal.space blackbox recorder %s"

func New(provider app.TokenProvider) *http.Client {
	return &http.Client{
		Transport: &transport{
			tokenProvider: provider,
			userAgent:     fmt.Sprintf(agent, version.RecorderVersion),
		},
	}
}

type transport struct {
	userAgent string
	http.RoundTripper
	tokenProvider app.TokenProvider
}

func (t *transport) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set("User-Agent", t.userAgent)
	r.Header.Set("Authorization", "Bearer "+t.tokenProvider.GetActiveCharacterToken(r.Context()))

	slog.Debug("headers", slog.Any("headers", r.Header))

	return http.DefaultTransport.RoundTrip(r)
}
