package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCacheStoreAndGet(t *testing.T) {
	dir := t.TempDir()
	c := NewCache(dir, time.Hour, 1024*1024*1024)

	artifactDir := c.ArtifactPath("owner", "repo", "123", "build")
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artifactDir, "index.html"), []byte("<html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	c.TrackSize(6) // len("<html>")

	got := c.ArtifactPath("owner", "repo", "123", "build")
	if _, err := os.Stat(filepath.Join(got, "index.html")); err != nil {
		t.Errorf("expected cached file to exist: %v", err)
	}
}

func TestCacheExists(t *testing.T) {
	dir := t.TempDir()
	c := NewCache(dir, time.Hour, 1024*1024*1024)

	if c.Exists("owner", "repo", "123", "build") {
		t.Error("expected cache miss for nonexistent artifact")
	}

	artifactDir := c.ArtifactPath("owner", "repo", "123", "build")
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if !c.Exists("owner", "repo", "123", "build") {
		t.Error("expected cache hit for existing artifact")
	}
}

func TestCacheEvictByTTL(t *testing.T) {
	dir := t.TempDir()
	c := NewCache(dir, 50*time.Millisecond, 1024*1024*1024)

	artifactDir := c.ArtifactPath("owner", "repo", "123", "build")
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artifactDir, "f.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	c.TrackSize(2)

	time.Sleep(100 * time.Millisecond)
	c.EvictExpired()

	if c.Exists("owner", "repo", "123", "build") {
		t.Error("expected artifact to be evicted after TTL")
	}
}

func TestCacheEvictBySize(t *testing.T) {
	dir := t.TempDir()
	c := NewCache(dir, time.Hour, 10) // 10 bytes max

	// Store a 6-byte artifact
	art1 := c.ArtifactPath("owner", "repo", "1", "a")
	if err := os.MkdirAll(art1, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(art1, "f.txt"), []byte("123456"), 0o644); err != nil {
		t.Fatal(err)
	}
	c.TrackSize(6)

	// Evict to make room for 6 more bytes (total would be 12 > 10)
	c.EvictForSize(6)

	if c.Exists("owner", "repo", "1", "a") {
		t.Error("expected oldest artifact to be evicted for size")
	}
}

func TestCacheTotalSize(t *testing.T) {
	dir := t.TempDir()
	c := NewCache(dir, time.Hour, 1024*1024*1024)

	c.TrackSize(100)
	c.TrackSize(200)
	if c.TotalSize() != 300 {
		t.Errorf("TotalSize = %d, want 300", c.TotalSize())
	}
}
