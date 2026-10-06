package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"gw-analytics/internal/metrics"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "gw-analytics/docs"
)

func NewRouter(h *Handlers) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(metricsMiddleware())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := r.Group("/api/v1/analytics")
	{
		api.GET("/events", h.Counts)
		api.GET("/latency", h.Latency)
		api.GET("/errors", h.Errors)
	}

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return r
}

func metricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		dur := time.Since(start)

		metrics.HTTPRequests.WithLabelValues(
			c.Request.Method,
			c.FullPath(),
			strconv.Itoa(c.Writer.Status()),
		).Inc()

		_ = dur
	}
}
