package uploader

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"
)

const ingestURL = "https://abyssal.space/action/ingest-autoupload"

func Upload(client *http.Client, filename string) (name string, err error) {
	f, err := os.Open(filename)
	if err != nil {
		return filename, err
	}

	defer f.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)

	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", ingestURL, f)
	if err != nil {
		return filename, err
	}

	req.Header.Set("Content-Type", "application/abyss-run-record")

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
