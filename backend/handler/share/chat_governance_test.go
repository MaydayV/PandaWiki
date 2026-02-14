package share

import (
	"testing"

	"github.com/chaitin/panda-wiki/domain"
	"github.com/stretchr/testify/require"
)

func TestEvaluateAPITokenGovernanceViolation(t *testing.T) {
	t.Run("no token no violation", func(t *testing.T) {
		require.Equal(t, "", evaluateAPITokenGovernanceViolation(nil, 100, 100))
	})

	t.Run("rate limit exceeded", func(t *testing.T) {
		token := &domain.APIToken{RateLimitPerMinute: 10, DailyQuota: 100}
		require.Equal(t, "rate_limit_error", evaluateAPITokenGovernanceViolation(token, 10, 20))
	})

	t.Run("daily quota exceeded", func(t *testing.T) {
		token := &domain.APIToken{RateLimitPerMinute: 100, DailyQuota: 50}
		require.Equal(t, "insufficient_quota", evaluateAPITokenGovernanceViolation(token, 10, 50))
	})

	t.Run("unlimited", func(t *testing.T) {
		token := &domain.APIToken{RateLimitPerMinute: 0, DailyQuota: 0}
		require.Equal(t, "", evaluateAPITokenGovernanceViolation(token, 1000, 100000))
	})
}

func TestOpenAIErrorStatusCodeQuotaAndRateLimit(t *testing.T) {
	require.Equal(t, 429, openAIErrorStatusCode("rate_limit_error"))
	require.Equal(t, 429, openAIErrorStatusCode("insufficient_quota"))
}
