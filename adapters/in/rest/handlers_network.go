// adapters/in/rest/handlers_network.go — gestion des réseaux Fabric (Phase 2 multi-réseau)
package rest

import (
	"net/http"
	"time"
)

// ── GET/POST /api/networks ────────────────────────────────────────────────────

// handleNetworks liste tous les profils réseau configurés.
//
//	@Summary		Lister les profils réseau
//	@Description	Public — ne renvoie que des métadonnées, jamais de credentials.
//	@Tags			networks
//	@Produce		json
//	@Success		200	{object}	map[string]interface{}
//	@Failure		500	{object}	map[string]string
//	@Router			/networks [get]
func (h *Handler) handleNetworks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	if h.networkSvc == nil {
		jsonOK(w, map[string]any{"networks": []any{}})
		return
	}
	profiles, err := h.networkSvc.List()
	if err != nil {
		internalErr(w, err)
		return
	}

	// Ajouter l'état de connexion depuis le pool
	type networkDTO struct {
		ID            string    `json:"id"`
		Name          string    `json:"name"`
		PeerEndpoint  string    `json:"peer_endpoint"`
		MSPID         string    `json:"msp_id"`
		FabricChannel string    `json:"fabric_channel"`
		Channels      []string  `json:"channels"`
		CAEndpoint    string    `json:"ca_endpoint,omitempty"`
		Active        bool      `json:"active"`
		Connected     bool      `json:"connected"`
		CreatedAt     time.Time `json:"created_at"`
	}
	dtos := make([]networkDTO, 0, len(profiles))
	for _, p := range profiles {
		connected := h.bcRouter != nil && h.bcRouter.BlockchainFor(p.ID) != nil
		dtos = append(dtos, networkDTO{
			ID:            p.ID,
			Name:          p.Name,
			PeerEndpoint:  p.PeerEndpoint,
			MSPID:         p.MSPID,
			FabricChannel: p.FabricChannel,
			Channels:      p.Channels,
			CAEndpoint:    p.CAEndpoint,
			Active:        p.Active,
			Connected:     connected,
			CreatedAt:     p.CreatedAt,
		})
	}
	jsonOK(w, map[string]any{"networks": dtos})
}

// ── GET/PUT /api/networks/active ─────────────────────────────────────────────

// handleNetworkActive retourne ou change le réseau actif de la session courante.
//
// GET  /api/networks/active  → profil du réseau actif + canaux disponibles
// PUT  /api/networks/active  → { "network_id": "net-xxx" } → change le réseau de la session
func (h *Handler) handleNetworkActive(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getNetworkActive(w, r)
	case http.MethodPut:
		h.putNetworkActive(w, r)
	default:
		methodNotAllowed(w)
	}
}

// getNetworkActive renvoie le profil et les canaux du réseau actif de la session.
//
//	@Summary	Réseau actif de la session
//	@Tags		networks
//	@Produce	json
//	@Success	200	{object}	map[string]interface{}
//	@Failure	500	{object}	map[string]string
//	@Security	MyrToken
//	@Router		/networks/active [get]
func (h *Handler) getNetworkActive(w http.ResponseWriter, r *http.Request) {
	networkID := h.networkFor(r)
	if h.networkSvc == nil {
		jsonOK(w, map[string]any{
			"network_id": networkID,
			"channel":    h.channelFor(r),
			"channels":   h.netInfo.Channels,
		})
		return
	}
	profiles, err := h.networkSvc.List()
	if err != nil {
		internalErr(w, err)
		return
	}
	for _, p := range profiles {
		if p.ID == networkID {
			jsonOK(w, map[string]any{
				"network_id":     p.ID,
				"name":           p.Name,
				"peer_endpoint":  p.PeerEndpoint,
				"fabric_channel": p.FabricChannel,
				"channel":        h.channelFor(r),
				"channels":       p.Channels,
				"connected":      h.bcRouter != nil,
			})
			return
		}
	}
	// Réseau introuvable : retourne les infos legacy
	jsonOK(w, map[string]any{
		"network_id": networkID,
		"channel":    h.channelFor(r),
		"channels":   h.netInfo.Channels,
	})
}

// putNetworkActive change le réseau actif de la session et réinitialise le
// canal sur le canal par défaut du nouveau réseau.
//
//	@Summary	Changer le réseau actif de la session
//	@Tags		networks
//	@Accept		json
//	@Produce	json
//	@Param		body	body		object	true	"network_id requis"
//	@Success	200	{object}	map[string]interface{}
//	@Failure	400	{object}	map[string]string
//	@Failure	401	{object}	map[string]string
//	@Failure	404	{object}	map[string]string
//	@Failure	501	{object}	map[string]string
//	@Security	MyrToken
//	@Router		/networks/active [put]
func (h *Handler) putNetworkActive(w http.ResponseWriter, r *http.Request) {
	sess := sessionFromCtx(r)
	// Accepte JWT (session injectée par requireAuth) ou X-Myr-Token legacy
	token := ""
	if sess != nil {
		token = sess.Token
	} else {
		token = r.Header.Get("X-Myr-Token")
		if h.sessions.get(token) == nil {
			jsonError(w, "authentification requise", http.StatusUnauthorized)
			return
		}
	}

	var body struct {
		NetworkID string `json:"network_id"`
	}
	if err := readJSON(r, &body); err != nil || body.NetworkID == "" {
		jsonError(w, "network_id requis", http.StatusBadRequest)
		return
	}

	// Valider que le réseau existe
	if h.networkSvc == nil {
		jsonError(w, "service réseau non configuré", http.StatusNotImplemented)
		return
	}
	profiles, err := h.networkSvc.List()
	if err != nil {
		internalErr(w, err)
		return
	}
	var found bool
	var channels []string
	var defaultChannel string
	for _, p := range profiles {
		if p.ID == body.NetworkID {
			found = true
			channels = p.Channels
			defaultChannel = p.FabricChannel
			break
		}
	}
	if !found {
		jsonError(w, "réseau introuvable", http.StatusNotFound)
		return
	}

	h.sessions.setNetwork(token, body.NetworkID)
	// Reset du canal vers le canal par défaut du nouveau réseau
	if defaultChannel != "" {
		h.sessions.setChannel(token, defaultChannel)
	}
	jsonOK(w, map[string]any{
		"network_id": body.NetworkID,
		"channel":    defaultChannel,
		"channels":   channels,
	})
}
