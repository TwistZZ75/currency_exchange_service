package service

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"gw-currency-wallet/internal/auth"
	"gw-currency-wallet/internal/domain"
	"gw-currency-wallet/internal/storages"
	"gw-currency-wallet/internal/storages/mocks"
)

func newLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestRegister_OK(t *testing.T) {
	st := &mocks.Storage{}
	st.On("CreateUser", mock.Anything, mock.AnythingOfType("*domain.User")).
		Return(int64(1), nil).Once()

	svc := &Service{Storage: st, Logger: newLogger()}
	err := svc.Register(context.Background(), "user1", "pass123", "u@example.com")
	require.NoError(t, err)
	st.AssertExpectations(t)

	arg := st.Calls[0].Arguments.Get(1).(*domain.User)
	require.NotEqual(t, "pass123", arg.Password, "пароль должен быть захеширован")
	require.NoError(t, bcrypt.CompareHashAndPassword([]byte(arg.Password), []byte("pass123")))
}

func TestRegister_UsernameTaken(t *testing.T) {
	st := &mocks.Storage{}
	st.On("CreateUser", mock.Anything, mock.Anything).Return(int64(0), storages.ErrUserExists).Once()

	svc := &Service{Storage: st, Logger: newLogger()}
	err := svc.Register(context.Background(), "user1", "pass123", "u@example.com")
	require.ErrorIs(t, err, ErrUserExists)
}

func TestRegister_EmailTaken(t *testing.T) {
	st := &mocks.Storage{}
	st.On("CreateUser", mock.Anything, mock.Anything).Return(int64(0), storages.ErrUserExists).Once()

	svc := &Service{Storage: st, Logger: newLogger()}
	err := svc.Register(context.Background(), "user1", "pass123", "u@example.com")
	require.ErrorIs(t, err, ErrUserExists)
}

func TestRegister_GenericUserExists(t *testing.T) {
	st := &mocks.Storage{}
	st.On("CreateUser", mock.Anything, mock.Anything).Return(int64(0), storages.ErrUserExists).Once()

	svc := &Service{Storage: st, Logger: newLogger()}
	err := svc.Register(context.Background(), "user1", "pass123", "u@example.com")
	require.ErrorIs(t, err, ErrUserExists)
}

func TestLogin_OK(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass123"), bcrypt.MinCost)

	st := &mocks.Storage{}
	st.On("GetUserByUsername", mock.Anything, "user1").
		Return(&domain.User{ID: 42, Username: "user1", Password: string(hash)}, nil).Once()

	jwt := auth.NewJWTManager("test-secret", time.Hour)
	svc := &Service{Storage: st, JWT: jwt, Logger: newLogger()}

	token, err := svc.Login(context.Background(), "user1", "pass123")
	require.NoError(t, err)
	require.NotEmpty(t, token)

	uid, err := jwt.Parse(token)
	require.NoError(t, err)
	require.EqualValues(t, 42, uid)
}

func TestLogin_WrongPassword(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("correct"), bcrypt.MinCost)

	st := &mocks.Storage{}
	st.On("GetUserByUsername", mock.Anything, "user1").
		Return(&domain.User{ID: 42, Password: string(hash)}, nil).Once()

	svc := &Service{Storage: st, JWT: auth.NewJWTManager("s", time.Hour), Logger: newLogger()}
	_, err := svc.Login(context.Background(), "user1", "wrong")
	require.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestLogin_UserNotFound(t *testing.T) {
	st := &mocks.Storage{}
	st.On("GetUserByUsername", mock.Anything, "user1").
		Return(nil, storages.ErrNotFound).Once()

	svc := &Service{Storage: st, JWT: auth.NewJWTManager("s", time.Hour), Logger: newLogger()}
	_, err := svc.Login(context.Background(), "user1", "pass")
	require.ErrorIs(t, err, ErrInvalidCredentials)
}
