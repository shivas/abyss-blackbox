package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"golang.org/x/exp/slog"
)

const availabilityURL = "https://abyssal.space/api/private/fittings/abyss-tracker/available"
const listFittingsURL = "https://abyssal.space/api/private/fittings/abyss-tracker/fits"
const importFittingURL = "https://abyssal.space/api/private/fittings/abyss-tracker/fits/%s"

func NewTrackerFittingsProvider(client *http.Client) *trackerFittingsProvider {
	if client == nil {
		client = http.DefaultClient
	}

	return &trackerFittingsProvider{
		httpClient: client,
	}
}

var _ FittingsProvider = (*trackerFittingsProvider)(nil)

type trackerFittingsProvider struct {
	httpClient *http.Client
}

func (t *trackerFittingsProvider) Available(ctx context.Context) AvailabilityResult {
	result := AvailabilityResult{}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, availabilityURL, http.NoBody)
	if err != nil {
		result.Err = err
		return result
	}

	r, err := t.httpClient.Do(req)
	if err != nil {
		result.Err = err
		return result
	}

	slog.Debug("available status code", slog.Int("statuscode", r.StatusCode))

	switch r.StatusCode {
	case http.StatusNoContent:
		result.Available = true
	case http.StatusFailedDependency:
		result.Available = false
		result.Err = fmt.Errorf("abyss tracker token not linked to abyssal.space account")

	default:
		result.Err = fmt.Errorf("unknown reason, status code: %d", r.StatusCode)
	}

	defer r.Body.Close()

	return result
}

func (*trackerFittingsProvider) SourceName() string {
	return "abyss.eve-nt.uk"
}

func (t *trackerFittingsProvider) AvailableFittingIDs(ctx context.Context) []string {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, listFittingsURL, http.NoBody)
	if err != nil {
		return nil
	}

	r, err := t.httpClient.Do(req)
	if err != nil {
		return nil
	}

	defer r.Body.Close()

	fittingsResponse := Fits{}

	err = json.NewDecoder(r.Body).Decode(&fittingsResponse)
	if err != nil {
		return nil
	}

	if !fittingsResponse.Success {
		return nil
	}

	results := make([]string, 0, len(fittingsResponse.Items))
	for _, item := range fittingsResponse.Items {
		results = append(results, fmt.Sprintf("%d", item.ID))
	}

	return results
}

func (t *trackerFittingsProvider) GetFittingDetails(ctx context.Context, id string) (*Fit, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf(importFittingURL, id), http.NoBody)
	if err != nil {
		return nil, err
	}

	r, err := t.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer r.Body.Close()

	fittingsResponse := Fit{}

	err = json.NewDecoder(r.Body).Decode(&fittingsResponse)
	if err != nil {
		return nil, err
	}

	if !fittingsResponse.Success {
		return nil, fmt.Errorf("failed fetching details of fit")
	}

	return &fittingsResponse, nil
}