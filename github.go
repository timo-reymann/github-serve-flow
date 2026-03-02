package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type GitHubClient struct {
	token   string
	baseURL string
	client  *http.Client
}

type artifactsResponse struct {
	Artifacts []artifact `json:"artifacts"`
}

type artifact struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

func NewGitHubClient(token, baseURL string) *GitHubClient {
	if baseURL == "" {
		baseURL = "https://api.github.com"
	}
	return &GitHubClient{
		token:   token,
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{},
	}
}

func (g *GitHubClient) FindArtifactID(owner, repo, runID, name string) (int64, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/actions/runs/%s/artifacts", g.baseURL, owner, repo, runID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+g.token)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := g.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("list artifacts request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var result artifactsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("decode artifacts response: %w", err)
	}

	for _, a := range result.Artifacts {
		if a.Name == name {
			return a.ID, nil
		}
	}
	return 0, fmt.Errorf("artifact %q not found in run %s", name, runID)
}

func (g *GitHubClient) DownloadAndExtract(owner, repo string, artifactID int64, dest string) (int64, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/actions/artifacts/%d/zip", g.baseURL, owner, repo, artifactID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+g.token)

	resp, err := g.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("download artifact request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("read artifact body: %w", err)
	}

	reader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return 0, fmt.Errorf("open zip: %w", err)
	}

	var totalSize int64
	for _, f := range reader.File {
		targetPath := filepath.Join(dest, f.Name)

		// Prevent zip slip
		if !strings.HasPrefix(filepath.Clean(targetPath), filepath.Clean(dest)+string(os.PathSeparator)) {
			continue
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(targetPath, 0o755)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return 0, err
		}

		rc, err := f.Open()
		if err != nil {
			return 0, err
		}

		out, err := os.Create(targetPath)
		if err != nil {
			rc.Close()
			return 0, err
		}

		n, err := io.Copy(out, rc)
		out.Close()
		rc.Close()
		if err != nil {
			return 0, err
		}
		totalSize += n
	}

	return totalSize, nil
}
