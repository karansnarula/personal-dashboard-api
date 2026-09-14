package middleware

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORS allows browser apps on the given origins to call the API. Only
// browsers enforce CORS; curl and Postman are unaffected either way.
func CORS(allowedOrigins []string) gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins:  allowedOrigins,
		AllowMethods:  []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:  []string{"Authorization", "Content-Type", HeaderRequestID},
		ExposeHeaders: []string{HeaderRequestID},
		MaxAge:        12 * time.Hour,
	})
}
