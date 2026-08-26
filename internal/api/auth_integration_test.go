package api_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dursuntokgoz/OpenControl/internal/api"
	"github.com/dursuntokgoz/OpenControl/internal/auth"
	"github.com/dursuntokgoz/OpenControl/internal/core"
	"github.com/dursuntokgoz/OpenControl/internal/store"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	sdb := &store.DB{DB: db, Driver: "sqlite"}
	if err := sdb.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	return db
}

func bootstrapAdmin(t *testing.T, db *sql.DB, username, password string) {
	t.Helper()
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO users (username, email, password_hash, role, language, created_at, updated_at)
		VALUES (?, ?, ?, 'admin', 'en', ?, ?)`,
		username, username+"@test.local", hash, time.Now(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
}

func TestFullLoginFlow(t *testing.T) {
	db := setupTestDB(t)
	bootstrapAdmin(t, db, "admin", "admin123")

	userRepo := store.NewUserRepo(db)
	auditRepo := store.NewAuditRepo(db)
	sessionRepo := store.NewSessionStore(db)
	sessMgr := auth.NewSessionManager(sessionRepo, time.Hour)
	pkgRepo := store.NewPackageRepo(db)
	acctRepo := store.NewAccountRepo(db)

	handler := api.NewRouter(api.Options{
		Sessions: sessMgr,
		Deps: &core.AppDeps{
			Users: userRepo, Audit: auditRepo, Packages: pkgRepo, Accounts: acctRepo,
		},
	})

	// 1. Login with bad credentials → 401
	body := marshal(t, map[string]string{"username": "admin", "password": "wrong"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("bad login: status = %d, want 401", rec.Code)
	}

	// 2. Login with correct credentials → 200 + session cookie + csrf token
	body = marshal(t, map[string]string{"username": "admin", "password": "admin123"})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("login: status = %d, want 200, body: %s", rec.Code, rec.Body.String())
	}
	var loginResp struct {
		CSRFToken string `json:"csrfToken"`
		Role      string `json:"role"`
		Username  string `json:"username"`
	}
	mustDecode(t, rec.Body.Bytes(), &loginResp)
	if loginResp.Role != "admin" || loginResp.Username != "admin" {
		t.Fatalf("unexpected login response: %+v", loginResp)
	}

	// Extract session cookie
	cookie := extractCookie(rec, "sp_session")
	if cookie == "" {
		t.Fatal("no sp_session cookie in response")
	}

	// 3. /api/v1/auth/me without cookie → 401
	req = httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("me without cookie: status = %d, want 401", rec.Code)
	}

	// 4. /api/v1/auth/me with cookie → 200
	req = httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: "sp_session", Value: cookie})
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("me with cookie: status = %d, body: %s", rec.Code, rec.Body.String())
	}
	var meResp struct {
		Username string `json:"username"`
		Role     string `json:"role"`
	}
	mustDecode(t, rec.Body.Bytes(), &meResp)
	if meResp.Username != "admin" || meResp.Role != "admin" {
		t.Fatalf("unexpected me response: %+v", meResp)
	}

	// 5. Create a package
	body = marshal(t, map[string]any{
		"name": "starter", "diskMb": 1024, "bandwidthMb": 10240,
		"maxDomains": 5, "maxDatabases": 2, "maxMailboxes": 10, "maxFtp": 2, "phpVersion": "8.3",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/packages", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "sp_session", Value: cookie})
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create package: status = %d, body: %s", rec.Code, rec.Body.String())
	}
	var pkgResp struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}
	mustDecode(t, rec.Body.Bytes(), &pkgResp)
	if pkgResp.Name != "starter" {
		t.Fatalf("unexpected package: %+v", pkgResp)
	}

	// 6. Create a hosting account
	body = marshal(t, map[string]any{
		"username": "user1", "primaryDomain": "example.com", "packageId": pkgResp.ID,
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/accounts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "sp_session", Value: cookie})
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create account: status = %d, body: %s", rec.Code, rec.Body.String())
	}

	// 7. Verify audit log has entries
	req = httptest.NewRequest(http.MethodGet, "/api/v1/audit?limit=10", nil)
	req.AddCookie(&http.Cookie{Name: "sp_session", Value: cookie})
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("audit log: status = %d", rec.Code)
	}
	var auditResp struct {
		Entries []core.AuditEntry `json:"entries"`
		Total   int64             `json:"total"`
	}
	mustDecode(t, rec.Body.Bytes(), &auditResp)
	if auditResp.Total < 3 { // login + pkg.create + acct.create
		t.Fatalf("expected at least 3 audit entries, got %d", auditResp.Total)
	}
	for _, e := range auditResp.Entries {
		if e.ActorUsername != "admin" {
			t.Errorf("unexpected actor: %s", e.ActorUsername)
		}
	}
}

func TestRBACEnforcement(t *testing.T) {
	db := setupTestDB(t)
	bootstrapAdmin(t, db, "admin", "admin123")

	// Create a regular user
	hash, _ := auth.HashPassword("user123")
	_, err := db.Exec(`INSERT INTO users (username, email, password_hash, role, language, created_at, updated_at)
		VALUES ('regular', 'regular@test.local', ?, 'user', 'en', ?, ?)`,
		hash, time.Now(), time.Now())
	if err != nil {
		t.Fatal(err)
	}

	userRepo := store.NewUserRepo(db)
	auditRepo := store.NewAuditRepo(db)
	sessionRepo := store.NewSessionStore(db)
	sessMgr := auth.NewSessionManager(sessionRepo, time.Hour)

	handler := api.NewRouter(api.Options{
		Sessions: sessMgr,
		Deps: &core.AppDeps{
			Users: userRepo, Audit: auditRepo,
			Packages: store.NewPackageRepo(db),
			Accounts: store.NewAccountRepo(db),
		},
	})

	// Login as regular user
	cookie := loginAs(t, handler, "regular", "user123")

	// Regular user should NOT be able to list users
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req.AddCookie(&http.Cookie{Name: "sp_session", Value: cookie})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("regular user accessing /users: status = %d, want 403", rec.Code)
	}

	// Regular user should NOT be able to create packages
	body := marshal(t, map[string]any{"name": "test"})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/packages", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "sp_session", Value: cookie})
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("regular user creating package: status = %d, want 403", rec.Code)
	}

	// Regular user SHOULD be able to access /auth/me
	req = httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: "sp_session", Value: cookie})
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("regular user /me: status = %d, want 200", rec.Code)
	}
}

func TestRateLimiterRejectsExcess(t *testing.T) {
	limiter := api.NewRateLimiter(3, time.Minute)
	for i := 0; i < 3; i++ {
		if !limiter.Allow("test-ip") {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}
	if limiter.Allow("test-ip") {
		t.Fatal("4th request should be rate-limited")
	}
	// Different IP should still be allowed.
	if !limiter.Allow("other-ip") {
		t.Fatal("different IP should not be rate-limited")
	}
}

func loginAs(t *testing.T, handler http.Handler, user, pass string) string {
	t.Helper()
	body := marshal(t, map[string]string{"username": user, "password": pass})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("login as %s failed: %d %s", user, rec.Code, rec.Body.String())
	}
	c := extractCookie(rec, "sp_session")
	if c == "" {
		t.Fatal("no session cookie")
	}
	return c
}

func marshal(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func mustDecode(t *testing.T, data []byte, v any) {
	t.Helper()
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatalf("json decode: %v (data: %s)", err, string(data))
	}
}

func extractCookie(rec *httptest.ResponseRecorder, name string) string {
	for _, c := range rec.Result().Cookies() {
		if c.Name == name {
			return c.Value
		}
	}
	return ""
}

// Ensure json.Marshal is used (prevents unused import).
var _ = bytes.Buffer{}
