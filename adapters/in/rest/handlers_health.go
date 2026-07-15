// adapters/in/rest/handlers_health.go — GET /api/health
package rest

import (
	"encoding/json"
	"net/http"
	"time"
)

// handleHealth retourne l'état de chaque composant du serveur (fabric,
// storage, sessions). Non authentifié — appelé par les load-balancers et
// l'infra de monitoring.
//
//	@Summary	Health check
//	@Tags		infra
//	@Produce	json
//	@Success	200	{object}	map[string]interface{}
//	@Failure	503	{object}	map[string]interface{}
//	@Router		/health [get]
func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	type componentStatus struct {
		Status  string `json:"status"` // "ok" | "degraded" | "unavailable"
		Message string `json:"message,omitempty"`
	}

	components := map[string]componentStatus{}
	overall := "ok"

	// Fabric / blockchain
	if h.bcRouter != nil || h.modelSvc != nil {
		components["fabric"] = componentStatus{Status: "ok"}
	} else {
		components["fabric"] = componentStatus{Status: "unavailable", Message: "non configuré"}
		overall = "degraded"
	}

	// Stockage fichiers
	if h.fileStore != nil {
		components["storage"] = componentStatus{Status: "ok"}
	} else {
		components["storage"] = componentStatus{Status: "unavailable", Message: "non configuré"}
		overall = "degraded"
	}

	// Sessions
	if h.sessions != nil {
		active := len(h.sessions.list())
		components["sessions"] = componentStatus{
			Status:  "ok",
			Message: formatCount(active, "session active"),
		}
	}

	code := http.StatusOK
	if overall != "ok" {
		code = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]any{ //nolint:errcheck
		"status":     overall,
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
		"components": components,
	})
}

func formatCount(n int, label string) string {
	if n == 0 {
		return "aucune " + label
	}
	if n == 1 {
		return "1 " + label
	}
	return itoa(n) + " " + label + "s"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := [20]byte{}
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[pos:])
}
