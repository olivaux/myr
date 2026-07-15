// adapters/in/rest/handlers_admin.go — API d'administration (rôle "admin" requis)
package rest

import (
	"net/http"
	"strings"
	"time"
)

// ── GET /api/admin/sessions  /  DELETE /api/admin/sessions/{token} ───────────

// handleAdminSessions liste toutes les sessions actives.
//
//	@Summary		Lister les sessions actives
//	@Description	Réservé au rôle admin. Le token n'est jamais renvoyé en entier (seulement les 8 premiers caractères) pour ne pas exposer un secret exploitable côté client.
//	@Tags			admin
//	@Produce		json
//	@Success		200	{object}	map[string]interface{}
//	@Failure		403	{object}	map[string]string
//	@Security		MyrToken
//	@Router			/admin/sessions [get]
func (h *Handler) handleAdminSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	sessions := h.sessions.list()

	type sessionDTO struct {
		Token     string    `json:"token_prefix"` // 8 premiers chars seulement (sécurité)
		Role      string    `json:"role"`
		Pseudo    string    `json:"pseudo"`
		NetworkID string    `json:"network_id,omitempty"`
		ExpiresAt time.Time `json:"expires_at"`
	}
	dtos := make([]sessionDTO, 0, len(sessions))
	for _, s := range sessions {
		prefix := s.Token
		if len(prefix) > 8 {
			prefix = prefix[:8] + "…"
		}
		dtos = append(dtos, sessionDTO{
			Token:     prefix,
			Role:      s.Role,
			Pseudo:    s.Pseudo,
			NetworkID: s.NetworkID,
			ExpiresAt: s.ExpiresAt,
		})
	}
	jsonOK(w, map[string]any{"sessions": dtos, "count": len(dtos)})
}

// handleAdminSession révoque une session par son token complet.
//
//	@Summary	Révoquer une session
//	@Tags		admin
//	@Param		token	path	string	true	"token complet de la session (pas le préfixe renvoyé par GET /admin/sessions)"
//	@Success	204	"pas de contenu"
//	@Failure	400	{object}	map[string]string
//	@Failure	403	{object}	map[string]string
//	@Security	MyrToken
//	@Router		/admin/sessions/{token} [delete]
func (h *Handler) handleAdminSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		methodNotAllowed(w)
		return
	}

	token := strings.TrimPrefix(r.URL.Path, "/api/admin/sessions/")
	if token == "" {
		jsonError(w, "token requis", http.StatusBadRequest)
		return
	}

	h.sessions.delete(token)
	w.WriteHeader(http.StatusNoContent)
}
