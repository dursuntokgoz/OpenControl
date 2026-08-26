// Package api wires the HTTP surface of panel-api.
package api

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/dursuntokgoz/OpenControl/internal/auth"
	"github.com/dursuntokgoz/OpenControl/internal/core"
)

const (
	sessionCookieName = "sp_session"
	sessionTTL        = 12 * time.Hour
)

// Options configures the HTTP router.
type Options struct {
	WebDist  string // directory with built SPA assets; empty disables static
	Logger   *slog.Logger
	Deps     *core.AppDeps
	Sessions *auth.SessionManager
}

// NewRouter builds the full middleware chain and routes.
func NewRouter(opts Options) http.Handler {
	log := opts.Logger
	if log == nil {
		log = slog.Default()
	}

	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(secureHeaders)
	r.Use(slogRequest(log))
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(30 * time.Second))

	r.Get("/healthz", handleHealth)
	r.Get("/api/v1/ping", handlePing)

	if opts.Deps != nil && opts.Sessions != nil {
		loginRateLimit := NewRateLimiter(10, time.Minute)
		r.With(RateLimit(loginRateLimit)).
			Post("/api/v1/auth/login", newAuthHandler(authDeps{
				sessions: opts.Sessions, users: opts.Deps.Users, audit: opts.Deps.Audit,
			}))

		r.Post("/api/v1/auth/login",
			newAuthHandler(authDeps{
				sessions: opts.Sessions, users: opts.Deps.Users, audit: opts.Deps.Audit,
			}))

		r.With(RequireSession(opts.Sessions, sessionCookieName), InjectUser(opts.Deps.Users)).
			Post("/api/v1/auth/logout", handleLogout(opts.Sessions, opts.Deps.Audit))

		r.With(RequireSession(opts.Sessions, sessionCookieName), InjectUser(opts.Deps.Users)).
			Get("/api/v1/auth/me", handleMe)

		r.With(RequireSession(opts.Sessions, sessionCookieName), InjectUser(opts.Deps.Users), RequireRole("admin", "reseller", "user")).
			Post("/api/v1/auth/2fa/enroll", handleTOTPEnroll)

		r.With(RequireSession(opts.Sessions, sessionCookieName), InjectUser(opts.Deps.Users), RequireCSRF(sessionCookieName)).
			Post("/api/v1/auth/2fa/verify", handleTOTPVerify(opts.Deps.Users, opts.Deps.Audit))

		r.With(RequireSession(opts.Sessions, sessionCookieName), InjectUser(opts.Deps.Users), RequireRole("admin")).
			Get("/api/v1/audit", handleAuditLog(opts.Deps.Audit))

		protected := func(r chi.Router) {
			r.Use(RequireSession(opts.Sessions, sessionCookieName))
			r.Use(InjectUser(opts.Deps.Users))
			registerUserRoutes(r, opts.Deps.Users, opts.Deps.Audit)
			registerPackageRoutes(r, opts.Deps.Packages, opts.Deps.Audit)
			registerAccountRoutes(r, opts.Deps.Accounts, opts.Deps.Packages, opts.Deps.Audit)
		}
		r.Group(protected)
	}

	if opts.WebDist != "" {
		r.Handle("/*", spaHandler(opts.WebDist))
	}
	return r
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"version": core.CurrentBuild().Version,
	})
}

func handlePing(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"pong": true})
}

// writeJSON emits a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(mustJSON(body))
}

// secureHeaders sets the baseline security headers on every response.
func secureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Content-Security-Policy",
			"default-src 'self'; script-src 'self'; style-src 'self'; "+
				"img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; "+
				"base-uri 'none'; form-action 'self'")
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		next.ServeHTTP(w, r)
	})
}

// spaHandler serves static files and falls back to index.html for client-side
// routes without a file extension.
func spaHandler(dir string) http.Handler {
	fs := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		if !hasFileExt(path) {
			r2 := new(http.Request)
			*r2 = *r
			r2.URL.Path = "/"
			fs.ServeHTTP(w, r2)
			return
		}
		fs.ServeHTTP(w, r)
	})
}

func hasFileExt(p string) bool {
	if i := strings.LastIndexByte(p, '.'); i >= 0 {
		return !strings.ContainsAny(p[i+1:], "/\\")
	}
	return false
}
