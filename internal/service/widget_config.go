package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/karansnarula/personal-dashboard-api/internal/domain"
)

const (
	DefaultNewsLimit = 5
	MaxNewsLimit     = 20
)

var (
	tickerRe   = regexp.MustCompile(`^[A-Z0-9.\-]{1,12}$`)
	currencyRe = regexp.MustCompile(`^[A-Z]{3}$`)
)

// NormalizeConfig checks that raw is a well-formed config for widget type t
// and returns a normalized copy (trimmed, upper-cased, defaults applied)
// suitable for storage. Unknown fields are rejected.
func NormalizeConfig(t domain.WidgetType, raw json.RawMessage) (json.RawMessage, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return nil, domain.Validationf("config is required")
	}

	var normalized any
	switch t {
	case domain.WidgetWeather:
		var c domain.WeatherConfig
		if err := decodeStrict(raw, &c); err != nil {
			return nil, err
		}
		c.City = strings.TrimSpace(c.City)
		if c.City == "" {
			return nil, domain.Validationf("weather widget requires a non-empty \"city\"")
		}
		normalized = c

	case domain.WidgetNews:
		var c domain.NewsConfig
		if err := decodeStrict(raw, &c); err != nil {
			return nil, err
		}
		c.Keyword = strings.TrimSpace(c.Keyword)
		if c.Keyword == "" {
			return nil, domain.Validationf("news widget requires a non-empty \"keyword\"")
		}
		if c.Limit == 0 {
			c.Limit = DefaultNewsLimit
		}
		if c.Limit < 1 || c.Limit > MaxNewsLimit {
			return nil, domain.Validationf("news widget \"limit\" must be between 1 and %d", MaxNewsLimit)
		}
		normalized = c

	case domain.WidgetStock:
		var c domain.StockConfig
		if err := decodeStrict(raw, &c); err != nil {
			return nil, err
		}
		c.Ticker = strings.ToUpper(strings.TrimSpace(c.Ticker))
		if c.Ticker == "" {
			return nil, domain.Validationf("stock widget requires a non-empty \"ticker\"")
		}
		if !tickerRe.MatchString(c.Ticker) {
			return nil, domain.Validationf("stock widget \"ticker\" %q is not a valid symbol", c.Ticker)
		}
		normalized = c

	case domain.WidgetCurrency:
		var c domain.CurrencyConfig
		if err := decodeStrict(raw, &c); err != nil {
			return nil, err
		}
		c.Base = strings.ToUpper(strings.TrimSpace(c.Base))
		c.Target = strings.ToUpper(strings.TrimSpace(c.Target))
		if !currencyRe.MatchString(c.Base) || !currencyRe.MatchString(c.Target) {
			return nil, domain.Validationf("currency widget requires 3-letter \"base\" and \"target\" codes (e.g. USD, EUR)")
		}
		if c.Base == c.Target {
			return nil, domain.Validationf("currency widget \"base\" and \"target\" must differ")
		}
		normalized = c

	default:
		return nil, domain.Validationf("unsupported widget type %q", t)
	}

	out, err := json.Marshal(normalized)
	if err != nil {
		return nil, fmt.Errorf("marshal config: %w", err)
	}
	return out, nil
}

func decodeStrict(raw json.RawMessage, v any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return domain.Validationf("invalid config: %v", err)
	}
	return nil
}
