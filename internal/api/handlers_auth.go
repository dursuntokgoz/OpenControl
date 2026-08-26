package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/dursuntokgoz/OpenControl/internal/auth"
	"github.com/dursuntokgoz/OpenControl/internal/core"
	"github.com/dursuntokgoz/OpenControl/internal/store"
)

type authDeps struct {
	sessions *auth.SessionManager
	users    core.UserRepo
	audit    core.AuditRepo
}

func newAuthHandler(d authDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
			return
		}
		user, err := d.users.GetByUsername(r.Context(), req.Username)
		if err != nil {
			if err == store.ErrNotFound {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}
		if user.DisabledAt != nil {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "account disabled"})
			return
		}
		if err := auth.VerifyPassword(req.Password, user.PasswordHash); err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
			return
		}
		sess, err := d.sessions.Issue(r.Context(), user.ID, extractIP(r), r.UserAgent())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "session creation failed"})
			return
		}
		_ = d.audit.Write(r.Context(), &core.AuditEntry{
			At: time.Now(), ActorUserID: &user.ID, ActorUsername: user.Username,
			ActorIP: extractIP(r), Action: "auth.login", TargetType: "user", TargetID: user.Username,
		})
		http.SetCookie(w, &http.Cookie{
			Name:     "sp_session",
			Value:    sess.ID,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   int(time.Until(sess.ExpiresAt).Seconds()),
		})
		writeJSON(w, http.StatusOK, map[string]any{
			"userId":    user.ID,
			"username":  user.Username,
			"role":      user.Role,
			"csrfToken": sess.CSRFToken,
		})
	}
}

func handleLogout(mgr *auth.SessionManager, audit core.AuditRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sess := SessionFromContext(r.Context())
		if sess != nil {
			_ = mgr.Revoke(r.Context(), sess.ID)
			user := UserFromContext(r.Context())
			username := ""
			if user != nil {
				username = user.Username
			}
			_ = audit.Write(r.Context(), &core.AuditEntry{
				At: time.Now(), ActorUsername: username,
				ActorIP: extractIP(r), Action: "auth.logout",
			})
		}
		http.SetCookie(w, &http.Cookie{
			Name: "sp_session", Value: "", Path: "/",
			HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: -1,
		})
		writeJSON(w, http.StatusOK, map[string]string{"status": "logged out"})
	}
}

func handleMe(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not authenticated"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":           user.ID,
		"username":     user.Username,
		"email":        user.Email,
		"role":         user.Role,
		"totpEnabled":  user.TOTPEnabled,
		"language":     user.Language,
		"contactEmail": user.ContactEmail,
	})
}

func handleTOTPEnroll(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not authenticated"})
		return
	}
	enr, err := auth.EnrollTOTP(user.Username)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "totp generation failed"})
		return
	}
	user.TOTPSecret = enr.Secret
	writeJSON(w, http.StatusOK, map[string]any{
		"secret":          enr.Secret,
		"provisioningUri": enr.ProvisioningURI,
	})
}

func handleTOTPVerify(users core.UserRepo, audit core.AuditRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := UserFromContext(r.Context())
		if user == nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not authenticated"})
			return
		}
		var req struct {
			Code string `json:"code"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
			return
		}
		if user.TOTPSecret == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "no pending enrollment"})
			return
		}
		if err := auth.ValidateTOTP(user.TOTPSecret, req.Code); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid code"})
			return
		}
		user.TOTPEnabled = true
		if err := users.Update(r.Context(), user); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "update failed"})
			return
		}
		_ = audit.Write(r.Context(), &core.AuditEntry{
			At: time.Now(), ActorUserID: &user.ID, ActorUsername: user.Username,
			ActorIP: extractIP(r), Action: "auth.totp_enable", TargetType: "user", TargetID: user.Username,
		})
		writeJSON(w, http.StatusOK, map[string]string{"status": "2fa enabled"})
	}
}

func extractIP(r *http.Request) string {
	if v := r.Header.Get("X-Forwarded-For"); v != "" {
		return v
	}
	if v := r.Header.Get("X-Real-IP"); v != "" {
		return v
	}
	return r.RemoteAddr
}
