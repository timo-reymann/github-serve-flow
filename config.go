package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	GitHubToken       string
	AllowedOwners     map[string]bool
	ListenAddr        string
	CacheDir          string
	CacheTTL          time.Duration
	CacheMaxSize      int64
	ClientCacheMaxAge int
	RateLimitWindow   time.Duration
	RateLimitMax      int
}

func LoadConfig() (*Config, error) {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("GITHUB_TOKEN is required")
	}

	ownersStr := os.Getenv("ALLOWED_OWNERS")
	if ownersStr == "" {
		return nil, fmt.Errorf("ALLOWED_OWNERS is required")
	}
	owners := make(map[string]bool)
	for _, o := range strings.Split(ownersStr, ",") {
		o = strings.TrimSpace(o)
		if o != "" {
			owners[strings.ToLower(o)] = true
		}
	}

	cfg := &Config{
		GitHubToken:       token,
		AllowedOwners:     owners,
		ListenAddr:        envOrDefault("LISTEN_ADDR", ":8080"),
		CacheDir:          envOrDefault("CACHE_DIR", filepath.Join(os.TempDir(), "github-serve-flow")),
		CacheTTL:          envDurationOrDefault("CACHE_TTL", time.Hour),
		CacheMaxSize:      envBytesOrDefault("CACHE_MAX_SIZE", 5*1024*1024*1024),
		ClientCacheMaxAge: envIntOrDefault("CLIENT_CACHE_MAX_AGE", 3600),
		RateLimitWindow:   envDurationOrDefault("RATE_LIMIT_WINDOW", time.Minute),
		RateLimitMax:      envIntOrDefault("RATE_LIMIT_MAX", 60),
	}
	return cfg, nil
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envDurationOrDefault(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}

func envIntOrDefault(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func envBytesOrDefault(key string, fallback int64) int64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	v = strings.TrimSpace(strings.ToUpper(v))
	multiplier := int64(1)
	if strings.HasSuffix(v, "GB") {
		multiplier = 1024 * 1024 * 1024
		v = strings.TrimSuffix(v, "GB")
	} else if strings.HasSuffix(v, "MB") {
		multiplier = 1024 * 1024
		v = strings.TrimSuffix(v, "MB")
	} else if strings.HasSuffix(v, "KB") {
		multiplier = 1024
		v = strings.TrimSuffix(v, "KB")
	}
	n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
	if err != nil {
		return fallback
	}
	return n * multiplier
}
