package widget

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/karansnarula/personal-dashboard-api/internal/domain"
)

const (
	providerOpenWeatherMap   = "OpenWeatherMap"
	DefaultOpenWeatherMapURL = "https://api.openweathermap.org"
)

type WeatherData struct {
	City        string  `json:"city"`
	TempC       float64 `json:"temp_c"`
	Condition   string  `json:"condition"`
	Description string  `json:"description"`
}

type WeatherClient struct {
	hc      *http.Client
	apiKey  string
	baseURL string
}

func NewWeatherClient(hc *http.Client, apiKey, baseURL string) *WeatherClient {
	return &WeatherClient{hc: hc, apiKey: apiKey, baseURL: baseURL}
}

func (c *WeatherClient) Fetch(ctx context.Context, raw json.RawMessage) (any, error) {
	if c.apiKey == "" {
		return nil, notConfigured(providerOpenWeatherMap)
	}
	cfg, err := parseConfig[domain.WeatherConfig](raw)
	if err != nil {
		return nil, err
	}

	q := url.Values{}
	q.Set("q", cfg.City)
	q.Set("appid", c.apiKey)
	q.Set("units", "metric")

	var resp struct {
		Name string `json:"name"`
		Main struct {
			Temp float64 `json:"temp"`
		} `json:"main"`
		Weather []struct {
			Main        string `json:"main"`
			Description string `json:"description"`
		} `json:"weather"`
	}
	if err := getJSON(ctx, c.hc, providerOpenWeatherMap, c.baseURL+"/data/2.5/weather?"+q.Encode(), nil, &resp); err != nil {
		return nil, err
	}

	data := WeatherData{City: resp.Name, TempC: resp.Main.Temp}
	if len(resp.Weather) > 0 {
		data.Condition = resp.Weather[0].Main
		data.Description = resp.Weather[0].Description
	}
	return data, nil
}
