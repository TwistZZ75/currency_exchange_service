package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

func Logging(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		dur := time.Since(start)

		attrs := []any{
			"http.method", c.Request.Method,
			"http.path", c.Request.URL.Path,
			"http.status", c.Writer.Status(),
			"http.duration_ms", dur.Milliseconds(),
			"request_id", c.GetString("request_id"),
		}
		if errs := c.Errors.String(); errs != "" {
			attrs = append(attrs, "errors", errs)
		}
		switch {
		case c.Writer.Status() >= 500:
			log.Error("http request", attrs...)
		case c.Writer.Status() >= 400:
			log.Warn("http request", attrs...)
		default:
			log.Info("http request", attrs...)
		}
	}
}
