// Package reqctx defines the keys used to pass per-request values (request
// ID, authenticated user) through the Gin context. Middleware writes them;
// handlers and response helpers read them.
package reqctx

import "github.com/gin-gonic/gin"

const (
	keyRequestID = "reqctx.request_id"
	keyUserID    = "reqctx.user_id"
)

func SetRequestID(c *gin.Context, id string) { c.Set(keyRequestID, id) }

func RequestID(c *gin.Context) string { return c.GetString(keyRequestID) }

func SetUserID(c *gin.Context, id int64) { c.Set(keyUserID, id) }

// UserID returns the authenticated user's ID, or 0 if the request is not
// authenticated.
func UserID(c *gin.Context) int64 {
	v, _ := c.Get(keyUserID)
	id, _ := v.(int64)
	return id
}
