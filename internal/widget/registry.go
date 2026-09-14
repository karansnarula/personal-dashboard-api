package widget

import (
	"net/http"
	"time"

	"github.com/karansnarula/personal-dashboard-api/internal/domain"
)

// Config carries API keys and optional base-URL overrides (used by tests to
// point clients at httptest servers).
type Config struct {
	OpenWeatherMapKey string
	NewsAPIKey        string
	FinnhubKey        string

	OpenWeatherMapURL string
	NewsAPIURL        string
	FinnhubURL        string
	FrankfurterURL    string

	// HTTPClient is shared by all clients. Defaults to a 10s-timeout client;
	// per-call deadlines are applied by the caller via ctx.
	HTTPClient *http.Client
}

// Registry maps each widget type to the client that serves it.
type Registry map[domain.WidgetType]Client

func NewRegistry(cfg Config) Registry {
	hc := cfg.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 10 * time.Second}
	}
	return Registry{
		domain.WidgetWeather:  NewWeatherClient(hc, cfg.OpenWeatherMapKey, orDefault(cfg.OpenWeatherMapURL, DefaultOpenWeatherMapURL)),
		domain.WidgetNews:     NewNewsClient(hc, cfg.NewsAPIKey, orDefault(cfg.NewsAPIURL, DefaultNewsAPIURL)),
		domain.WidgetStock:    NewStockClient(hc, cfg.FinnhubKey, orDefault(cfg.FinnhubURL, DefaultFinnhubURL)),
		domain.WidgetCurrency: NewCurrencyClient(hc, orDefault(cfg.FrankfurterURL, DefaultFrankfurterURL)),
	}
}

func orDefault(v, fallback string) string {
	if v != "" {
		return v
	}
	return fallback
}
