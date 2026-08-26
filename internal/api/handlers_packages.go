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

func registerPackageRoutes(r chi.Router, pkgs core.PackageRepo, audit core.AuditRepo) {
	r.Route("/api/v1/packages", func(r chi.Router) {
		r.Use(RequireRole("admin"))
		r.Get("/", listPackages(pkgs))
		r.Post("/", createPackage(pkgs, audit))
		r.Route("/{pkgId}", func(r chi.Router) {
			r.Get("/", getPackage(pkgs))
			r.Put("/", updatePackage(pkgs, audit))
			r.Delete("/", deletePackage(pkgs, audit))
		})
	})
}

func listPackages(pkgs core.PackageRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit <= 0 || limit > 100 {
			limit = 25
		}
		list, total, err := pkgs.List(r.Context(), offset, limit)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "query failed"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"packages": list, "total": total})
	}
}

func createPackage(pkgs core.PackageRepo, audit core.AuditRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name         string `json:"name"`
			DiskMB       int64  `json:"diskMb"`
			BandwidthMB  int64  `json:"bandwidthMb"`
			MaxDomains   int    `json:"maxDomains"`
			MaxDatabases int    `json:"maxDatabases"`
			MaxMailboxes int    `json:"maxMailboxes"`
			MaxFTP       int    `json:"maxFtp"`
			PHPVersion   string `json:"phpVersion"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
			return
		}
		if req.Name == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name required"})
			return
		}
		if req.PHPVersion == "" {
			req.PHPVersion = "default"
		}
		p := &core.Package{
			Name: req.Name, DiskMB: req.DiskMB, BandwidthMB: req.BandwidthMB,
			MaxDomains: req.MaxDomains, MaxDatabases: req.MaxDatabases,
			MaxMailboxes: req.MaxMailboxes, MaxFTP: req.MaxFTP,
			PHPVersion: req.PHPVersion, CreatedAt: time.Now(),
		}
		if err := pkgs.Create(r.Context(), p); err != nil {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "package already exists"})
			return
		}
		actor := UserFromContext(r.Context())
		username := ""
		if actor != nil {
			username = actor.Username
		}
		_ = audit.Write(r.Context(), &core.AuditEntry{
			At: time.Now(), ActorUsername: username,
			ActorIP: extractIP(r), Action: "package.create",
			TargetType: "package", TargetID: req.Name,
		})
		writeJSON(w, http.StatusCreated, map[string]any{"id": p.ID, "name": p.Name})
	}
}

func getPackage(pkgs core.PackageRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "pkgId"), 10, 64)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid pkg id"})
			return
		}
		p, err := pkgs.GetByID(r.Context(), id)
		if err != nil {
			if err == store.ErrNotFound {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
			return
		}
		writeJSON(w, http.StatusOK, p)
	}
}

func updatePackage(pkgs core.PackageRepo, audit core.AuditRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "pkgId"), 10, 64)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid pkg id"})
			return
		}
		p, err := pkgs.GetByID(r.Context(), id)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		var req struct {
			Name         string `json:"name"`
			DiskMB       int64  `json:"diskMb"`
			BandwidthMB  int64  `json:"bandwidthMb"`
			MaxDomains   int    `json:"maxDomains"`
			MaxDatabases int    `json:"maxDatabases"`
			MaxMailboxes int    `json:"maxMailboxes"`
			MaxFTP       int    `json:"maxFtp"`
			PHPVersion   string `json:"phpVersion"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
			return
		}
		if req.Name != "" {
			p.Name = req.Name
		}
		if req.DiskMB > 0 {
			p.DiskMB = req.DiskMB
		}
		if req.BandwidthMB > 0 {
			p.BandwidthMB = req.BandwidthMB
		}
		if req.MaxDomains > 0 {
			p.MaxDomains = req.MaxDomains
		}
		if req.MaxDatabases > 0 {
			p.MaxDatabases = req.MaxDatabases
		}
		if req.MaxMailboxes > 0 {
			p.MaxMailboxes = req.MaxMailboxes
		}
		if req.MaxFTP > 0 {
			p.MaxFTP = req.MaxFTP
		}
		if req.PHPVersion != "" {
			p.PHPVersion = req.PHPVersion
		}
		if err := pkgs.Update(r.Context(), p); err != nil {
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
			ActorIP: extractIP(r), Action: "package.update",
			TargetType: "package", TargetID: p.Name,
		})
		writeJSON(w, http.StatusOK, p)
	}
}

func deletePackage(pkgs core.PackageRepo, audit core.AuditRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "pkgId"), 10, 64)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid pkg id"})
			return
		}
		p, err := pkgs.GetByID(r.Context(), id)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		if err := pkgs.Delete(r.Context(), id); err != nil {
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
			ActorIP: extractIP(r), Action: "package.delete",
			TargetType: "package", TargetID: p.Name,
		})
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	}
}
