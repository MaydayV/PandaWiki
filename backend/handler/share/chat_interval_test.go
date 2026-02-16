package share

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCalculateAskIntervalRemainingSeconds(t *testing.T) {
	now := time.Date(2026, 2, 16, 12, 0, 0, 0, time.UTC)

	t.Run("disabled interval returns zero", func(t *testing.T) {
		require.Equal(t, 0, calculateAskIntervalRemainingSeconds(now.Add(-1*time.Second), now, 0))
	})

	t.Run("elapsed greater than interval returns zero", func(t *testing.T) {
		require.Equal(t, 0, calculateAskIntervalRemainingSeconds(now.Add(-6*time.Second), now, 5))
	})

	t.Run("elapsed equal to interval returns zero", func(t *testing.T) {
		require.Equal(t, 0, calculateAskIntervalRemainingSeconds(now.Add(-5*time.Second), now, 5))
	})

	t.Run("returns ceil remaining seconds", func(t *testing.T) {
		require.Equal(t, 3, calculateAskIntervalRemainingSeconds(now.Add(-2200*time.Millisecond), now, 5))
	})

	t.Run("future last ask time falls back to full interval", func(t *testing.T) {
		require.Equal(t, 5, calculateAskIntervalRemainingSeconds(now.Add(2*time.Second), now, 5))
	})
}
