package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestLogging_OK(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))

	r := gin.New()
	r.Use(RequestID())
	r.Use(Logging(log))
	r.GET("/ok", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/ok", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, buf.String(), `"http.status":200`)
	require.Contains(t, buf.String(), `"http.path":"/ok"`)
	require.Contains(t, buf.String(), `"request_id"`)
	require.Contains(t, buf.String(), `"http.duration_ms"`)
}

func TestLogging_Error_Level(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))

	r := gin.New()
	r.Use(RequestID())
	r.Use(Logging(log))
	r.GET("/err", func(c *gin.Context) { c.Status(http.StatusInternalServerError) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/err", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.Contains(t, buf.String(), `"level":"ERROR"`)
	require.Contains(t, buf.String(), `"http.status":500`)
}

func TestLogging_Warn_Level(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))

	r := gin.New()
	r.Use(RequestID())
	r.Use(Logging(log))
	r.GET("/nf", func(c *gin.Context) { c.Status(http.StatusNotFound) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/nf", nil)
	r.ServeHTTP(w, req)

	require.Contains(t, buf.String(), `"level":"WARN"`)
}
