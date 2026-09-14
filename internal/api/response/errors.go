package response

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/karansnarula/personal-dashboard-api/internal/api/reqctx"
	"github.com/karansnarula/personal-dashboard-api/internal/domain"
)

// FromError maps a domain error to an HTTP response. Anything unrecognised
// is logged with the request ID and reported as a generic 500 so internal
// details never leak to clients.
func FromError(c *gin.Context, err error) {
	var ve *domain.ValidationError
	switch {
	case errors.As(err, &ve):
		Error(c, http.StatusBadRequest, CodeValidation, ve.Message)
	case errors.Is(err, domain.ErrNotFound):
		Error(c, http.StatusNotFound, CodeNotFound, "resource not found")
	case errors.Is(err, domain.ErrEmailTaken):
		Error(c, http.StatusConflict, CodeConflict, "email already registered")
	case errors.Is(err, domain.ErrInvalidCredentials):
		Error(c, http.StatusUnauthorized, CodeUnauthorized, "invalid email or password")
	default:
		_ = c.Error(err)
		slog.ErrorContext(c.Request.Context(), "unhandled error",
			"request_id", reqctx.RequestID(c),
			"err", err,
		)
		Error(c, http.StatusInternalServerError, CodeInternal, "internal server error")
	}
}
