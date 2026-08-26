package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/dursuntokgoz/OpenControl/internal/auth"
	"github.com/dursuntokgoz/OpenControl/internal/core"
	"github.com/dursuntokgoz/OpenControl/internal/store"
	"github.com/go-chi/chi/v5"
)

func registerUserRoutes(r chi.Router, users core.UserRepo, audit core.AuditRepo) {
	r.Route("/api/v1/users", func(r chi.Router) {
		r.Use(RequireRole("admin"))
		r.Get("/", listUsers(users))
		r.Post("/", createUser(users, audit))
		r.Route("/{userId}", func(r chi.Router) {
			r.Get("/", getUser(users))
			r.Put("/", updateUser(users, audit))
			r.Delete("/", deleteUser(users, audit))
		})
	})
}

func listUsers(users core.UserRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit <= 0 || limit > 100 {
			limit = 25
		}
		list, total, err := users.List(r.Context(), offset, limit)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "query failed"})
			return
		}
		type userView struct {
			ID       int64  `json:"id"`
			Username string `json:"username"`
			Email    string `json:"email"`
			Role     string `json:"role"`
			Enabled  bool   `json:"enabled"`
		}
		out := make([]userView, len(list))
		for i, u := range list {
			out[i] = userView{ID: u.ID, Username: u.Username, Email: u.Email, Role: u.Role, Enabled: u.DisabledAt == nil}
		}
		writeJSON(w, http.StatusOK, map[string]any{"users": out, "total": total})
	}
}

func createUser(users core.UserRepo, audit core.AuditRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Username string `json:"username"`
			Email    string `json:"email"`
			Password string `json:"password"`
			Role     string `json:"role"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
			return
		}
		if req.Username == "" || req.Email == "" || req.Password == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "username, email, password required"})
			return
		}
		switch req.Role {
		case "admin", "reseller", "user":
		default:
			req.Role = "user"
		}
		hash, err := auth.HashPassword(req.Password)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "hash failed"})
			return
		}
		u := &core.User{
			Username: req.Username, Email: req.Email, PasswordHash: hash,
			Role: req.Role, Language: "en", CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}
		if err := users.Create(r.Context(), u); err != nil {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "user already exists"})
			return
		}
		actor := UserFromContext(r.Context())
		username := ""
		if actor != nil {
			username = actor.Username
		}
		_ = audit.Write(r.Context(), &core.AuditEntry{
			At: time.Now(), ActorUsername: username,
			ActorIP: extractIP(r), Action: "user.create",
			TargetType: "user", TargetID: req.Username,
		})
		writeJSON(w, http.StatusCreated, map[string]any{"id": u.ID, "username": u.Username})
	}
}

func getUser(users core.UserRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "userId"), 10, 64)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid user id"})
			return
		}
		u, err := users.GetByID(r.Context(), id)
		if err != nil {
			if err == store.ErrNotFound {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"id": u.ID, "username": u.Username, "email": u.Email,
			"role": u.Role, "totpEnabled": u.TOTPEnabled, "language": u.Language,
			"disabledAt": u.DisabledAt,
		})
	}
}

func updateUser(users core.UserRepo, audit core.AuditRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "userId"), 10, 64)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid user id"})
			return
		}
		u, err := users.GetByID(r.Context(), id)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		var req struct {
			Email    string `json:"email"`
			Role     string `json:"role"`
			Password string `json:"password"`
			Disabled *bool  `json:"disabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
			return
		}
		if req.Email != "" {
			u.Email = req.Email
		}
		if req.Role == "admin" || req.Role == "reseller" || req.Role == "user" {
			u.Role = req.Role
		}
		if req.Password != "" {
			hash, err := auth.HashPassword(req.Password)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "hash failed"})
				return
			}
			u.PasswordHash = hash
		}
		if req.Disabled != nil {
			if *req.Disabled {
				now := time.Now()
				u.DisabledAt = &now
			} else {
				u.DisabledAt = nil
			}
		}
		if err := users.Update(r.Context(), u); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "update failed"})
			return
		}
		actor := UserFromContext(r.Context())
		username := ""
		if actor != nil {
			username = actor.Username
		}
		_ = audit.Write(r.Context(), &core.AuditEntry{
			At: time.Now(), ActorUsername: username,
			ActorIP: extractIP(r), Action: "user.update",
			TargetType: "user", TargetID: u.Username,
		})
		writeJSON(w, http.StatusOK, map[string]any{"id": u.ID, "username": u.Username})
	}
}

func deleteUser(users core.UserRepo, audit core.AuditRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "userId"), 10, 64)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid user id"})
			return
		}
		u, err := users.GetByID(r.Context(), id)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		if err := users.Delete(r.Context(), id); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "delete failed"})
			return
		}
		actor := UserFromContext(r.Context())
		username := ""
		if actor != nil {
			username = actor.Username
		}
		_ = audit.Write(r.Context(), &core.AuditEntry{
			At: time.Now(), ActorUsername: username,
			ActorIP: extractIP(r), Action: "user.delete",
			TargetType: "user", TargetID: u.Username,
		})
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	}
}
