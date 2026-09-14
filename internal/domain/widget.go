package domain

import (
	"encoding/json"
	"time"
)

type WidgetType string

const (
	WidgetWeather  WidgetType = "weather"
	WidgetNews     WidgetType = "news"
	WidgetStock    WidgetType = "stock"
	WidgetCurrency WidgetType = "currency"
)

// WidgetTypes lists every supported type, in display order.
var WidgetTypes = []WidgetType{WidgetWeather, WidgetNews, WidgetStock, WidgetCurrency}

func (t WidgetType) String() string { return string(t) }

func (t WidgetType) Valid() bool {
	for _, known := range WidgetTypes {
		if t == known {
			return true
		}
	}
	return false
}

// ParseWidgetType validates a raw type string from a request.
func ParseWidgetType(s string) (WidgetType, error) {
	t := WidgetType(s)
	if !t.Valid() {
		return "", Validationf("unsupported widget type %q (must be one of: weather, news, stock, currency)", s)
	}
	return t, nil
}

// Widget is a stored widget configuration. Config is the normalized JSON
// for the widget's type (see the per-type config structs below).
type Widget struct {
	ID        int64           `json:"id"`
	UserID    int64           `json:"user_id"`
	Type      WidgetType      `json:"type"`
	Config    json.RawMessage `json:"config"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// Per-type config shapes. These are what gets validated and stored.

type WeatherConfig struct {
	City string `json:"city"`
}

type NewsConfig struct {
	Keyword string `json:"keyword"`
	Limit   int    `json:"limit"`
}

type StockConfig struct {
	Ticker string `json:"ticker"`
}

type CurrencyConfig struct {
	Base   string `json:"base"`
	Target string `json:"target"`
}
