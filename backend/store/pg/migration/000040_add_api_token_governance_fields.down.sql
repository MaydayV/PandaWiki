ALTER TABLE api_tokens
DROP COLUMN IF EXISTS rate_limit_per_minute;

ALTER TABLE api_tokens
DROP COLUMN IF EXISTS daily_quota;
