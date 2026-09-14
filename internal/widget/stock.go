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
	providerFinnhub   = "Finnhub"
	DefaultFinnhubURL = "https://finnhub.io"
)

type StockData struct {
	Ticker        string  `json:"ticker"`
	Price         float64 `json:"price"`
	Change        float64 `json:"change"`
	PercentChange float64 `json:"percent_change"`
	PreviousClose float64 `json:"previous_close"`
}

type StockClient struct {
	hc      *http.Client
	apiKey  string
	baseURL string
}

func NewStockClient(hc *http.Client, apiKey, baseURL string) *StockClient {
	return &StockClient{hc: hc, apiKey: apiKey, baseURL: baseURL}
}

func (c *StockClient) Fetch(ctx context.Context, raw json.RawMessage) (any, error) {
	if c.apiKey == "" {
		return nil, notConfigured(providerFinnhub)
	}
	cfg, err := parseConfig[domain.StockConfig](raw)
	if err != nil {
		return nil, err
	}

	q := url.Values{}
	q.Set("symbol", cfg.Ticker)

	// Finnhub quote fields: c=current, d=change, dp=percent change,
	// pc=previous close, t=timestamp. Unknown symbols come back all-zero.
	var resp struct {
		Current       float64 `json:"c"`
		Change        float64 `json:"d"`
		PercentChange float64 `json:"dp"`
		PreviousClose float64 `json:"pc"`
		Timestamp     int64   `json:"t"`
	}
	headers := map[string]string{"X-Finnhub-Token": c.apiKey}
	if err := getJSON(ctx, c.hc, providerFinnhub, c.baseURL+"/api/v1/quote?"+q.Encode(), headers, &resp); err != nil {
		return nil, err
	}
	if resp.Timestamp == 0 && resp.Current == 0 {
		return nil, fmt.Errorf("%s returned no quote for %q (unknown symbol?)", providerFinnhub, cfg.Ticker)
	}

	return StockData{
		Ticker:        cfg.Ticker,
		Price:         resp.Current,
		Change:        resp.Change,
		PercentChange: resp.PercentChange,
		PreviousClose: resp.PreviousClose,
	}, nil
}
