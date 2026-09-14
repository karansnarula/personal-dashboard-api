package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/karansnarula/personal-dashboard-api/internal/domain"
	"github.com/karansnarula/personal-dashboard-api/internal/widget"
)

// ModeSequential labels dashboards assembled one widget at a time.
const ModeSequential = "sequential"

type WidgetLister interface {
	ListByUser(ctx context.Context, userID int64) ([]domain.Widget, error)
}

// DashboardService loads a user's widgets and resolves live data for each.
type DashboardService struct {
	widgets WidgetLister
	clients widget.Registry
	timeout time.Duration
	logger  *slog.Logger
}

// NewDashboardService builds the service. timeout bounds each external call.
func NewDashboardService(widgets WidgetLister, clients widget.Registry, timeout time.Duration, logger *slog.Logger) *DashboardService {
	if logger == nil {
		logger = slog.Default()
	}
	return &DashboardService{widgets: widgets, clients: clients, timeout: timeout, logger: logger}
}

// Build assembles the dashboard for userID. Widget failures are reported
// per widget, never as a failure of the whole call.
func (s *DashboardService) Build(ctx context.Context, userID int64) (domain.Dashboard, error) {
	list, err := s.widgets.ListByUser(ctx, userID)
	if err != nil {
		return domain.Dashboard{}, err
	}

	results, elapsed := s.fetchSequential(ctx, list)

	return domain.Dashboard{
		Mode:        ModeSequential,
		WidgetCount: len(list),
		ElapsedMS:   elapsed.Milliseconds(),
		Widgets:     results,
	}, nil
}

// fetchSequential is the deliberate "before" implementation: widgets are
// resolved one after another, so total latency is roughly the sum of every
// external call. The elapsed time is returned so it can be compared against
// a concurrent implementation later.
func (s *DashboardService) fetchSequential(ctx context.Context, widgets []domain.Widget) ([]domain.WidgetResult, time.Duration) {
	start := time.Now()
	results := make([]domain.WidgetResult, 0, len(widgets))
	for _, w := range widgets {
		results = append(results, s.fetchOne(ctx, w))
	}
	elapsed := time.Since(start)

	s.logger.InfoContext(ctx, "dashboard assembled",
		"mode", ModeSequential,
		"widgets", len(widgets),
		"elapsed_ms", elapsed.Milliseconds(),
	)
	return results, elapsed
}

// fetchOne resolves a single widget under a per-call timeout. Errors are
// captured in the result rather than returned.
func (s *DashboardService) fetchOne(ctx context.Context, w domain.Widget) domain.WidgetResult {
	res := domain.WidgetResult{WidgetID: w.ID, Type: w.Type, Config: w.Config}

	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	start := time.Now()
	data, err := s.fetch(ctx, w)
	res.DurationMS = time.Since(start).Milliseconds()

	if err != nil {
		res.Status = domain.ResultError
		res.Error = s.describe(err, w.Type)
		s.logger.WarnContext(ctx, "widget fetch failed",
			"widget_id", w.ID, "type", w.Type, "duration_ms", res.DurationMS, "err", err,
		)
		return res
	}

	res.Status = domain.ResultOK
	res.Data = data
	s.logger.DebugContext(ctx, "widget fetched",
		"widget_id", w.ID, "type", w.Type, "duration_ms", res.DurationMS,
	)
	return res
}

func (s *DashboardService) fetch(ctx context.Context, w domain.Widget) (any, error) {
	client, ok := s.clients[w.Type]
	if !ok {
		return nil, fmt.Errorf("no client registered for widget type %q", w.Type)
	}
	return client.Fetch(ctx, w.Config)
}

// describe turns an error into a message suitable for the response.
func (s *DashboardService) describe(err error, t domain.WidgetType) string {
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Sprintf("%s provider did not respond within %s", t, s.timeout)
	}
	return err.Error()
}
