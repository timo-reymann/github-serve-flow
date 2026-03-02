package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHealthEndpoint(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()
	s.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestArtifactServing(t *testing.T) {
	var zipBuf bytes.Buffer
	zw := zip.NewWriter(&zipBuf)
	fw, _ := zw.Create("index.html")
	fw.Write([]byte("<html>test</html>"))
	zw.Close()

	ghServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/allowed/repo/actions/runs/1/artifacts":
			json.NewEncoder(w).Encode(artifactsResponse{
				Artifacts: []artifact{{ID: 42, Name: "site"}},
			})
		case "/repos/allowed/repo/actions/artifacts/42/zip":
			w.Write(zipBuf.Bytes())
		default:
			http.NotFound(w, r)
		}
	}))
	defer ghServer.Close()

	s := newTestServerWithGitHub(t, ghServer.URL, map[string]bool{"allowed": true})

	req := httptest.NewRequest("GET", "/allowed/repo/actions/runs/1/artifacts/site/index.html", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	rec := httptest.NewRecorder()
	s.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q, want text/html; charset=utf-8", ct)
	}
	if rec.Body.String() != "<html>test</html>" {
		t.Errorf("body = %q, want %q", rec.Body.String(), "<html>test</html>")
	}
}

func TestOwnerNotAllowed(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest("GET", "/blocked/repo/actions/runs/1/artifacts/site/index.html", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	rec := httptest.NewRecorder()
	s.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func newTestServer(t *testing.T) *Server {
	t.Helper()
	return newTestServerWithGitHub(t, "http://localhost:0", map[string]bool{"allowed": true})
}

func newTestServerWithGitHub(t *testing.T, ghURL string, owners map[string]bool) *Server {
	t.Helper()
	cacheDir := t.TempDir()
	cfg := &Config{
		GitHubToken:       "test-token",
		AllowedOwners:     owners,
		ListenAddr:        ":0",
		CacheDir:          cacheDir,
		CacheTTL:          time.Hour,
		CacheMaxSize:      1024 * 1024 * 1024,
		ClientCacheMaxAge: 3600,
		RateLimitWindow:   time.Minute,
		RateLimitMax:      1000,
	}
	return NewServer(cfg, ghURL)
}
