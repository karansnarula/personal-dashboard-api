package response

import (
	"github.com/gin-gonic/gin"

	"github.com/karansnarula/personal-dashboard-api/internal/api/reqctx"
)

// Machine-readable error codes returned in the error envelope.
const (
	CodeBadRequest       = "bad_request"
	CodeValidation       = "validation_error"
	CodeUnauthorized     = "unauthorized"
	CodeNotFound         = "not_found"
	CodeMethodNotAllowed = "method_not_allowed"
	CodeConflict         = "conflict"
	CodeInternal         = "internal_error"
	CodeUnavailable      = "service_unavailable"
)

// ErrorResponse is the single error shape every endpoint returns.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

// JSON writes a success body.
func JSON(c *gin.Context, status int, body any) {
	c.JSON(status, body)
}

// Error writes the error envelope and aborts the handler chain.
func Error(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, ErrorResponse{Error: ErrorDetail{
		Code:      code,
		Message:   message,
		RequestID: reqctx.RequestID(c),
	}})
}
