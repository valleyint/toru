package repology

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClient_Fetch(t *testing.T) {
	// Mock Repology Server
	mux := http.NewServeMux()
	
	// Mock the exact redirect behavior Repology uses
	mux.HandleFunc("/tools/project-by", func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		if name == "missing-pkg" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		
		// Redirect to the master project page
		masterName := name
		if name == "python" {
			masterName = "python3" // simulate a rename
		}
		
		redirectURL := fmt.Sprintf("/api/v1/project/%s", masterName)
		http.Redirect(w, r, redirectURL, http.StatusFound)
	})

	mux.HandleFunc("/api/v1/project/python3", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		// Simulate Repology JSON response
		w.Write([]byte(`[
			{"repo": "arch", "name": "python", "binname": "python", "version": "3.10"},
			{"repo": "nix_unstable", "name": "python3", "binname": "python3", "version": "3.10"}
		]`))
	})
	
	mux.HandleFunc("/api/v1/project/weird-pkg", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		// Simulate Repology JSON response with EMPTY name fields (requires cheat code fallback)
		w.Write([]byte(`[
			{"repo": "nix_unstable", "name": "", "binname": "", "version": "1.0"}
		]`))
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	client := NewClient("toru-test/1.0")
	// Reduce delay for tests so they run fast
	client.delay = 10 * time.Millisecond 

	// Hack to rewrite the hardcoded URL in the client to hit our test server instead
	// In Go, since we can't easily monkey-patch the hardcoded URL without refactoring, 
	// we will intercept it at the transport layer for the test.
	client.httpClient.Transport = &rewriteTransport{serverURL: ts.URL}

	// 1. Test Successful Translation
	translated, master, err := client.Fetch("python", "nix_unstable")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if translated != "python3" || master != "python3" {
		t.Errorf("got %s, %s; want python3, python3", translated, master)
	}

	// 2. Test Cheat Code Fallback (empty JSON names)
	translated, master, err = client.Fetch("weird-pkg", "nix_unstable")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if translated != "weird-pkg" || master != "weird-pkg" {
		t.Errorf("got %s, %s; want weird-pkg, weird-pkg", translated, master)
	}

	// 3. Test Missing Package
	_, _, err = client.Fetch("missing-pkg", "nix_unstable")
	if err == nil {
		t.Fatalf("expected error for missing package")
	}
}

// rewriteTransport intercepts requests and routes them to the test server
type rewriteTransport struct {
	serverURL string
}

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Re-route the request to our httptest server while keeping the path and query
	req.URL.Scheme = "http"
	req.URL.Host = t.serverURL[7:] // strip "http://"
	return http.DefaultTransport.RoundTrip(req)
}
