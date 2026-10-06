package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"gw-currency-wallet/internal/auth"
)

func setupAuthRouter(jwt *auth.JWTManager) *gin.Engine {
	r := gin.New()
	r.Use(Auth(jwt))
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"user_id": UserID(c)})
	})
	return r
}

func TestAuth_OK(t *testing.T) {
	jwt := auth.NewJWTManager("secret", time.Hour)
	r := setupAuthRouter(jwt)

	token, _ := jwt.Generate(42, "u")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"user_id":42`)
}

func TestAuth_MissingHeader(t *testing.T) {
	jwt := auth.NewJWTManager("secret", time.Hour)
	r := setupAuthRouter(jwt)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuth_NotBearer(t *testing.T) {
	jwt := auth.NewJWTManager("secret", time.Hour)
	r := setupAuthRouter(jwt)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Token abc")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuth_InvalidToken(t *testing.T) {
	jwt := auth.NewJWTManager("secret", time.Hour)
	r := setupAuthRouter(jwt)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer garbage")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuth_WrongSecret(t *testing.T) {
	a := auth.NewJWTManager("a", time.Hour)
	b := auth.NewJWTManager("b", time.Hour)
	token, _ := a.Generate(1, "u")

	r := setupAuthRouter(b)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}
