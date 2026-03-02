package main

import (
	"testing"
	"time"
)

func TestLoadConfigDefaults(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "test-token")
	t.Setenv("ALLOWED_OWNERS", "owner1,owner2")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.GitHubToken != "test-token" {
		t.Errorf("GitHubToken = %q, want %q", cfg.GitHubToken, "test-token")
	}
	if len(cfg.AllowedOwners) != 2 || cfg.AllowedOwners["owner1"] != true || cfg.AllowedOwners["owner2"] != true {
		t.Errorf("AllowedOwners = %v, want {owner1, owner2}", cfg.AllowedOwners)
	}
	if cfg.ListenAddr != ":8080" {
		t.Errorf("ListenAddr = %q, want %q", cfg.ListenAddr, ":8080")
	}
	if cfg.CacheTTL != time.Hour {
		t.Errorf("CacheTTL = %v, want %v", cfg.CacheTTL, time.Hour)
	}
	if cfg.CacheMaxSize != 5*1024*1024*1024 {
		t.Errorf("CacheMaxSize = %d, want %d", cfg.CacheMaxSize, 5*1024*1024*1024)
	}
	if cfg.ClientCacheMaxAge != 3600 {
		t.Errorf("ClientCacheMaxAge = %d, want %d", cfg.ClientCacheMaxAge, 3600)
	}
	if cfg.RateLimitWindow != time.Minute {
		t.Errorf("RateLimitWindow = %v, want %v", cfg.RateLimitWindow, time.Minute)
	}
	if cfg.RateLimitMax != 60 {
		t.Errorf("RateLimitMax = %d, want %d", cfg.RateLimitMax, 60)
	}
}

func TestLoadConfigMissingRequired(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("ALLOWED_OWNERS", "")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error for missing required env vars")
	}
}

func TestLoadConfigCustomValues(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "tok")
	t.Setenv("ALLOWED_OWNERS", "acme")
	t.Setenv("LISTEN_ADDR", ":9090")
	t.Setenv("CACHE_TTL", "30m")
	t.Setenv("CACHE_MAX_SIZE", "1GB")
	t.Setenv("CLIENT_CACHE_MAX_AGE", "7200")
	t.Setenv("RATE_LIMIT_WINDOW", "5m")
	t.Setenv("RATE_LIMIT_MAX", "100")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ListenAddr != ":9090" {
		t.Errorf("ListenAddr = %q, want %q", cfg.ListenAddr, ":9090")
	}
	if cfg.CacheTTL != 30*time.Minute {
		t.Errorf("CacheTTL = %v, want %v", cfg.CacheTTL, 30*time.Minute)
	}
	if cfg.CacheMaxSize != 1*1024*1024*1024 {
		t.Errorf("CacheMaxSize = %d, want %d", cfg.CacheMaxSize, 1*1024*1024*1024)
	}
	if cfg.RateLimitWindow != 5*time.Minute {
		t.Errorf("RateLimitWindow = %v, want %v", cfg.RateLimitWindow, 5*time.Minute)
	}
	if cfg.RateLimitMax != 100 {
		t.Errorf("RateLimitMax = %d, want %d", cfg.RateLimitMax, 100)
	}
}
