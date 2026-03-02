package main

import (
	"fmt"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Server struct {
	cfg      *Config
	cache    *Cache
	github   *GitHubClient
	handler  http.Handler
	inflight sync.Map // artifact key -> *sync.Once for dedup
}

func NewServer(cfg *Config, githubBaseURL string) *Server {
	s := &Server{
		cfg:    cfg,
		cache:  NewCache(cfg.CacheDir, cfg.CacheTTL, cfg.CacheMaxSize),
		github: NewGitHubClient(cfg.GitHubToken, githubBaseURL),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /{owner}/{repo}/actions/runs/{runId}/artifacts/{artifactName}/{filePath...}", s.handleArtifact)

	rl := NewRateLimiter(cfg.RateLimitWindow, cfg.RateLimitMax)
	s.handler = rl.Middleware(mux)
	return s
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func (s *Server) handleArtifact(w http.ResponseWriter, r *http.Request) {
	owner := strings.ToLower(r.PathValue("owner"))
	repo := r.PathValue("repo")
	runID := r.PathValue("runId")
	artifactName := r.PathValue("artifactName")
	filePath := r.PathValue("filePath")

	if !s.cfg.AllowedOwners[owner] {
		http.Error(w, "owner not allowed", http.StatusForbidden)
		return
	}

	if !s.cache.Exists(owner, repo, runID, artifactName) {
		if err := s.fetchArtifact(owner, repo, runID, artifactName); err != nil {
			log.Printf("error fetching artifact: %v", err)
			http.Error(w, "failed to fetch artifact", http.StatusBadGateway)
			return
		}
	}

	target := filepath.Join(s.cache.ArtifactPath(owner, repo, runID, artifactName), filePath)

	// Prevent path traversal
	cleanTarget := filepath.Clean(target)
	cacheBase := filepath.Clean(s.cache.ArtifactPath(owner, repo, runID, artifactName))
	if !strings.HasPrefix(cleanTarget, cacheBase+string(os.PathSeparator)) && cleanTarget != cacheBase {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	info, err := os.Stat(cleanTarget)
	if err != nil {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}
	if info.IsDir() {
		indexPath := filepath.Join(cleanTarget, "index.html")
		if _, err := os.Stat(indexPath); err == nil {
			cleanTarget = indexPath
		} else {
			http.Error(w, "file not found", http.StatusNotFound)
			return
		}
	}

	f, err := os.Open(cleanTarget)
	if err != nil {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}

	ext := filepath.Ext(cleanTarget)
	ct := mime.TypeByExtension(ext)
	if ct == "" {
		buf := make([]byte, 512)
		n, _ := f.Read(buf)
		ct = http.DetectContentType(buf[:n])
		f.Seek(0, 0)
	}
	if ct != "" {
		w.Header().Set("Content-Type", ct)
	}

	w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", s.cfg.ClientCacheMaxAge))
	http.ServeContent(w, r, cleanTarget, fi.ModTime(), f)
}

func (s *Server) fetchArtifact(owner, repo, runID, artifactName string) error {
	key := strings.Join([]string{owner, repo, runID, artifactName}, "/")

	actual, loaded := s.inflight.LoadOrStore(key, &sync.Once{})
	once := actual.(*sync.Once)

	var fetchErr error
	once.Do(func() {
		defer s.inflight.Delete(key)

		id, err := s.github.FindArtifactID(owner, repo, runID, artifactName)
		if err != nil {
			fetchErr = err
			return
		}

		dest := s.cache.ArtifactPath(owner, repo, runID, artifactName)
		if err := os.MkdirAll(dest, 0o755); err != nil {
			fetchErr = err
			return
		}

		s.cache.EvictForSize(0)
		size, err := s.github.DownloadAndExtract(owner, repo, id, dest)
		if err != nil {
			os.RemoveAll(dest)
			fetchErr = err
			return
		}

		s.cache.TrackSize(size)
		s.cache.EvictForSize(0)
	})

	if !loaded && fetchErr != nil {
		return fetchErr
	}

	if loaded {
		if !s.cache.Exists(owner, repo, runID, artifactName) {
			return fmt.Errorf("artifact fetch by another goroutine failed")
		}
	}

	return nil
}

func (s *Server) ListenAndServe() error {
	stop := make(chan struct{})
	go s.cache.StartEvictionLoop(stop)

	log.Printf("listening on %s", s.cfg.ListenAddr)
	return http.ListenAndServe(s.cfg.ListenAddr, s.handler)
}
