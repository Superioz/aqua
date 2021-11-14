package middleware

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func RestrictBodySize(max int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		var w http.ResponseWriter = c.Writer
		c.Request.Body = http.MaxBytesReader(w, c.Request.Body, max)

		c.Next()
	}
}
