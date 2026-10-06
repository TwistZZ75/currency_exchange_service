package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestJWT_GenerateAndParse(t *testing.T) {
	mgr := NewJWTManager("test-secret", time.Hour)
	token, err := mgr.Generate(42, "user1")
	require.NoError(t, err)
	require.NotEmpty(t, token)

	uid, err := mgr.Parse(token)
	require.NoError(t, err)
	require.EqualValues(t, 42, uid)
}

func TestJWT_WrongSecret(t *testing.T) {
	a := NewJWTManager("secret-a", time.Hour)
	b := NewJWTManager("secret-b", time.Hour)

	token, err := a.Generate(1, "u")
	require.NoError(t, err)

	_, err = b.Parse(token)
	require.Error(t, err, "sign with another secret must be fail")
}

func TestJWT_Expired(t *testing.T) {
	mgr := NewJWTManager("s", -time.Hour)
	token, err := mgr.Generate(1, "u")
	require.NoError(t, err)

	_, err = mgr.Parse(token)
	require.Error(t, err)
}

func TestJWT_Garbage(t *testing.T) {
	mgr := NewJWTManager("s", time.Hour)
	_, err := mgr.Parse("not.a.jwt")
	require.Error(t, err)

	_, err = mgr.Parse("")
	require.Error(t, err)
}
