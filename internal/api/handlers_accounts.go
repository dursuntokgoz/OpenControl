package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/dursuntokgoz/OpenControl/internal/core"
	"github.com/dursuntokgoz/OpenControl/internal/store"
	"github.com/go-chi/chi/v5"
)

func registerAccountRoutes(r chi.Router, accounts core.AccountRepo, pkgs core.PackageRepo, audit core.AuditRepo) {
	r.Route("/api/v1/accounts", func(r chi.Router) {
		r.Get("/", listAccounts(accounts))
		r.With(RequireRole("admin", "reseller")).Post("/", createAccount(accounts, pkgs, audit))
		r.Route("/{acctId}", func(r chi.Router) {
			r.Get("/", getAccount(accounts))
			r.With(RequireRole("admin", "reseller")).Put("/", updateAccount(accounts, audit))
			r.With(RequireRole("admin")).Delete("/", deleteAccount(accounts, audit))
		})
	})
}

func listAccounts(accounts core.AccountRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit <= 0 || limit > 100 {
			limit = 25
		}
		user := UserFromContext(r.Context())
		if user == nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not authenticated"})
			return
		}
		var list []core.Account
		var total int64
		var err error
		if user.IsAdmin() {
			list, total, err = accounts.List(r.Context(), offset, limit)
		} else {
			list, total, err = accounts.ListByOwner(r.Context(), user.ID, offset, limit)
		}
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "query failed"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"accounts": list, "total": total})
	}
}

func createAccount(accounts core.AccountRepo, pkgs core.PackageRepo, audit core.AuditRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Username      string `json:"username"`
			PrimaryDomain string `json:"primaryDomain"`
			PackageID     int64  `json:"packageId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
			return
		}
		if req.Username == "" || req.PrimaryDomain == "" || req.PackageID == 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "username, primaryDomain, packageId required"})
			return
		}
		if _, err := pkgs.GetByID(r.Context(), req.PackageID); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "package not found"})
			return
		}
		user := UserFromContext(r.Context())
		a := &core.Account{
			Username: req.Username, PrimaryDomain: req.PrimaryDomain,
			OwnerUserID: user.ID, PackageID: req.PackageID,
			HomeDir: "/home/" + req.Username, Status: "active", CreatedAt: time.Now(),
		}
		if err := accounts.Create(r.Context(), a); err != nil {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "account already exists"})
			return
		}
		_ = audit.Write(r.Context(), &core.AuditEntry{
			At: time.Now(), ActorUserID: &user.ID, ActorUsername: user.Username,
			ActorIP: extractIP(r), Action: "account.create",
			TargetType: "account", TargetID: req.Username,
		})
		writeJSON(w, http.StatusCreated, map[string]any{"id": a.ID, "username": a.Username})
	}
}

func getAccount(accounts core.AccountRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "acctId"), 10, 64)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid account id"})
			return
		}
		a, err := accounts.GetByID(r.Context(), id)
		if err != nil {
			if err == store.ErrNotFound {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
			return
		}
		writeJSON(w, http.StatusOK, a)
	}
}

func updateAccount(accounts core.AccountRepo, audit core.AuditRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "acctId"), 10, 64)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid account id"})
			return
		}
		a, err := accounts.GetByID(r.Context(), id)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		user := UserFromContext(r.Context())
		if !user.IsAdmin() && a.OwnerUserID != user.ID {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "not your account"})
			return
		}
		var req struct {
			PrimaryDomain string `json:"primaryDomain"`
			PackageID     int64  `json:"packageId"`
			Status        string `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
			return
		}
		if req.PrimaryDomain != "" {
			a.PrimaryDomain = req.PrimaryDomain
		}
		if req.PackageID > 0 {
			a.PackageID = req.PackageID
		}
		if req.Status == "active" || req.Status == "suspended" {
			a.Status = req.Status
		}
		if err := accounts.Update(r.Context(), a); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "update failed"})
			return
		}
		_ = audit.Write(r.Context(), &core.AuditEntry{
			At: time.Now(), ActorUserID: &user.ID, ActorUsername: user.Username,
			ActorIP: extractIP(r), Action: "account.update",
			TargetType: "account", TargetID: a.Username,
		})
		writeJSON(w, http.StatusOK, a)
	}
}

func deleteAccount(accounts core.AccountRepo, audit core.AuditRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "acctId"), 10, 64)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid account id"})
			return
		}
		a, err := accounts.GetByID(r.Context(), id)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		if err := accounts.Delete(r.Context(), id); err != nil {
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
			ActorIP: extractIP(r), Action: "account.delete",
			TargetType: "account", TargetID: a.Username,
		})
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	}
}
