package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestGitHubClientListArtifacts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/repo/actions/runs/123/artifacts" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Error("missing or wrong Authorization header")
		}
		json.NewEncoder(w).Encode(artifactsResponse{
			Artifacts: []artifact{
				{ID: 1, Name: "build"},
				{ID: 2, Name: "logs"},
			},
		})
	}))
	defer srv.Close()

	client := NewGitHubClient("test-token", srv.URL)
	id, err := client.FindArtifactID("owner", "repo", "123", "build")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 1 {
		t.Errorf("artifact ID = %d, want 1", id)
	}
}

func TestGitHubClientArtifactNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(artifactsResponse{Artifacts: []artifact{}})
	}))
	defer srv.Close()

	client := NewGitHubClient("test-token", srv.URL)
	_, err := client.FindArtifactID("owner", "repo", "123", "missing")
	if err == nil {
		t.Fatal("expected error for missing artifact")
	}
}

func TestGitHubClientDownloadArtifact(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	fw, _ := zw.Create("index.html")
	fw.Write([]byte("<html>hello</html>"))
	zw.Close()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/owner/repo/actions/artifacts/1/zip" {
			w.Header().Set("Content-Type", "application/zip")
			w.Write(buf.Bytes())
			return
		}
		t.Errorf("unexpected path: %s", r.URL.Path)
	}))
	defer srv.Close()

	client := NewGitHubClient("test-token", srv.URL)
	dest := t.TempDir()
	size, err := client.DownloadAndExtract("owner", "repo", 1, dest)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if size == 0 {
		t.Error("expected non-zero extracted size")
	}

	content, err := os.ReadFile(filepath.Join(dest, "index.html"))
	if err != nil {
		t.Fatalf("expected index.html to exist: %v", err)
	}
	if string(content) != "<html>hello</html>" {
		t.Errorf("content = %q, want %q", content, "<html>hello</html>")
	}
}
