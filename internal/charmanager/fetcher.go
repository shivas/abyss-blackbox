package charmanager

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

type backgroundTokenFetcher struct {
	sessionID string
	running   bool
	ctx       context.Context
	cancel    context.CancelFunc
}

func (f *backgroundTokenFetcher) run(ctx context.Context, sessionID string, callback func(string)) {
	if f.running {
		return
	}

	f.sessionID = sessionID
	f.ctx, f.cancel = context.WithTimeout(ctx, time.Minute*10)

	go func() {
		ticker := time.NewTicker(time.Second * 5)

		defer func() {
			f.cancel()
			f.running = false

			ticker.Stop()
		}()

		for {
			select {
			case <-f.ctx.Done():
				f.running = false

				callback("")

				return
			case t := <-ticker.C:
				log.Printf("attempting to fetch token: %v", t)

				resp, err := http.DefaultClient.Get("https://abyssal.space/auth/token/" + f.sessionID)
				if err != nil {
					log.Printf("error fetching token: %v", err)
				}

				if resp.StatusCode == http.StatusNoContent {
					log.Printf("no contents for token")
					resp.Body.Close()

					continue
				}

				result := struct {
					Token string `json:"token"`
				}{}

				err = json.NewDecoder(resp.Body).Decode(&result)
				if err != nil {
					log.Printf("error unmarshaling token: %v", err)
					resp.Body.Close()

					continue
				}

				resp.Body.Close()

				callback(result.Token)

				return
			}
		}
	}()
}
