package uploader

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"
)

const injestURL = "https://abyssal.space/action/ingest-autoupload"

func Upload(filename, token string) (name string, err error) {
	f, err := os.Open(filename)
	if err != nil {
		return filename, err
	}

	defer f.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)

	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", injestURL, f)
	if err != nil {
		return filename, err
	}

	client := http.Client{
		Transport: &transport{userAgent: "abyssal.space blackbox recorder", userToken: token},
	}

	resp, err := client.Do(req)
	if err != nil {
		return filename, fmt.Errorf("failed uploading: %s, with error: %w", filename, err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return filename, fmt.Errorf("failed uploading: %s, status: %s", filename, resp.Status)
	}

	return filename, nil
}

type transport struct {
	userAgent string
	userToken string
	http.RoundTripper
}

func (t *transport) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set("User-Agent", t.userAgent)
	r.Header.Set("Authorization", "Bearer "+t.userToken)
	r.Header.Set("Content-Type", "application/abyss-run-record")

	return http.DefaultTransport.RoundTrip(r)
}
