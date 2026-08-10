package secfetch

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config controls what secfetch.Client is allowed to reach and how much of
// a response it will return. There is no default allowlist: MCP_FETCH_ALLOWED_HOSTS
// must be set explicitly or every fetch is rejected.
type Config struct {
	AllowedHosts []string
	DeniedHosts  []string
	Timeout      time.Duration
	MaxRedirects int
	MaxBodyBytes int64
	UserAgent    string
}

// Load reads Config from the environment, matching the getEnv-with-default
// pattern used by internal/config.
func Load() Config {
	return Config{
		AllowedHosts: splitCSV(getEnv("MCP_FETCH_ALLOWED_HOSTS", "")),
		DeniedHosts:  splitCSV(getEnv("MCP_FETCH_DENIED_HOSTS", "")),
		Timeout:      getDuration("MCP_FETCH_TIMEOUT", 10*time.Second),
		MaxRedirects: getInt("MCP_FETCH_MAX_REDIRECTS", 3),
		MaxBodyBytes: getInt64("MCP_FETCH_MAX_BODY_BYTES", 2<<20),
		UserAgent:    getEnv("MCP_FETCH_USER_AGENT", "myproject-mcp-fetch/1.0"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
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

func splitCSV(v string) []string {
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
