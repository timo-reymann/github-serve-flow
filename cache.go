package main

import (
	"os"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

type Cache struct {
	dir       string
	ttl       time.Duration
	maxSize   int64
	totalSize atomic.Int64
	mu        sync.Mutex // protects eviction
}

type cacheEntry struct {
	path    string
	modTime time.Time
	size    int64
}

func NewCache(dir string, ttl time.Duration, maxSize int64) *Cache {
	c := &Cache{dir: dir, ttl: ttl, maxSize: maxSize}
	c.reconcileSize()
	return c
}

func (c *Cache) ArtifactPath(owner, repo, runID, artifactName string) string {
	return filepath.Join(c.dir, owner, repo, runID, artifactName)
}

func (c *Cache) Exists(owner, repo, runID, artifactName string) bool {
	p := c.ArtifactPath(owner, repo, runID, artifactName)
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}

func (c *Cache) TrackSize(bytes int64) {
	c.totalSize.Add(bytes)
}

func (c *Cache) TotalSize() int64 {
	return c.totalSize.Load()
}

func (c *Cache) EvictExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	cutoff := time.Now().Add(-c.ttl)
	entries := c.listEntries()
	for _, e := range entries {
		if e.modTime.Before(cutoff) {
			os.RemoveAll(e.path)
			c.totalSize.Add(-e.size)
		}
	}
}

func (c *Cache) EvictForSize(needed int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.totalSize.Load()+needed <= c.maxSize {
		return
	}

	entries := c.listEntries()
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].modTime.Before(entries[j].modTime)
	})

	for _, e := range entries {
		if c.totalSize.Load()+needed <= c.maxSize {
			break
		}
		os.RemoveAll(e.path)
		c.totalSize.Add(-e.size)
	}
}

func (c *Cache) StartEvictionLoop(stop <-chan struct{}) {
	ticker := time.NewTicker(c.ttl / 2)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.EvictExpired()
		case <-stop:
			return
		}
	}
}

func (c *Cache) listEntries() []cacheEntry {
	var entries []cacheEntry
	owners, _ := os.ReadDir(c.dir)
	for _, owner := range owners {
		if !owner.IsDir() {
			continue
		}
		repos, _ := os.ReadDir(filepath.Join(c.dir, owner.Name()))
		for _, repo := range repos {
			if !repo.IsDir() {
				continue
			}
			runs, _ := os.ReadDir(filepath.Join(c.dir, owner.Name(), repo.Name()))
			for _, run := range runs {
				if !run.IsDir() {
					continue
				}
				artifacts, _ := os.ReadDir(filepath.Join(c.dir, owner.Name(), repo.Name(), run.Name()))
				for _, art := range artifacts {
					if !art.IsDir() {
						continue
					}
					artPath := filepath.Join(c.dir, owner.Name(), repo.Name(), run.Name(), art.Name())
					info, err := art.Info()
					if err != nil {
						continue
					}
					size := dirSize(artPath)
					entries = append(entries, cacheEntry{
						path:    artPath,
						modTime: info.ModTime(),
						size:    size,
					})
				}
			}
		}
	}
	return entries
}

func (c *Cache) reconcileSize() {
	var total int64
	filepath.Walk(c.dir, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			total += info.Size()
		}
		return nil
	})
	c.totalSize.Store(total)
}

func dirSize(path string) int64 {
	var size int64
	filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size
}
