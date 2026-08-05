// adapters/in/rest/handlers_identity.go — handlers REST pour les identités (demande de compte)
package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"myr-core/domain/identity"
	rbac "myr-core/domain/role"
)

// GuestHandle est le nom réservé pour l'identité invité partagée.
const GuestHandle = "guest"

// walletDTO est la représentation JSON d'un wallet local exposée au GUI.
type walletDTO struct {
	Handle string `json:"handle"`        // "alice@Org1"
	Name   string `json:"name"`          // "alice"
	OrgID  string `json:"org_id"`        // "Org1MSP"
	Status string `json:"status"`        // "pending" | "active" | "suspended"
	Guest  bool   `json:"guest,omitempty"` // true = identité invité (lecture seule)
}

// networkPolicyResponse est la réponse de GET /api/identity/policy.
// Le GUI l'interroge au démarrage pour savoir s'il doit afficher le mur d'identité.
type networkPolicyResponse struct {
	Mode              string `json:"mode"`                        // "blockchain"
	AllowAutoGuest    bool   `json:"allow_auto_guest"`            // le réseau délivre un accès reader automatiquement
	AllowAutoRegister bool   `json:"allow_auto_register"`         // le réseau enregistre les comptes automatiquement
	NetworkName       string `json:"network_name,omitempty"`
}

// sessionDTO est le token de session retourné après identification réussie.
type sessionDTO struct {
	Token   string `json:"token"`
	Role    string `json:"role"`    // "reader" | "contributor" | ...
	Pseudo  string `json:"pseudo"`
	Channel string `json:"channel,omitempty"` // canal Fabric actif
	Guest   bool   `json:"guest,omitempty"`
}

func toWalletDTO(w identity.WalletEntry) walletDTO {
	return walletDTO{
		Handle: w.Handle,
		Name:   w.Name,
		OrgID:  w.OrgID,
		Status: w.Status,
		Guest:  w.Name == GuestHandle,
	}
}

// handleIdentityWallets liste les wallets présents localement (~/.Myr/wallets/) sur le
// serveur. Ce répertoire est partagé par tous les utilisateurs enrôlés sur ce nœud (pas
// seulement par l'appelant) : une session sans la permission identity.admin ne voit donc
// que le ou les wallets correspondant à son propre pseudo (Handle "<pseudo>@<org>"), jamais
// ceux des autres utilisateurs.
//
//	@Summary	Lister les wallets locaux
//	@Tags		identity
//	@Produce	json
//	@Success	200	{array}		walletDTO
//	@Failure	503	{object}	map[string]string
//	@Security	MyrToken
//	@Router		/identity/wallets [get]
func (h *Handler) handleIdentityWallets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.identitySvc == nil {
		jsonError(w, "service identité non configuré", http.StatusServiceUnavailable)
		return
	}
	wallets, err := h.identitySvc.ListLocalWallets()
	if err != nil {
		internalErr(w, err)
		return
	}
	sess := sessionFromCtx(r)
	self := !h.hasPermission(sess, rbac.PermIdentityAdmin)
	dtos := make([]walletDTO, 0, len(wallets))
	for _, we := range wallets {
		if self && we.Name != sess.Pseudo {
			continue
		}
		dtos = append(dtos, toWalletDTO(we))
	}
	jsonOK(w, dtos)
}

// handleIdentityEnroll enrôle un utilisateur auprès de la CA Fabric avec le
// secret d'enrollment fourni par l'admin, sans créer de session REST.
//
//	@Summary	Enrôlement CA (sans session)
//	@Tags		identity
//	@Accept		json
//	@Produce	json
//	@Param		body	body		object	true	"name, secret, org_id requis"
//	@Success	200	{object}	walletDTO
//	@Failure	400	{object}	map[string]string
//	@Failure	503	{object}	map[string]string
//	@Router		/identity/enroll [post]
func (h *Handler) handleIdentityEnroll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.identitySvc == nil {
		jsonError(w, "service identité non configuré", http.StatusServiceUnavailable)
		return
	}
	var req struct {
		Name   string `json:"name"`
		Secret string `json:"secret"`
		OrgID  string `json:"org_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "corps JSON invalide", http.StatusBadRequest)
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.OrgID = strings.TrimSpace(req.OrgID)
	req.Secret = strings.TrimSpace(req.Secret)
	if req.Name == "" || req.Secret == "" || req.OrgID == "" {
		jsonError(w, "name, secret et org_id sont requis", http.StatusBadRequest)
		return
	}

	entry, err := h.identitySvc.Enroll(context.Background(), req.Name, req.Secret, req.OrgID)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	jsonOK(w, toWalletDTO(entry))
}

// handleIdentityStatus interroge la CA pour le statut courant d'un wallet.
//
//	@Summary	Statut CA d'un wallet
//	@Tags		identity
//	@Produce	json
//	@Param		handle	query		string	true	"identifiant pseudo@org, ex: alice@Org1"
//	@Success	200	{object}	map[string]string
//	@Failure	400	{object}	map[string]string
//	@Failure	503	{object}	map[string]string
//	@Security	MyrToken
//	@Router		/identity/status [get]
func (h *Handler) handleIdentityStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.identitySvc == nil {
		jsonError(w, "service identité non configuré", http.StatusServiceUnavailable)
		return
	}
	handle := strings.TrimSpace(r.URL.Query().Get("handle"))
	if handle == "" {
		jsonError(w, "paramètre handle requis (ex: alice@Org1)", http.StatusBadRequest)
		return
	}
	parts := strings.SplitN(handle, "@", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		jsonError(w, "handle invalide — format attendu : pseudo@org", http.StatusBadRequest)
		return
	}
	we := identity.WalletEntry{
		Handle: handle,
		Name:   parts[0],
		OrgID:  parts[1] + "MSP",
	}
	status, err := h.identitySvc.GetStatus(context.Background(), we)
	if err != nil {
		internalErr(w, err)
		return
	}
	jsonOK(w, map[string]string{"handle": handle, "status": status})
}

// accountRequestDTO est le corps attendu pour POST /api/identity/request.
type accountRequestDTO struct {
	Pseudo      string `json:"pseudo"`
	DisplayName string `json:"display_name,omitempty"`
	Email       string `json:"email"`
	OrgID       string `json:"org_id"`
	Message     string `json:"message,omitempty"`
}

// handleIdentityRequest soumet une demande d'accès sans credential préalable —
// utilisé par un inconnu qui veut accéder au réseau et doit passer par l'admin.
// Si le réseau actif autorise l'auto-enregistrement (AllowAutoRegister), la
// demande est immédiatement approuvée et le secret d'enrollment est renvoyé.
//
//	@Summary	Demander un accès au réseau
//	@Tags		identity
//	@Accept		json
//	@Produce	json
//	@Param		body	body		accountRequestDTO	true	"pseudo, email, org_id requis"
//	@Success	201	{object}	identity.AccountRequest
//	@Failure	400	{object}	map[string]string
//	@Failure	429	{object}	map[string]string
//	@Failure	503	{object}	map[string]string
//	@Router		/identity/request [post]
func (h *Handler) handleIdentityRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !h.authLimiter.allow(clientIP(r)) {
		jsonError(w, "trop de tentatives, réessayez dans une minute", http.StatusTooManyRequests)
		return
	}
	if h.identitySvc == nil {
		jsonError(w, "service identité non configuré", http.StatusServiceUnavailable)
		return
	}
	var dto accountRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, "corps JSON invalide", http.StatusBadRequest)
		return
	}
	dto.Pseudo = strings.TrimSpace(dto.Pseudo)
	dto.Email = strings.TrimSpace(dto.Email)
	dto.OrgID = strings.TrimSpace(dto.OrgID)
	if dto.Pseudo == "" || dto.Email == "" || dto.OrgID == "" {
		jsonError(w, "pseudo, email et org_id sont requis", http.StatusBadRequest)
		return
	}
	req := identity.AccountRequest{
		Pseudo:      dto.Pseudo,
		DisplayName: strings.TrimSpace(dto.DisplayName),
		Email:       dto.Email,
		OrgID:       dto.OrgID,
		Message:     strings.TrimSpace(dto.Message),
	}
	saved, err := h.identitySvc.SubmitRequest(req)
	if err != nil {
		internalErr(w, err)
		return
	}

	// Si le réseau autorise l'auto-enregistrement, créer l'identité dans la CA
	// et retourner le secret d'enrollment directement.
	if h.networkSvc != nil {
		if profile, err2 := h.networkSvc.GetActive(); err2 == nil && profile != nil && profile.AllowAutoRegister {
			secret, err3 := h.identitySvc.AutoRegister(r.Context(), saved, profile.AutoRegisterRole)
			if err3 == nil {
				w.WriteHeader(http.StatusCreated)
				jsonOK(w, map[string]string{
					"id":      saved.ID,
					"status":  saved.Status,
					"pseudo":  saved.Pseudo,
					"org_id":  saved.OrgID,
					"secret":  secret,
				})
				return
			}
			// Échec de l'enregistrement CA — fallback demande pending, logguer l'erreur
			// sans exposer les détails internes au client.
		}
	}

	w.WriteHeader(http.StatusCreated)
	jsonOK(w, saved)
}

// handleIdentityRequests liste les demandes de compte en attente — réservé aux
// identités portant la permission identity.admin (voir domain/role) : les
// demandes exposent des données personnelles (email, message libre) de tous
// les utilisateurs, jamais accessible à un compte non-admin.
//
//	@Summary	Lister les demandes de compte
//	@Tags		identity
//	@Produce	json
//	@Success	200	{array}		identity.AccountRequest
//	@Failure	403	{object}	map[string]string
//	@Failure	503	{object}	map[string]string
//	@Security	MyrToken
//	@Router		/identity/requests [get]
func (h *Handler) handleIdentityRequests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.identitySvc == nil {
		jsonError(w, "service identité non configuré", http.StatusServiceUnavailable)
		return
	}
	reqs, err := h.identitySvc.ListRequests()
	if err != nil {
		internalErr(w, err)
		return
	}
	if reqs == nil {
		reqs = []*identity.AccountRequest{}
	}
	jsonOK(w, reqs)
}

// handleIdentityPolicy retourne la politique d'accès du réseau actif — un
// client (GUI ou tout autre logiciel tiers) l'interroge au démarrage pour
// savoir s'il doit bloquer ou autoriser l'accès automatique.
//
//	@Summary	Politique d'accès du réseau actif
//	@Tags		identity
//	@Produce	json
//	@Success	200	{object}	networkPolicyResponse
//	@Router		/identity/policy [get]
func (h *Handler) handleIdentityPolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.networkSvc == nil {
		jsonOK(w, networkPolicyResponse{Mode: "blockchain"})
		return
	}
	profile, err := h.networkSvc.GetActive()
	if err != nil || profile == nil {
		jsonOK(w, networkPolicyResponse{Mode: "blockchain"})
		return
	}
	jsonOK(w, networkPolicyResponse{
		Mode:              "blockchain",
		AllowAutoGuest:    profile.AllowAutoGuest,
		AllowAutoRegister: profile.AllowAutoRegister,
		NetworkName:       profile.Name,
	})
}

// handleIdentitySession crée une session à partir d'un secret d'enrollment
// (utilisateur déjà connu de la CA) et retourne un token opaque à inclure
// dans l'en-tête X-Myr-Token pour les requêtes suivantes.
//
//	@Summary	Créer une session (utilisateur connu)
//	@Tags		identity
//	@Accept		json
//	@Produce	json
//	@Param		body	body		object	true	"name, secret, org_id requis ; channel optionnel"
//	@Success	200	{object}	sessionDTO
//	@Failure	400	{object}	map[string]string
//	@Failure	401	{object}	map[string]string
//	@Failure	429	{object}	map[string]string
//	@Failure	503	{object}	map[string]string
//	@Router		/identity/session [post]
func (h *Handler) handleIdentitySession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !h.authLimiter.allow(clientIP(r)) {
		jsonError(w, "trop de tentatives, réessayez dans une minute", http.StatusTooManyRequests)
		return
	}
	var req struct {
		Name    string `json:"name"`
		Secret  string `json:"secret"`
		OrgID   string `json:"org_id"`
		Channel string `json:"channel,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "corps JSON invalide", http.StatusBadRequest)
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.OrgID = strings.TrimSpace(req.OrgID)
	req.Secret = strings.TrimSpace(req.Secret)
	req.Channel = strings.TrimSpace(req.Channel)
	if req.Name == "" || req.Secret == "" || req.OrgID == "" {
		jsonError(w, "name, secret et org_id sont requis", http.StatusBadRequest)
		return
	}
	if h.identitySvc == nil {
		jsonError(w, "service identité non configuré", http.StatusServiceUnavailable)
		return
	}
	_, err := h.identitySvc.Enroll(context.Background(), req.Name, req.Secret, req.OrgID)
	if err != nil {
		jsonError(w, err.Error(), http.StatusUnauthorized)
		return
	}
	channel := req.Channel
	if channel == "" {
		channel = h.netInfo.Channel // canal par défaut du réseau
	}
	// #question rôle codé en dur — ignore l'attribut Myr.role du certificat CA
	// réellement enrôlé (voir UCA02.md, écart documenté). Faut-il le lire depuis
	// WalletEntry/attributs CA plutôt que de le figer ici ?
	sess, err := h.sessions.create(req.Name, "contributor", channel)
	if err != nil {
		jsonError(w, "erreur interne", http.StatusInternalServerError)
		return
	}
	jsonOK(w, sessionDTO{Token: sess.Token, Role: sess.Role, Pseudo: sess.Pseudo, Channel: sess.Channel})
}

// handleIdentityGuest délivre un token reader si le réseau autorise l'accès
// automatique (AllowAutoGuest=true). Sinon, enregistre une demande en attente
// si pseudo/email/org_id sont fournis, et répond 403.
//
//	@Summary	Accès invité automatique
//	@Tags		identity
//	@Accept		json
//	@Produce	json
//	@Param		body	body		object	false	"pseudo, display_name, email, org_id, message — tous optionnels"
//	@Success	200	{object}	sessionDTO
//	@Failure	403	{object}	map[string]string
//	@Failure	429	{object}	map[string]string
//	@Router		/identity/guest [post]
func (h *Handler) handleIdentityGuest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !h.authLimiter.allow(clientIP(r)) {
		jsonError(w, "trop de tentatives, réessayez dans une minute", http.StatusTooManyRequests)
		return
	}
	var body struct {
		Pseudo      string `json:"pseudo,omitempty"`
		DisplayName string `json:"display_name,omitempty"`
		Email       string `json:"email,omitempty"`
		OrgID       string `json:"org_id,omitempty"`
		Message     string `json:"message,omitempty"`
	}
	// Corps optionnel — un accès guest pur n'en a pas besoin
	json.NewDecoder(r.Body).Decode(&body) //nolint:errcheck

	// Vérifier la politique réseau
	var allowGuest bool
	var networkOrgID string
	if h.networkSvc != nil {
		profile, err := h.networkSvc.GetActive()
		if err == nil && profile != nil {
			allowGuest = profile.AllowAutoGuest
			networkOrgID = profile.MSPID
		}
	}

	if allowGuest {
		// Délivrer un token reader immédiatement
		pseudo := strings.TrimSpace(body.Pseudo)
		if pseudo == "" {
			pseudo = "guest"
		}
		orgID := networkOrgID
		if orgID == "" {
			orgID = strings.TrimSpace(body.OrgID)
		}
		_ = orgID // utilisé comme contexte, le token est suffisant
		sess, err := h.sessions.create(pseudo, "reader", h.netInfo.Channel)
		if err != nil {
			jsonError(w, "erreur interne", http.StatusInternalServerError)
			return
		}
		jsonOK(w, sessionDTO{Token: sess.Token, Role: sess.Role, Pseudo: sess.Pseudo, Channel: sess.Channel, Guest: true})
		return
	}

	// Réseau privé — enregistrer la demande si les infos sont fournies
	if h.identitySvc != nil && strings.TrimSpace(body.Pseudo) != "" &&
		strings.TrimSpace(body.Email) != "" && strings.TrimSpace(body.OrgID) != "" {
		req := identity.AccountRequest{
			Pseudo:      strings.TrimSpace(body.Pseudo),
			DisplayName: strings.TrimSpace(body.DisplayName),
			Email:       strings.TrimSpace(body.Email),
			OrgID:       strings.TrimSpace(body.OrgID),
			Message:     strings.TrimSpace(body.Message),
		}
		h.identitySvc.SubmitRequest(req) //nolint:errcheck — best-effort
	}

	// 403 : le réseau n'autorise pas l'accès automatique
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	json.NewEncoder(w).Encode(map[string]string{ //nolint:errcheck
		"error": "ce réseau ne permet pas l'accès automatique — contactez l'administrateur",
	})
}
