package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func init() { gin.SetMode(gin.TestMode) }

func TestRequestID_Generated(t *testing.T) {
	r := gin.New()
	r.Use(RequestID())
	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, c.GetString("request_id"))
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	header := w.Header().Get(RequestIDHeader)
	require.NotEmpty(t, header)
	require.Equal(t, header, w.Body.String())
}

func TestRequestID_FromClient(t *testing.T) {
	r := gin.New()
	r.Use(RequestID())
	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, c.GetString("request_id"))
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set(RequestIDHeader, "client-abc")
	r.ServeHTTP(w, req)

	require.Equal(t, "client-abc", w.Header().Get(RequestIDHeader))
	require.Equal(t, "client-abc", w.Body.String())
}
