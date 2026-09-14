package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/karansnarula/personal-dashboard-api/internal/domain"
	"github.com/karansnarula/personal-dashboard-api/internal/widget"
)

type fakeLister struct{ widgets []domain.Widget }

func (f fakeLister) ListByUser(context.Context, int64) ([]domain.Widget, error) {
	return f.widgets, nil
}

type stubClient struct {
	data  any
	err   error
	delay time.Duration
}

func (s stubClient) Fetch(ctx context.Context, _ json.RawMessage) (any, error) {
	if s.delay > 0 {
		select {
		case <-time.After(s.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return s.data, s.err
}

func TestDashboardService_PartialFailure(t *testing.T) {
	widgets := []domain.Widget{
		{ID: 1, Type: domain.WidgetCurrency, Config: json.RawMessage(`{"base":"USD","target":"EUR"}`)},
		{ID: 2, Type: domain.WidgetStock, Config: json.RawMessage(`{"ticker":"AAPL"}`)},
		{ID: 3, Type: domain.WidgetWeather, Config: json.RawMessage(`{"city":"Oslo"}`)},
		{ID: 4, Type: domain.WidgetType("unknown"), Config: json.RawMessage(`{}`)},
	}
	clients := widget.Registry{
		domain.WidgetCurrency: stubClient{data: map[string]float64{"rate": 0.9}},
		domain.WidgetStock:    stubClient{err: errors.New("upstream exploded")},
		domain.WidgetWeather:  stubClient{data: "sunny", delay: time.Second}, // slower than timeout
	}
	svc := NewDashboardService(fakeLister{widgets}, clients, 50*time.Millisecond, slog.New(slog.NewTextHandler(io.Discard, nil)))

	start := time.Now()
	d, err := svc.Build(context.Background(), 1)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if took := time.Since(start); took > 500*time.Millisecond {
		t.Fatalf("timeout not enforced; took %s", took)
	}

	if d.Mode != ModeSequential || d.WidgetCount != 4 || len(d.Widgets) != 4 {
		t.Fatalf("unexpected envelope: %+v", d)
	}

	// Results come back in widget order regardless of outcome.
	for i, w := range widgets {
		if d.Widgets[i].WidgetID != w.ID {
			t.Fatalf("result %d: widget %d, want %d", i, d.Widgets[i].WidgetID, w.ID)
		}
	}

	if r := d.Widgets[0]; r.Status != domain.ResultOK || r.Data == nil || r.Error != "" {
		t.Errorf("currency: %+v", r)
	}
	if r := d.Widgets[1]; r.Status != domain.ResultError || r.Error != "upstream exploded" || r.Data != nil {
		t.Errorf("stock: %+v", r)
	}
	if r := d.Widgets[2]; r.Status != domain.ResultError || r.Error != "weather provider did not respond within 50ms" {
		t.Errorf("weather timeout: %+v", r)
	}
	if r := d.Widgets[3]; r.Status != domain.ResultError || r.Error == "" {
		t.Errorf("unknown type: %+v", r)
	}
}

func TestDashboardService_Empty(t *testing.T) {
	svc := NewDashboardService(fakeLister{}, widget.Registry{}, time.Second, nil)
	d, err := svc.Build(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if d.WidgetCount != 0 || d.Widgets == nil || len(d.Widgets) != 0 {
		t.Fatalf("expected empty non-nil widgets: %+v", d)
	}
}
