package service

import (
	"encoding/json"
	"testing"

	"github.com/karansnarula/personal-dashboard-api/internal/domain"
)

func TestNormalizeConfig(t *testing.T) {
	cases := []struct {
		name string
		typ  domain.WidgetType
		raw  string
		want string // normalized JSON on success; empty means expect error
	}{
		{"weather ok trims", domain.WidgetWeather, `{"city":"  London "}`, `{"city":"London"}`},
		{"weather missing city", domain.WidgetWeather, `{"city":""}`, ""},
		{"weather unknown field", domain.WidgetWeather, `{"city":"Paris","zip":"75001"}`, ""},

		{"news ok applies default limit", domain.WidgetNews, `{"keyword":"golang"}`, `{"keyword":"golang","limit":5}`},
		{"news explicit limit", domain.WidgetNews, `{"keyword":"golang","limit":3}`, `{"keyword":"golang","limit":3}`},
		{"news missing keyword", domain.WidgetNews, `{"limit":3}`, ""},
		{"news limit too high", domain.WidgetNews, `{"keyword":"go","limit":99}`, ""},

		{"stock ok uppercases", domain.WidgetStock, `{"ticker":"aapl"}`, `{"ticker":"AAPL"}`},
		{"stock missing ticker", domain.WidgetStock, `{}`, ""},
		{"stock bad symbol", domain.WidgetStock, `{"ticker":"not a ticker"}`, ""},

		{"currency ok", domain.WidgetCurrency, `{"base":"usd","target":"eur"}`, `{"base":"USD","target":"EUR"}`},
		{"currency same pair", domain.WidgetCurrency, `{"base":"USD","target":"USD"}`, ""},
		{"currency bad code", domain.WidgetCurrency, `{"base":"DOLLARS","target":"EUR"}`, ""},

		{"unsupported type", domain.WidgetType("crypto"), `{"coin":"BTC"}`, ""},
		{"empty config", domain.WidgetWeather, ``, ""},
		{"null config", domain.WidgetWeather, `null`, ""},
		{"wrong json type", domain.WidgetWeather, `{"city":123}`, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := NormalizeConfig(tc.typ, json.RawMessage(tc.raw))
			if tc.want == "" {
				if err == nil {
					t.Fatalf("expected error, got %s", got)
				}
				if !domain.IsValidation(err) {
					t.Fatalf("expected ValidationError, got %T: %v", err, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(got) != tc.want {
				t.Fatalf("got %s, want %s", got, tc.want)
			}
		})
	}
}
