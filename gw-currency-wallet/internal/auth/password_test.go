package auth

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHashPassword_And_Check(t *testing.T) {
	hash, err := HashPassword("pass123")
	require.NoError(t, err)
	require.NotEmpty(t, hash)
	require.NotEqual(t, "pass123", hash)

	require.True(t, CheckPassword(hash, "pass123"))
	require.False(t, CheckPassword(hash, "wrong"))
	require.False(t, CheckPassword(hash, ""))
}

func TestHashPassword_DifferentSalt(t *testing.T) {
	h1, err := HashPassword("same")
	require.NoError(t, err)
	h2, err := HashPassword("same")
	require.NoError(t, err)
	require.NotEqual(t, h1, h2, "bcrypt must create different hash because of salt")

	require.True(t, CheckPassword(h1, "same"))
	require.True(t, CheckPassword(h2, "same"))
}

func TestCheckPassword_BadHash(t *testing.T) {
	require.False(t, CheckPassword("not-a-hash", "pass"))
}
