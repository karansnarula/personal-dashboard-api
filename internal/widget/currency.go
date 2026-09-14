package widget

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/karansnarula/personal-dashboard-api/internal/domain"
)

const (
	providerFrankfurter   = "Frankfurter"
	DefaultFrankfurterURL = "https://api.frankfurter.dev"
)

type CurrencyData struct {
	Base   string  `json:"base"`
	Target string  `json:"target"`
	Rate   float64 `json:"rate"`
	Date   string  `json:"date"`
}

// CurrencyClient talks to Frankfurter, which needs no API key.
type CurrencyClient struct {
	hc      *http.Client
	baseURL string
}

func NewCurrencyClient(hc *http.Client, baseURL string) *CurrencyClient {
	return &CurrencyClient{hc: hc, baseURL: baseURL}
}

func (c *CurrencyClient) Fetch(ctx context.Context, raw json.RawMessage) (any, error) {
	cfg, err := parseConfig[domain.CurrencyConfig](raw)
	if err != nil {
		return nil, err
	}

	q := url.Values{}
	q.Set("base", cfg.Base)
	q.Set("symbols", cfg.Target)

	var resp struct {
		Base  string             `json:"base"`
		Date  string             `json:"date"`
		Rates map[string]float64 `json:"rates"`
	}
	if err := getJSON(ctx, c.hc, providerFrankfurter, c.baseURL+"/v1/latest?"+q.Encode(), nil, &resp); err != nil {
		return nil, err
	}
	rate, ok := resp.Rates[cfg.Target]
	if !ok {
		return nil, fmt.Errorf("%s has no rate for %s/%s", providerFrankfurter, cfg.Base, cfg.Target)
	}
	return CurrencyData{Base: cfg.Base, Target: cfg.Target, Rate: rate, Date: resp.Date}, nil
}
