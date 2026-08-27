package repology

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"sync"
	"time"
)

// Package represents a single item returned by the Repology API.
type Package struct {
	Repo        string `json:"repo"`
	Name        string `json:"name"`
	BinName     string `json:"binname"`
	SrcName     string `json:"srcname"`
	VisibleName string `json:"visiblename"`
	Version     string `json:"version"`
}

// ResolveName attempts to find the best name field for the package manager.
// If all name fields are empty (which happens sometimes in Repology's JSON),
// it falls back to the master project name extracted from the URL.
func (p Package) ResolveName(urlFallback string) string {
	if p.Name != "" {
		return p.Name
	}
	if p.BinName != "" {
		return p.BinName
	}
	if p.VisibleName != "" {
		return p.VisibleName
	}
	if p.SrcName != "" {
		return p.SrcName
	}
	return urlFallback
}

// Client manages communication with the Repology API, including rate limiting.
type Client struct {
	userAgent  string
	delay      time.Duration
	lastReq    time.Time
	mu         sync.Mutex
	httpClient *http.Client
}

// NewClient returns a new Repology API client.
func NewClient(userAgent string) *Client {
	return &Client{
		userAgent:  userAgent,
		delay:      1 * time.Second, // Repology restricts to 1 request per second
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// Fetch queries Repology for the target architecture name, follows the redirect
// to find the master project, and returns the translated package name for the
// target repository (e.g., "nix_unstable").
// Returns (TranslatedName, MasterProjectName, Error).
func (c *Client) Fetch(archName string, targetRepo string) (string, string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Enforce 1-second rate limit
	elapsed := time.Since(c.lastReq)
	if elapsed < c.delay {
		time.Sleep(c.delay - elapsed)
	}

	url := fmt.Sprintf("https://repology.org/tools/project-by?repo=arch&name_type=binname&target_page=api_v1_project&name=%s", archName)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("User-Agent", c.userAgent)

	c.lastReq = time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return "", "", fmt.Errorf("package '%s' not found on repology", archName)
	}
	if resp.StatusCode != 200 {
		return "", "", fmt.Errorf("API error: unexpected status %d", resp.StatusCode)
	}

	// Extract the "Cheat Code" fallback from the final redirected URL path
	// (e.g., /api/v1/project/python -> python)
	masterProjectName := path.Base(resp.Request.URL.Path)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read response body: %w", err)
	}

	var pkgs []Package
	if err := json.Unmarshal(body, &pkgs); err != nil {
		return "", "", fmt.Errorf("JSON parse error: %w", err)
	}

	for _, pkg := range pkgs {
		if pkg.Repo == targetRepo {
			return pkg.ResolveName(masterProjectName), masterProjectName, nil
		}
	}

	return "", masterProjectName, fmt.Errorf("project '%s' found, but has no equivalent in '%s'", archName, targetRepo)
}
