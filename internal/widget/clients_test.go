package widget

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func stub(t *testing.T, status int, body string, check func(r *http.Request)) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if check != nil {
			check(r)
		}
		w.WriteHeader(status)
		io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestWeatherClient(t *testing.T) {
	srv := stub(t, 200, `{"name":"London","main":{"temp":17.5},"weather":[{"main":"Clouds","description":"broken clouds"}]}`,
		func(r *http.Request) {
			if r.URL.Path != "/data/2.5/weather" || r.URL.Query().Get("q") != "London" || r.URL.Query().Get("appid") != "k" || r.URL.Query().Get("units") != "metric" {
				t.Errorf("unexpected request: %s", r.URL)
			}
		})
	c := NewWeatherClient(srv.Client(), "k", srv.URL)

	got, err := c.Fetch(context.Background(), json.RawMessage(`{"city":"London"}`))
	if err != nil {
		t.Fatal(err)
	}
	want := WeatherData{City: "London", TempC: 17.5, Condition: "Clouds", Description: "broken clouds"}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestWeatherClient_MissingKey(t *testing.T) {
	c := NewWeatherClient(http.DefaultClient, "", "http://unused")
	_, err := c.Fetch(context.Background(), json.RawMessage(`{"city":"London"}`))
	if !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("got %v, want ErrNotConfigured", err)
	}
}

func TestNewsClient(t *testing.T) {
	srv := stub(t, 200, `{"status":"ok","articles":[{"title":"Go 1.26","url":"http://x","source":{"name":"Blog"},"publishedAt":"2026-01-01T00:00:00Z"}]}`,
		func(r *http.Request) {
			if r.Header.Get("X-Api-Key") != "k" || r.URL.Query().Get("q") != "golang" || r.URL.Query().Get("pageSize") != "3" {
				t.Errorf("unexpected request: %s %v", r.URL, r.Header)
			}
		})
	c := NewNewsClient(srv.Client(), "k", srv.URL)

	got, err := c.Fetch(context.Background(), json.RawMessage(`{"keyword":"golang","limit":3}`))
	if err != nil {
		t.Fatal(err)
	}
	data := got.(NewsData)
	if data.Keyword != "golang" || len(data.Articles) != 1 || data.Articles[0].Headline != "Go 1.26" || data.Articles[0].Source != "Blog" {
		t.Fatalf("unexpected data: %+v", data)
	}
}

func TestStockClient(t *testing.T) {
	t.Run("quote", func(t *testing.T) {
		srv := stub(t, 200, `{"c":190.5,"d":1.5,"dp":0.79,"pc":189,"t":1700000000}`, func(r *http.Request) {
			if r.Header.Get("X-Finnhub-Token") != "k" || r.URL.Query().Get("symbol") != "AAPL" {
				t.Errorf("unexpected request: %s", r.URL)
			}
		})
		c := NewStockClient(srv.Client(), "k", srv.URL)
		got, err := c.Fetch(context.Background(), json.RawMessage(`{"ticker":"AAPL"}`))
		if err != nil {
			t.Fatal(err)
		}
		want := StockData{Ticker: "AAPL", Price: 190.5, Change: 1.5, PercentChange: 0.79, PreviousClose: 189}
		if got != want {
			t.Fatalf("got %+v, want %+v", got, want)
		}
	})

	t.Run("unknown symbol is all zeros", func(t *testing.T) {
		srv := stub(t, 200, `{"c":0,"d":null,"dp":null,"pc":0,"t":0}`, nil)
		c := NewStockClient(srv.Client(), "k", srv.URL)
		if _, err := c.Fetch(context.Background(), json.RawMessage(`{"ticker":"NOPE"}`)); err == nil {
			t.Fatal("expected error for empty quote")
		}
	})
}

func TestCurrencyClient(t *testing.T) {
	t.Run("rate", func(t *testing.T) {
		srv := stub(t, 200, `{"amount":1,"base":"USD","date":"2026-09-12","rates":{"EUR":0.91}}`, func(r *http.Request) {
			if r.URL.Path != "/v1/latest" || r.URL.Query().Get("base") != "USD" || r.URL.Query().Get("symbols") != "EUR" {
				t.Errorf("unexpected request: %s", r.URL)
			}
		})
		c := NewCurrencyClient(srv.Client(), srv.URL)
		got, err := c.Fetch(context.Background(), json.RawMessage(`{"base":"USD","target":"EUR"}`))
		if err != nil {
			t.Fatal(err)
		}
		want := CurrencyData{Base: "USD", Target: "EUR", Rate: 0.91, Date: "2026-09-12"}
		if got != want {
			t.Fatalf("got %+v, want %+v", got, want)
		}
	})

	t.Run("missing rate", func(t *testing.T) {
		srv := stub(t, 200, `{"base":"USD","date":"2026-09-12","rates":{}}`, nil)
		c := NewCurrencyClient(srv.Client(), srv.URL)
		if _, err := c.Fetch(context.Background(), json.RawMessage(`{"base":"USD","target":"XXX"}`)); err == nil {
			t.Fatal("expected error for missing rate")
		}
	})
}

func TestUpstreamErrors(t *testing.T) {
	t.Run("non-2xx", func(t *testing.T) {
		srv := stub(t, 401, `{"cod":401,"message":"Invalid API key"}`, nil)
		c := NewWeatherClient(srv.Client(), "bad", srv.URL)
		_, err := c.Fetch(context.Background(), json.RawMessage(`{"city":"London"}`))
		var ue *UpstreamError
		if !errors.As(err, &ue) || ue.Status != 401 || ue.Provider != "OpenWeatherMap" {
			t.Fatalf("got %v, want UpstreamError 401", err)
		}
	})

	t.Run("rate limited", func(t *testing.T) {
		srv := stub(t, 429, `slow down`, nil)
		c := NewCurrencyClient(srv.Client(), srv.URL)
		_, err := c.Fetch(context.Background(), json.RawMessage(`{"base":"USD","target":"EUR"}`))
		if err == nil || err.Error() != "Frankfurter rate limit exceeded (HTTP 429)" {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("cancelled context", func(t *testing.T) {
		srv := stub(t, 200, `{}`, nil)
		c := NewCurrencyClient(srv.Client(), srv.URL)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err := c.Fetch(ctx, json.RawMessage(`{"base":"USD","target":"EUR"}`)); !errors.Is(err, context.Canceled) {
			t.Fatalf("got %v, want context.Canceled", err)
		}
	})

	t.Run("malformed json", func(t *testing.T) {
		srv := stub(t, 200, `not json`, nil)
		c := NewCurrencyClient(srv.Client(), srv.URL)
		if _, err := c.Fetch(context.Background(), json.RawMessage(`{"base":"USD","target":"EUR"}`)); err == nil {
			t.Fatal("expected decode error")
		}
	})
}

func TestNewRegistry(t *testing.T) {
	reg := NewRegistry(Config{})
	for _, typ := range []string{"weather", "news", "stock", "currency"} {
		if _, ok := reg[domainType(typ)]; !ok {
			t.Errorf("no client for %s", typ)
		}
	}
}
