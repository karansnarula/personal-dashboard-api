package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/karansnarula/personal-dashboard-api/internal/api/reqctx"
	"github.com/karansnarula/personal-dashboard-api/internal/api/response"
)

// TokenParser is satisfied by *auth.TokenIssuer.
type TokenParser interface {
	Parse(token string) (int64, error)
}

// Auth requires a valid "Authorization: Bearer <jwt>" header and stores the
// user ID in the request context.
func Auth(parser TokenParser) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, ok := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
		token = strings.TrimSpace(token)
		if !ok || token == "" {
			response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized,
				"missing or malformed Authorization header (expected: Bearer <token>)")
			return
		}
		userID, err := parser.Parse(token)
		if err != nil {
			response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "invalid or expired token")
			return
		}
		reqctx.SetUserID(c, userID)
		c.Next()
	}
}
