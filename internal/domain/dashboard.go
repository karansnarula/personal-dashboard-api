package domain

import "encoding/json"

type ResultStatus string

const (
	ResultOK    ResultStatus = "ok"
	ResultError ResultStatus = "error"
)

// WidgetResult is the live-data outcome for one widget. A failed widget
// carries Status "error" and a message instead of Data, so a frontend can
// render "unavailable" while the rest of the dashboard still works.
type WidgetResult struct {
	WidgetID   int64           `json:"widget_id"`
	Type       WidgetType      `json:"type"`
	Config     json.RawMessage `json:"config"`
	Status     ResultStatus    `json:"status"`
	Data       any             `json:"data,omitempty"`
	Error      string          `json:"error,omitempty"`
	DurationMS int64           `json:"duration_ms"`
}

// Dashboard is the assembled response. Mode records which fetch strategy
// produced it so timings can be compared across implementations.
type Dashboard struct {
	Mode        string         `json:"mode"`
	WidgetCount int            `json:"widget_count"`
	ElapsedMS   int64          `json:"elapsed_ms"`
	Widgets     []WidgetResult `json:"widgets"`
}
