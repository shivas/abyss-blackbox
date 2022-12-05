package provider

import "context"

type FittingsProvider interface {
	Available(ctx context.Context) AvailabilityResult
	SourceName() string
	AvailableFittingIDs(ctx context.Context) []string
	GetFittingDetails(ctx context.Context, ID string) (*Fit, error)
}

type Ship struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Size string `json:"size"`
}

type Fits struct {
	Success bool   `json:"success"`
	Items   []Item `json:"items"`
	Count   int64  `json:"count"`
}

type Item struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Ship Ship   `json:"ship"`
	EFT  string `json:"eft"`
	FFH  string `json:"flexibleFitHash"`
}

type Fit struct {
	Success bool `json:"success"`
	Item    Item `json:"item"`
}

type AvailabilityResult struct {
	Available bool
	Err       error
}
