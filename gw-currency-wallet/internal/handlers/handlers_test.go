package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"gw-currency-wallet/internal/auth"
	"gw-currency-wallet/internal/domain"
	"gw-currency-wallet/internal/handlers"
	"gw-currency-wallet/internal/middleware"
	"gw-currency-wallet/internal/service"
	"gw-currency-wallet/internal/storages"
	"gw-currency-wallet/internal/storages/mocks"
)

func init() { gin.SetMode(gin.TestMode) }

func newLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

type noopRates struct{}

func (noopRates) Get(context.Context) (map[string]float64, error) {
	return map[string]float64{"USD": 90, "EUR": 100, "RUB": 1}, nil
}
func (noopRates) Invalidate() {}

func setup(svc *service.Service) (*gin.Engine, *auth.JWTManager) {
	jwt := auth.NewJWTManager("test-secret", time.Hour)
	svc.JWT = jwt
	svc.Logger = newLogger()
	if svc.Rates == nil {
		svc.Rates = noopRates{}
	}

	h := handlers.NewHandler(svc, svc.Logger)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestID())

	api := r.Group("/api/v1")
	{
		api.POST("/register", h.Register)
		api.POST("/login", h.Login)

		authed := api.Group("")
		authed.Use(middleware.Auth(jwt))
		{
			authed.GET("/balance", h.Balance)
			authed.POST("/wallet/deposit", h.Deposit)
			authed.POST("/wallet/withdraw", h.Withdraw)
			authed.GET("/exchange/rates", h.Rates)
			authed.POST("/exchange", h.Exchange)
		}
	}
	return r, jwt
}

// ---------- Register ----------

func TestRegister_HTTP_OK(t *testing.T) {
	st := &mocks.Storage{}
	st.On("CreateUser", mock.Anything, mock.Anything).Return(int64(1), nil).Once()

	r, _ := setup(&service.Service{Storage: st})
	body := `{"username":"user1","password":"pass123","email":"u@example.com"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	require.Contains(t, w.Body.String(), "User registered successfully")
}

func TestRegister_HTTP_BadRequest(t *testing.T) {
	r, _ := setup(&service.Service{Storage: &mocks.Storage{}})
	// пароль короче min=6
	body := `{"username":"user1","password":"p","email":"u@example.com"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

// Общая проверка: любой конфликт уникальности (username или email)
// → 400 с сообщением "Username or email already exists".
// Отдельного различия username/email в API нет.
func TestRegister_HTTP_AlreadyExists(t *testing.T) {
	cases := []struct {
		name string
		err  error
	}{
		{"generic conflict", storages.ErrUserExists},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			st := &mocks.Storage{}
			st.On("CreateUser", mock.Anything, mock.Anything).
				Return(int64(0), c.err).Once()

			r, _ := setup(&service.Service{Storage: st})
			body := `{"username":"user1","password":"pass123","email":"u@example.com"}`

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/register", bytes.NewBufferString(body))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)

			require.Equal(t, http.StatusBadRequest, w.Code)
			require.Contains(t, w.Body.String(), "Username or email already exists",
				"любой конфликт уникальности должен давать общую ошибку")
		})
	}
}

// ---------- Login ----------

func TestLogin_HTTP_OK(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass123"), bcrypt.MinCost)

	st := &mocks.Storage{}
	st.On("GetUserByUsername", mock.Anything, "user1").
		Return(&domain.User{ID: 1, Username: "user1", Password: string(hash)}, nil).Once()

	r, _ := setup(&service.Service{Storage: st})
	body := `{"username":"user1","password":"pass123"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.NotEmpty(t, resp["token"])
}

func TestLogin_HTTP_Unauthorized(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("correct"), bcrypt.MinCost)

	st := &mocks.Storage{}
	st.On("GetUserByUsername", mock.Anything, "user1").
		Return(&domain.User{ID: 1, Password: string(hash)}, nil).Once()

	r, _ := setup(&service.Service{Storage: st})
	body := `{"username":"user1","password":"wrong"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	require.Contains(t, w.Body.String(), "Invalid username or password")
}

func TestLogin_HTTP_UserNotFound(t *testing.T) {
	st := &mocks.Storage{}
	st.On("GetUserByUsername", mock.Anything, "ghost").
		Return(nil, storages.ErrNotFound).Once()

	r, _ := setup(&service.Service{Storage: st})
	body := `{"username":"ghost","password":"whatever"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	// Не раскрываем, что пользователя нет, — та же ошибка, что и при неверном пароле.
	require.Equal(t, http.StatusUnauthorized, w.Code)
	require.Contains(t, w.Body.String(), "Invalid username or password")
}

// ---------- Balance ----------

func TestBalance_WithoutAuth(t *testing.T) {
	r, _ := setup(&service.Service{Storage: &mocks.Storage{}})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/balance", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestBalance_HTTP_OK(t *testing.T) {
	st := &mocks.Storage{}
	st.On("GetWallet", mock.Anything, int64(42)).
		Return(&domain.Wallet{UserID: 42, USD: 100, RUB: 5000, EUR: 50}, nil).Once()

	r, jwt := setup(&service.Service{Storage: st})
	token, _ := jwt.Generate(42, "user1")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/balance", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"USD":100`)
	require.Contains(t, w.Body.String(), `"RUB":5000`)
	require.Contains(t, w.Body.String(), `"EUR":50`)
}

// ---------- Deposit ----------

func TestDeposit_HTTP_OK(t *testing.T) {
	st := &mocks.Storage{}
	st.On("Deposit", mock.Anything, mock.Anything, int64(42), "USD", float64(100)).
		Return(&domain.Wallet{UserID: 42, USD: 100}, nil).Once()

	r, jwt := setup(&service.Service{Storage: st})
	token, _ := jwt.Generate(42, "user1")

	body := `{"amount":100,"currency":"USD"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/wallet/deposit", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "Account topped up successfully")
}

func TestDeposit_HTTP_BadRequest(t *testing.T) {
	r, jwt := setup(&service.Service{Storage: &mocks.Storage{}})
	token, _ := jwt.Generate(42, "user1")

	body := `{"amount":-1,"currency":"USD"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/wallet/deposit", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "Invalid amount or currency")
}

func TestDeposit_HTTP_WithIdempotencyKey(t *testing.T) {
	st := &mocks.Storage{}
	key := "550e8400-e29b-41d4-a716-446655440000"
	st.On("Deposit", mock.Anything, key, int64(42), "USD", float64(100)).
		Return(&domain.Wallet{UserID: 42, USD: 100}, nil).Once()

	r, jwt := setup(&service.Service{Storage: st})
	token, _ := jwt.Generate(42, "user1")

	body := `{"amount":100,"currency":"USD"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/wallet/deposit", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", key)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	st.AssertExpectations(t)
}

// ---------- Withdraw ----------

func TestWithdraw_HTTP_OK(t *testing.T) {
	st := &mocks.Storage{}
	st.On("Withdraw", mock.Anything, mock.Anything, int64(42), "USD", float64(50)).
		Return(&domain.Wallet{UserID: 42, USD: 50}, nil).Once()

	r, jwt := setup(&service.Service{Storage: st})
	token, _ := jwt.Generate(42, "user1")

	body := `{"amount":50,"currency":"USD"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/wallet/withdraw", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "Withdrawal successful")
}

func TestWithdraw_HTTP_InsufficientFunds(t *testing.T) {
	st := &mocks.Storage{}
	st.On("Withdraw", mock.Anything, mock.Anything, int64(42), "USD", float64(99999)).
		Return(nil, storages.ErrInsufficientFunds).Once()

	r, jwt := setup(&service.Service{Storage: st})
	token, _ := jwt.Generate(42, "user1")

	body := `{"amount":99999,"currency":"USD"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/wallet/withdraw", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "Insufficient funds or invalid amount")
}

// ---------- Rates ----------

func TestRates_HTTP_OK(t *testing.T) {
	r, jwt := setup(&service.Service{Storage: &mocks.Storage{}})
	token, _ := jwt.Generate(1, "u")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/exchange/rates", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"USD":90`)
}

// ---------- Exchange ----------

func TestExchange_HTTP_OK(t *testing.T) {
	st := &mocks.Storage{}
	st.On("Exchange", mock.Anything, mock.Anything, int64(42),
		"USD", "EUR", float64(100), mock.Anything, mock.Anything).
		Return(
			&domain.Wallet{UserID: 42, USD: 0, EUR: 90},
			&domain.Operation{ExchangedAmount: 90, Rate: 0.9},
			nil,
		).Once()

	r, jwt := setup(&service.Service{Storage: st})
	token, _ := jwt.Generate(42, "user1")

	body := `{"from_currency":"USD","to_currency":"EUR","amount":100}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/exchange", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "Exchange successful")
	require.Contains(t, w.Body.String(), `"EUR":90`)
}

func TestExchange_HTTP_BadRequest(t *testing.T) {
	r, jwt := setup(&service.Service{Storage: &mocks.Storage{}})
	token, _ := jwt.Generate(42, "user1")

	body := `{"from_currency":"USD","to_currency":"EUR","amount":0}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/exchange", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "Insufficient funds or invalid currencies")
}

func TestExchange_HTTP_InsufficientFunds(t *testing.T) {
	st := &mocks.Storage{}
	st.On("Exchange", mock.Anything, mock.Anything, int64(42),
		"USD", "EUR", float64(100), mock.Anything, mock.Anything).
		Return(nil, nil, storages.ErrInsufficientFunds).Once()

	r, jwt := setup(&service.Service{Storage: st})
	token, _ := jwt.Generate(42, "user1")

	body := `{"from_currency":"USD","to_currency":"EUR","amount":100}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/exchange", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "Insufficient funds or invalid currencies")
}
