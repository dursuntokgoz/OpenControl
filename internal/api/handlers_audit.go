package api

import (
	"net/http"
	"strconv"

	"github.com/dursuntokgoz/OpenControl/internal/core"
)

func handleAuditLog(audit core.AuditRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit <= 0 || limit > 100 {
			limit = 25
		}
		list, total, err := audit.List(r.Context(), offset, limit)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "query failed"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"entries": list, "total": total})
	}
}
