package response

import "github.com/karansnarula/personal-dashboard-api/internal/domain"

type WidgetList struct {
	Widgets []domain.Widget `json:"widgets"`
}
