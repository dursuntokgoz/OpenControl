package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/dursuntokgoz/OpenControl/internal/auth"
	"github.com/dursuntokgoz/OpenControl/internal/core"
)

type contextKey string

const (
	sessionCtxKey     contextKey = "session"
	currentUserCtxKey contextKey = "currentUser"
)

// SessionFromContext returns the session stored in the request context.
func SessionFromContext(ctx context.Context) *auth.Session {
	s, _ := ctx.Value(sessionCtxKey).(*auth.Session)
	return s
}

// UserFromContext returns the authenticated user stored in the request context.
func UserFromContext(ctx context.Context) *core.User {
	u, _ := ctx.Value(currentUserCtxKey).(*core.User)
	return u
}

// RequireSession extracts and validates the session cookie, storing it in context.
func RequireSession(mgr *auth.SessionManager, cookieName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(cookieName)
			if err != nil || cookie.Value == "" {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			sess, err := mgr.Validate(r.Context(), cookie.Value)
			if err != nil {
				http.Error(w, `{"error":"session expired"}`, http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), sessionCtxKey, sess)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// InjectUser loads the full User record from the store and puts it in context.
// Must be chained AFTER RequireSession.
func InjectUser(repo core.UserRepo) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sess := SessionFromContext(r.Context())
			if sess == nil {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			user, err := repo.GetByID(r.Context(), sess.UserID)
			if err != nil {
				http.Error(w, `{"error":"user not found"}`, http.StatusUnauthorized)
				return
			}
			if user.DisabledAt != nil {
				http.Error(w, `{"error":"account disabled"}`, http.StatusForbidden)
				return
			}
			ctx := context.WithValue(r.Context(), currentUserCtxKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole restricts access to users with one of the listed roles.
// Must be chained AFTER InjectUser.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(roles))
	for _, role := range roles {
		allowed[strings.ToLower(role)] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := UserFromContext(r.Context())
			if user == nil {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			if !allowed[strings.ToLower(user.Role)] {
				http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireCSRF validates the X-CSRF-Token header against the session's CSRF token.
// Must be chained AFTER RequireSession.
func RequireCSRF(cookieName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sess := SessionFromContext(r.Context())
			if sess == nil {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			header := r.Header.Get("X-CSRF-Token")
			if header == "" || header != sess.CSRFToken {
				http.Error(w, `{"error":"csrf token mismatch"}`, http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
