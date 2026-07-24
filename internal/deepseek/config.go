// Package deepseek is a thin client for DeepSeek's OpenAI-compatible
// chat-completions API, used by cmd/mcp-deepseek.
//
// Unlike internal/secfetch, this client always dials a single fixed,
// operator-configured host (DeepSeek's API) rather than an arbitrary
// caller-supplied URL. The destination is never attacker-controlled per
// request, so there is no allowlist/SSRF surface to defend here -- that
// threat model is specific to secfetch's fetch_url tool, not this one.
package deepseek

import (
	"os"
	"strconv"
	"time"
)

// Config controls how Client talks to the DeepSeek API. APIKey has no
// default and Load does not fail if it's empty (matching secfetch's
// fail-closed-at-call-time pattern): every chat request is rejected with a
// clear error until DEEPSEEK_API_KEY is set, rather than the server
// refusing to start.
type Config struct {
	APIKey       string
	BaseURL      string
	DefaultModel string
	Timeout      time.Duration
	MaxBodyBytes int64
}

// Load reads Config from the environment, matching the getEnv-with-default
// pattern used by internal/config and internal/secfetch.
func Load() Config {
	return Config{
		APIKey:  os.Getenv("DEEPSEEK_API_KEY"),
		BaseURL: getEnv("DEEPSEEK_BASE_URL", "https://api.deepseek.com"),
		// DeepSeek's own docs give "deepseek-chat" as the model to specify
		// for standard chat completions; override via DEEPSEEK_MODEL if
		// your account's available model IDs have moved on since.
		DefaultModel: getEnv("DEEPSEEK_MODEL", "deepseek-chat"),
		Timeout:      getDuration("DEEPSEEK_TIMEOUT", 60*time.Second),
		MaxBodyBytes: getInt64("DEEPSEEK_MAX_BODY_BYTES", 4<<20),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getInt64(key string, fallback int64) int64 {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return n
		}
	}
	return fallback
}

func getDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
