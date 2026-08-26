package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestHealthEndpoint(t *testing.T) {
	handler := NewRouter(Options{})
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body struct {
		Status  string `json:"status"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body %q: %v", rec.Body.String(), err)
	}
	if body.Status != "ok" {
		t.Errorf("status = %q, want ok", body.Status)
	}
}

func TestSecurityHeadersPresent(t *testing.T) {
	handler := NewRouter(Options{})
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	want := map[string]string{
		"X-Content-Type-Options":  "nosniff",
		"X-Frame-Options":         "DENY",
		"Content-Security-Policy": "",
	}
	for k := range want {
		if rec.Header().Get(k) == "" {
			t.Errorf("missing security header %s", k)
		}
	}
}

func TestPingEndpoint(t *testing.T) {
	handler := NewRouter(Options{})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["pong"] != true {
		t.Errorf("pong = %v", body["pong"])
	}
}

// buildFakeDist creates a minimal SPA directory.
func buildFakeDist(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"index.html": "<!doctype html><html><body>ServerPanel</body></html>",
		"app.js":     "console.log(1)",
	}
	for name, content := range files {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestSPAHandlerServesIndexForClientRoutes(t *testing.T) {
	dist := buildFakeDist(t)
	handler := NewRouter(Options{WebDist: dist})

	// Client-side route without extension must fall back to index.html.
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/accounts/list", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("client route status = %d", rec.Code)
	}
	if got := rec.Body.String(); got == "" || !contains(got, "ServerPanel") {
		t.Errorf("expected index.html content, got %q", got)
	}

	// Real assets are served directly with their path.
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec2.Code != http.StatusOK || !contains(rec2.Body.String(), "console.log") {
		t.Fatalf("asset not served correctly (code=%d)", rec2.Code)
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && indexOf(haystack, needle) >= 0
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
