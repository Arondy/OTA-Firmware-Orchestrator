package update

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsGreaterSemver_CurrentLower_ReturnsFalse(t *testing.T) {
	t.Parallel()
	greater, err := isGreaterSemver("1.0.0", "2.0.0")
	require.NoError(t, err)
	assert.False(t, greater)
}

func TestIsGreaterSemver_CurrentEqual_ReturnsTrue(t *testing.T) {
	t.Parallel()
	greater, err := isGreaterSemver("1.2.3", "1.2.3")
	require.NoError(t, err)
	assert.True(t, greater)
}

func TestIsGreaterSemver_CurrentHigher_ReturnsTrue(t *testing.T) {
	t.Parallel()
	greater, err := isGreaterSemver("2.0.0", "1.0.0")
	require.NoError(t, err)
	assert.True(t, greater)
}

func TestIsGreaterSemver_Prerelease(t *testing.T) {
	t.Parallel()
	lower, err := isGreaterSemver("1.0.0-alpha", "1.0.0")
	require.NoError(t, err)
	assert.False(t, lower)

	higher, err := isGreaterSemver("1.0.0", "1.0.0-beta")
	require.NoError(t, err)
	assert.True(t, higher)
}

func TestIsGreaterSemver_InvalidCurrent_ReturnsError(t *testing.T) {
	t.Parallel()
	_, err := isGreaterSemver("not-a-version", "1.0.0")
	require.Error(t, err)
}

func TestIsGreaterSemver_InvalidTarget_ReturnsError(t *testing.T) {
	t.Parallel()
	_, err := isGreaterSemver("1.0.0", "")
	require.Error(t, err)
}
