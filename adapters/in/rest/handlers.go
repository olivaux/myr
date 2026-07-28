// adapters/in/rest/handlers.go — handlers REST (adaptateur entrant HTTP)
package rest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"myr-core/domain/identity"
	"myr-core/domain/model"
	"myr-core/domain/network"
	rbac "myr-core/domain/role"
)

// validFieldID valide les identifiants libres (owner_id, channel_id, parent_id…).
var validFieldID = regexp.MustCompile(`^[a-zA-Z0-9_\-\.@]{1,128}$`)

// Handler traduit les requêtes HTTP en appels aux ports d'entrée du domaine.
// NetworkInfo regroupe les métadonnées du réseau Fabric transmises au frontend.
type NetworkInfo struct {
	Channel  string   // canal par défaut (depuis fabric.env)
	Channels []string // tous les canaux disponibles sur ce réseau
	Peer     string
	Network  string
}

// ipRateLimiter limite le nombre de tentatives par IP sur les routes sensibles.
// Fenêtre glissante : maxAttempts par window.
type ipRateLimiter struct {
	mu          sync.Mutex
	attempts    map[string][]time.Time
	maxAttempts int
	window      time.Duration
}

func newIPRateLimiter(max int, window time.Duration) *ipRateLimiter {
	l := &ipRateLimiter{attempts: make(map[string][]time.Time), maxAttempts: max, window: window}
	go l.cleanup()
	return l
}

func (l *ipRateLimiter) allow(ip string) bool {
	now := time.Now()
	cutoff := now.Add(-l.window)
	l.mu.Lock()
	defer l.mu.Unlock()
	times := l.attempts[ip]
	valid := times[:0]
	for _, t := range times {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	valid = append(valid, now)
	l.attempts[ip] = valid
	return len(valid) <= l.maxAttempts
}

func (l *ipRateLimiter) cleanup() {
	for range time.Tick(5 * time.Minute) {
		cutoff := time.Now().Add(-l.window)
		l.mu.Lock()
		for ip, times := range l.attempts {
			valid := times[:0]
			for _, t := range times {
				if t.After(cutoff) {
					valid = append(valid, t)
				}
			}
			if len(valid) == 0 {
				delete(l.attempts, ip)
			} else {
				l.attempts[ip] = valid
			}
		}
		l.mu.Unlock()
	}
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if idx := strings.IndexByte(xff, ','); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}
	if host, _, ok := strings.Cut(r.RemoteAddr, ":"); ok {
		return host
	}
	return r.RemoteAddr
}

// blockchainRouter sélectionne le bon BlockchainPort pour un réseau donné.
// Implémenté par fabric.NetworkPool — défini ici pour éviter l'import circulaire
// entre adapters/in et adapters/out.
type blockchainRouter interface {
	BlockchainFor(networkID string) model.BlockchainPort
}

type Handler struct {
	modelSvc    model.ModelService
	thumbStore  model.ThumbnailStore
	ifaceStore  model.InterfaceStore
	draftStore  model.DraftStore
	identitySvc identity.IdentityService
	networkSvc  network.NetworkService
	roleSvc     rbac.RoleService
	sessions    sessionBackend
	netInfo     NetworkInfo
	authLimiter *ipRateLimiter

	// Phase 2 : multi-réseau
	bcRouter         blockchainRouter     // nil en mode legacy (réseau unique)
	fileStore        model.FileStoragePort // stockage local
	connStore        model.ConnectionStore // pour svcFor
	defaultNetworkID string               // réseau actif par défaut au démarrage

	fabricConnected bool   // vrai si le gateway Fabric a démarré correctement
	buildDate       string // injecté via ldflags à la compilation
	version         string // injecté via ldflags à la compilation
}

func NewHandler(svc model.ModelService, ts model.ThumbnailStore, is model.InterfaceStore) *Handler {
	return &Handler{
		modelSvc:    svc,
		thumbStore:  ts,
		ifaceStore:  is,
		sessions:    newSessionStore(),
		authLimiter: newIPRateLimiter(10, time.Minute), // 10 tentatives / minute / IP
	}
}

// WithIdentityService attache le service d'identité au handler (requis en mode blockchain).
func (h *Handler) WithIdentityService(svc identity.IdentityService) *Handler {
	h.identitySvc = svc
	return h
}

// WithNetworkService attache le service réseau au handler (requis en mode blockchain).
func (h *Handler) WithNetworkService(svc network.NetworkService) *Handler {
	h.networkSvc = svc
	return h
}

// WithRoleService attache le service de rôles RBAC au handler, consulté par
// requireRole. Sans cet appel, requireRole retombe sur la hiérarchie de rang
// historique (reader < contributor < admin) pour compatibilité.
func (h *Handler) WithRoleService(svc rbac.RoleService) *Handler {
	h.roleSvc = svc
	return h
}

// WithNetworkInfo injecte les infos du réseau Fabric dans le handler (exposées via /api/status).
func (h *Handler) WithNetworkInfo(ni NetworkInfo) *Handler {
	h.netInfo = ni
	return h
}

// WithBlockchainRouter attache le routeur multi-réseau (fabric.NetworkPool).
func (h *Handler) WithBlockchainRouter(r blockchainRouter) *Handler {
	h.bcRouter = r
	return h
}

// WithFileStore attache le stockage de fichiers (MinIO ou local) pour svcFor.
func (h *Handler) WithFileStore(fs model.FileStoragePort) *Handler {
	h.fileStore = fs
	return h
}

// WithConnStore attache le store de connexions pour svcFor.
func (h *Handler) WithConnStore(cs model.ConnectionStore) *Handler {
	h.connStore = cs
	return h
}

// WithDraftStore attache le store de brouillons pour svcFor (composants créés draft:true).
func (h *Handler) WithDraftStore(ds model.DraftStore) *Handler {
	h.draftStore = ds
	return h
}

// WithDefaultNetwork définit le réseau Fabric actif par défaut (ID du NetworkProfile).
func (h *Handler) WithDefaultNetwork(networkID string) *Handler {
	h.defaultNetworkID = networkID
	return h
}

// WithFabricConnected indique si la gateway Fabric a démarré avec succès.
// Utilisé par /api/status pour éviter un appel blockchain bloquant à chaque heartbeat.
func (h *Handler) WithFabricConnected(connected bool) *Handler {
	h.fabricConnected = connected
	return h
}

// WithBuildDate transmet la date de compilation injectée via ldflags.
func (h *Handler) WithBuildDate(d string) *Handler {
	h.buildDate = d
	return h
}

// WithVersion transmet le numéro de version injecté via ldflags.
func (h *Handler) WithVersion(v string) *Handler {
	h.version = v
	return h
}

// WithSessionPersistence active la persistance des sessions sur disque.
// Sans effet si le backend est Redis (les sessions sont déjà persistantes côté Redis).
func (h *Handler) WithSessionPersistence(path string) *Handler {
	if ms, ok := h.sessions.(*sessionStore); ok {
		ms.withPersistence(path)
	}
	return h
}

// WithRedisSession remplace le store de sessions en mémoire par un backend Redis.
// url : "redis://[:password@]host[:port][/db]" ou "rediss://..." pour TLS.
// En cas d'erreur de connexion, le store en mémoire est conservé.
func (h *Handler) WithRedisSession(url string) *Handler {
	rs, err := newRedisSessionStore(url)
	if err != nil {
		return h
	}
	h.sessions = rs
	return h
}

// networkFor retourne l'ID du réseau actif pour cette requête.
// Priorité : session utilisateur → defaultNetworkID configuré au démarrage.
func (h *Handler) networkFor(r *http.Request) string {
	if sess := sessionFromCtx(r); sess != nil && sess.NetworkID != "" {
		return sess.NetworkID
	}
	return h.defaultNetworkID
}

// svcFor retourne un model.ModelService configuré pour le réseau actif de la requête.
// En mode legacy (bcRouter non configuré ou networkID vide), retourne le modelSvc pré-wired.
// En mode multi-réseau, crée une instance légère par requête avec le bon blockchain.
func (h *Handler) svcFor(r *http.Request) model.ModelService {
	if h.bcRouter == nil || h.fileStore == nil {
		return h.modelSvc
	}
	networkID := h.networkFor(r)
	if networkID == "" {
		// Pas de réseau configuré dans le pool — utilise le service wired au démarrage
		return h.modelSvc
	}
	bc := h.bcRouter.BlockchainFor(networkID)
	svc := model.NewService(bc, h.fileStore)
	if h.connStore != nil {
		svc = svc.WithConnStore(h.connStore)
	}
	return svc.WithThumbStore(h.thumbStore).WithIfaceStore(h.ifaceStore).WithDraftStore(h.draftStore)
}

// channelFor retourne le canal actif de la session, ou le canal par défaut du réseau.
func (h *Handler) channelFor(r *http.Request) string {
	if sess := sessionFromCtx(r); sess != nil && sess.Channel != "" {
		return sess.Channel
	}
	return h.netInfo.Channel
}

// requireAuth est un middleware qui vérifie l'authentification.
// Ordre de priorité : 1) JWT Bearer (nouveau), 2) X-Myr-Token (legacy).
func (h *Handler) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-Myr-Token")
		sess := h.sessions.get(token)
		if sess == nil {
			jsonError(w, "authentification requise", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), ctxSession, sess)
		next(w, r.WithContext(ctx))
	}
}

// legacyRoleRank est le repli utilisé quand aucun rbac.RoleService n'est injecté
// (compatibilité rétroactive) : hiérarchie reader < contributor < admin.
var legacyRoleRank = map[string]int{"reader": 1, "contributor": 2, "admin": 3}
var legacyPermRank = map[rbac.Permission]int{
	rbac.PermRead: 1, rbac.PermWrite: 2,
	rbac.PermAdmin: 3, rbac.PermNetworkAdmin: 3, rbac.PermRoleAdmin: 3, rbac.PermIdentityAdmin: 3,
}

// requireRole est un middleware qui vérifie que la session porte la permission requise.
// Consulte le rbac.RoleService injecté (WithRoleService) ; à défaut, retombe sur la
// hiérarchie de rang historique (reader < contributor < admin).
func (h *Handler) requireRole(perm rbac.Permission, next http.HandlerFunc) http.HandlerFunc {
	return h.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		sess := sessionFromCtx(r)
		if sess == nil {
			jsonError(w, "droits insuffisants", http.StatusForbidden)
			return
		}
		var allowed bool
		if h.roleSvc != nil {
			allowed = h.roleSvc.HasPermission(sess.Role, perm)
		} else {
			allowed = legacyRoleRank[sess.Role] >= legacyPermRank[perm]
		}
		if !allowed {
			jsonError(w, "droits insuffisants", http.StatusForbidden)
			return
		}
		next(w, r)
	})
}

// ── DTOs ─────────────────────────────────────────────────────────────────────

type componentDTO struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Description string             `json:"description,omitempty"`
	Category    model.Category     `json:"category"`
	OwnerID     string             `json:"owner_id"`
	ParentID    string             `json:"parent_id,omitempty"`
	BlockID     string             `json:"block_id,omitempty"`
	Hash        string             `json:"hash,omitempty"`
	LicenseID   string             `json:"license_id,omitempty"`
	Tags        []string           `json:"tags"`
	Links       []string           `json:"links,omitempty"`
	Thumbnail   string             `json:"thumbnail,omitempty"`
	// Status vaut "draft" (créé via draft:true, pas encore engagé sur la blockchain
	// — voir POST /components/{id}/submit) ou "submitted" (comportement nominal,
	// immédiat). Vide pour un composant créé avant l'introduction du brouillon.
	Status    model.ModuleStatus `json:"status,omitempty"`
	CreatedAt time.Time          `json:"created_at"`
}

type connectionDTO struct {
	ID              string `json:"id"`
	From            string `json:"from"`
	To              string `json:"to"`
	Label           string `json:"label,omitempty"`
	FromIfaceID     string `json:"from_iface_id,omitempty"`
	ToIfaceID       string `json:"to_iface_id,omitempty"`
	FromInstanceID  string `json:"from_instance_id,omitempty"`
	ToInstanceID    string `json:"to_instance_id,omitempty"`
	FastenerAssetID string `json:"fastener_asset_id,omitempty"`
	Incompatible    bool   `json:"incompatible,omitempty"`
}

type graphResponse struct {
	Components  []componentDTO  `json:"components"`
	Connections []connectionDTO `json:"connections"`
	Total       int             `json:"total"`
}

type statusResponse struct {
	Mode      string `json:"mode"`
	Connected bool   `json:"connected"`
	Assets    int    `json:"assets"`
	Channel   string `json:"channel,omitempty"`
	Peer      string `json:"peer,omitempty"`
	Network   string `json:"network,omitempty"`
}

func toComponentDTO(m *model.Model3D, thumbnail string) componentDTO {
	tags := m.Tags
	if tags == nil {
		tags = []string{}
	}
	return componentDTO{
		ID:          m.ID,
		Name:        m.Name,
		Description: m.Description,
		Category:    m.Category,
		OwnerID:     m.OwnerID,
		ParentID:    m.ParentID,
		BlockID:     m.BlockID,
		Hash:        m.Hash,
		LicenseID:   m.LicenseID,
		Tags:        tags,
		Links:       m.Links,
		Thumbnail:   thumbnail,
		Status:      m.Status,
		CreatedAt:   m.CreatedAt,
	}
}

// ── /api/components ──────────────────────────────────────────────────────────────

func (h *Handler) handleComponents(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listGraph(w, r)
	case http.MethodPost:
		h.createAsset(w, r)
	default:
		http.Error(w, "méthode non autorisée", http.StatusMethodNotAllowed)
	}
}

// listGraph liste les composants (hors modules) avec filtres serveur.
//
//	@Summary		Lister les composants
//	@Description	Filtre côté serveur — ne renvoie qu'un sous-ensemble pour éviter de tout charger. Les modules sont exclus (voir /api/modules).
//	@Tags			components
//	@Produce		json
//	@Param			q			query		string	false	"recherche texte (nom, description, tags)"
//	@Param			categories	query		string	false	"catégories séparées par virgule (ex: base,variation)"
//	@Param			owner_id	query		string	false	"filtre par propriétaire"
//	@Param			parent_id	query		string	false	"filtre par asset parent"
//	@Param			hash		query		string	false	"filtre par hash SHA-256 exact"
//	@Param			tags		query		string	false	"tags séparés par virgule (correspondance sur au moins un tag)"
//	@Param			limit		query		int		false	"nombre maximum de résultats (défaut 200, max 1000)"
//	@Success		200	{object}	graphResponse
//	@Failure		500	{object}	map[string]string
//	@Router			/components [get]
func (h *Handler) listGraph(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("q")))
	cats := r.URL.Query().Get("categories") // "base,variation" ou vide = tous
	ownerID := strings.TrimSpace(r.URL.Query().Get("owner_id"))
	parentID := strings.TrimSpace(r.URL.Query().Get("parent_id"))
	hashFilter := strings.TrimSpace(r.URL.Query().Get("hash")) // hash SHA-256 exact
	tagsParam := r.URL.Query().Get("tags") // "tag1,tag2" — match si l'asset possède au moins un
	limit := 200
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 && l <= 1000 {
		limit = l
	}
	catSet := map[string]bool{}
	if cats != "" {
		for _, c := range strings.Split(cats, ",") {
			if c = strings.TrimSpace(c); c != "" {
				catSet[c] = true
			}
		}
	}
	filterTags := []string{}
	if tagsParam != "" {
		for _, t := range strings.Split(tagsParam, ",") {
			if t = strings.TrimSpace(strings.ToLower(t)); t != "" {
				filterTags = append(filterTags, t)
			}
		}
	}

	all, err := h.svcFor(r).List(h.channelFor(r))
	if err != nil {
		internalErr(w, err)
		return
	}

	// Filtrer
	var filtered []*model.Model3D
	for _, m := range all {
		if m.IsModule() {
			continue // les modules sont exposés via /api/modules, pas /api/components
		}
		if len(catSet) > 0 && !catSet[string(m.Category)] {
			continue
		}
		if q != "" && !assetMatchesQuery(m, q) {
			continue
		}
		if ownerID != "" && m.OwnerID != ownerID {
			continue
		}
		if parentID != "" && m.ParentID != parentID {
			continue
		}
		if hashFilter != "" && m.Hash != hashFilter {
			continue
		}
		if len(filterTags) > 0 && !assetHasAnyTag(m, filterTags) {
			continue
		}
		filtered = append(filtered, m)
	}
	total := len(filtered)
	if limit < len(filtered) {
		filtered = filtered[:limit]
	}

	// Construire les DTOs
	loadedIDs := make(map[string]bool, len(filtered))
	dtos := make([]componentDTO, 0, len(filtered))
	for _, m := range filtered {
		thumb, _ := h.svcFor(r).GetThumbnail(m.ID)
		dtos = append(dtos, toComponentDTO(m, thumb))
		loadedIDs[m.ID] = true
	}

	// Ne renvoyer que les connexions entre assets chargés
	conns, _ := h.svcFor(r).ListConnections()
	connDTOs := make([]connectionDTO, 0)
	for _, c := range conns {
		if loadedIDs[c.From] && loadedIDs[c.To] {
			connDTOs = append(connDTOs, connectionDTO{
				ID: c.ID, From: c.From, To: c.To, Label: c.Label,
				FromIfaceID: c.FromIfaceID, ToIfaceID: c.ToIfaceID,
				FromInstanceID: c.FromInstanceID, ToInstanceID: c.ToInstanceID,
				FastenerAssetID: c.FastenerAssetID, Incompatible: c.Incompatible,
			})
		}
	}

	jsonOK(w, graphResponse{Components: dtos, Connections: connDTOs, Total: total})
}

// assetHasAnyTag retourne true si l'asset possède au moins un des tags (comparaison insensible à la casse).
func assetHasAnyTag(m *model.Model3D, tags []string) bool {
	for _, ft := range tags {
		for _, t := range m.Tags {
			if strings.Contains(strings.ToLower(t), ft) {
				return true
			}
		}
	}
	return false
}

// assetMatchesQuery retourne true si l'asset correspond à la recherche textuelle (nom, tags, description).
func assetMatchesQuery(m *model.Model3D, q string) bool {
	if strings.Contains(strings.ToLower(m.Name), q) {
		return true
	}
	if strings.Contains(strings.ToLower(m.Description), q) {
		return true
	}
	for _, t := range m.Tags {
		if strings.Contains(strings.ToLower(t), q) {
			return true
		}
	}
	return false
}

// createAsset crée un composant à partir d'un formulaire multipart.
//
//	@Summary		Créer un composant
//	@Description	Crée un asset (catégorie par défaut "base"). Le fichier CAO 3D est optionnel (champ "file", ou "stl" pour rétrocompatibilité) ; une miniature base64 peut être fournie directement ou est dérivée de l'og:image du premier lien si aucun fichier n'est envoyé. Par défaut le composant est engagé immédiatement sur la blockchain (une transaction, "status":"submitted"). Si "draft" vaut true, il est stocké localement ("status":"draft", aucune transaction) — voir POST /components/{id}/submit pour l'engager ensuite.
//	@Tags			components
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			name		formData	string	true	"nom du composant (max 256 caractères)"
//	@Param			description	formData	string	false	"description (max 10000 caractères)"
//	@Param			category	formData	string	false	"catégorie (défaut: base)"
//	@Param			owner_id	formData	string	false	"propriétaire"
//	@Param			parent_id	formData	string	false	"asset parent (dérivation)"
//	@Param			channel_id	formData	string	false	"canal cible (défaut: canal de la session)"
//	@Param			license_id	formData	string	false	"licence"
//	@Param			tags		formData	string	false	"tags séparés par virgule"
//	@Param			links		formData	string	false	"liens externes, tableau JSON encodé en chaîne"
//	@Param			file		formData	file	false	"fichier CAO 3D"
//	@Param			thumbnail	formData	string	false	"miniature en data URL base64"
//	@Param			draft		formData	bool	false	"true : crée en brouillon local, sans transaction blockchain (défaut: false)"
//	@Success		201	{object}	componentDTO
//	@Failure		400	{object}	map[string]string
//	@Failure		503	{object}	map[string]string	"blockchain indisponible (composant non-brouillon uniquement)"
//	@Security		MyrToken
//	@Router			/components [post]
func (h *Handler) createAsset(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		jsonError(w, "parsing form: "+err.Error(), http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	description := strings.TrimSpace(r.FormValue("description"))
	ownerID := strings.TrimSpace(r.FormValue("owner_id"))
	channelID := strings.TrimSpace(r.FormValue("channel_id"))

	if name == "" {
		jsonError(w, "le champ 'name' est requis", http.StatusBadRequest)
		return
	}
	if len(name) > 256 {
		jsonError(w, "name trop long (max 256)", http.StatusBadRequest)
		return
	}
	if len(description) > 10000 {
		jsonError(w, "description trop longue (max 10000)", http.StatusBadRequest)
		return
	}
	if ownerID != "" && !validFieldID.MatchString(ownerID) {
		jsonError(w, "owner_id invalide", http.StatusBadRequest)
		return
	}
	if channelID != "" && !validFieldID.MatchString(channelID) {
		jsonError(w, "channel_id invalide", http.StatusBadRequest)
		return
	}
	if channelID == "" {
		channelID = h.channelFor(r)
	}

	req := model.AddRequest{
		Name:        name,
		Description: description,
		Category:    model.Category(r.FormValue("category")),
		OwnerID:     ownerID,
		ParentID:    strings.TrimSpace(r.FormValue("parent_id")),
		ChannelID:   channelID,
		LicenseID:   strings.TrimSpace(r.FormValue("license_id")),
		Tags:        parseTags(r.FormValue("tags")),
		Draft:       r.FormValue("draft") == "true",
	}
	if raw := r.FormValue("links"); raw != "" {
		_ = json.Unmarshal([]byte(raw), &req.Links)
	}
	if req.Category == "" {
		req.Category = model.CategoryBase
	}

	// Fichier 3D optionnel — accepte le champ "file" (générique) ou "stl" (rétrocompat)
	fileField := "file"
	if _, _, err := r.FormFile("file"); err != nil {
		fileField = "stl"
	}
	if file, header, err := r.FormFile(fileField); err == nil {
		defer file.Close()
		ext := filepath.Ext(header.Filename)
		if ext == "" {
			ext = ".bin"
		}
		tmp, err := os.CreateTemp("", "myr-model-*"+ext)
		if err != nil {
			internalErr(w, err)
			return
		}
		defer os.Remove(tmp.Name())

		buf := make([]byte, header.Size)
		tmp.Close()

		// Réécrire proprement
		if f2, err := os.Create(tmp.Name()); err == nil {
			defer f2.Close()
			buf = make([]byte, 32*1024)
			for {
				n, err := file.Read(buf)
				if n > 0 {
					f2.Write(buf[:n])
				}
				if err != nil {
					break
				}
			}
		}
		req.FilePath = tmp.Name()
		_ = buf
	}

	m, err := h.svcFor(r).AddFull(req)
	if err != nil {
		internalErr(w, err)
		return
	}

	// Miniature base64 envoyée par le frontend (Three.js renderer STL)
	if thumb := r.FormValue("thumbnail"); thumb != "" {
		_ = h.svcFor(r).SaveThumbnail(m.ID, thumb)
	} else if len(req.Links) > 0 {
		// Pas de fichier 3D : dériver la miniature de l'og:image du premier lien
		_, _ = h.svcFor(r).RegenerateThumbnail(m.ID)
	}

	thumb, _ := h.svcFor(r).GetThumbnail(m.ID)
	w.WriteHeader(http.StatusCreated)
	jsonOK(w, toComponentDTO(m, thumb))
}

// ── /api/components/:id ──────────────────────────────────────────────────────────

func (h *Handler) handleComponent(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/components/")
	if rest == "" {
		http.NotFound(w, r)
		return
	}
	// Sous-route /:id/interfaces
	if strings.HasSuffix(rest, "/interfaces") {
		id := strings.TrimSuffix(rest, "/interfaces")
		h.handleComponentInterfaces(w, r, id)
		return
	}
	// Sous-route /:id/tree — arbre d'évolution complet (ancêtres + descendants)
	if strings.HasSuffix(rest, "/tree") {
		id := strings.TrimSuffix(rest, "/tree")
		h.handleComponentTree(w, r, id)
		return
	}
	// Sous-route /:id/thumbnail/regenerate — redérive la miniature depuis le lien source
	if strings.HasSuffix(rest, "/thumbnail/regenerate") {
		id := strings.TrimSuffix(rest, "/thumbnail/regenerate")
		h.regenerateThumbnail(w, r, id)
		return
	}
	// Sous-route /:id/submit — engage un composant brouillon sur la blockchain
	if strings.HasSuffix(rest, "/submit") {
		id := strings.TrimSuffix(rest, "/submit")
		h.submitComponent(w, r, id)
		return
	}
	id := rest
	switch r.Method {
	case http.MethodDelete:
		h.deleteComponent(w, r, id)
	case http.MethodGet:
		h.getComponent(w, r, id)
	case http.MethodPatch:
		h.patchComponent(w, r, id)
	default:
		http.Error(w, "méthode non autorisée", http.StatusMethodNotAllowed)
	}
}

// regenerateThumbnail redérive la miniature d'un asset (composant ou module) depuis
// sa source durable — l'og:image du premier lien externe enregistré (Links). Sans
// lien externe, il n'existe pas de source régénérable côté serveur (le rendu d'un
// modèle 3D est produit par le client GUI, pas par myr) — voir ModelService.RegenerateThumbnail.
//
//	@Summary		Régénérer la miniature d'un asset depuis son lien source
//	@Description	Redérive la miniature depuis l'og:image du premier lien externe enregistré (Links). Échoue si l'asset n'a aucun lien externe — un modèle 3D sans lien n'a pas de source régénérable côté serveur.
//	@Tags			components
//	@Produce		json
//	@Param			id	path		string	true	"identifiant de l'asset (composant ou module)"
//	@Success		200	{object}	map[string]string
//	@Failure		400	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		MyrToken
//	@Router			/components/{id}/thumbnail/regenerate [post]
func (h *Handler) regenerateThumbnail(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		http.Error(w, "méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}
	dataURL, err := h.svcFor(r).RegenerateThumbnail(id)
	if err != nil {
		internalErr(w, err)
		return
	}
	jsonOK(w, map[string]string{"thumbnail": dataURL})
}

// submitComponent engage un composant créé en brouillon (draft:true) sur la blockchain.
//
//	@Summary		Soumettre un composant en brouillon
//	@Description	Committe en une transaction les métadonnées et les interfaces locales du composant, puis passe "status" à "submitted". Échoue si le composant n'est pas un brouillon local (déjà soumis, ou inexistant).
//	@Tags			components
//	@Produce		json
//	@Param			id	path		string	true	"identifiant du composant en brouillon"
//	@Success		200	{object}	componentDTO
//	@Failure		400	{object}	map[string]string
//	@Failure		503	{object}	map[string]string	"blockchain indisponible"
//	@Security		MyrToken
//	@Router			/components/{id}/submit [post]
func (h *Handler) submitComponent(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		http.Error(w, "méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}
	m, err := h.svcFor(r).Submit(id)
	if err != nil {
		internalErr(w, err)
		return
	}
	thumb, _ := h.svcFor(r).GetThumbnail(m.ID)
	jsonOK(w, toComponentDTO(m, thumb))
}

// patchComponent modifie un composant existant (champs partiels).
//
//	@Summary		Modifier un composant
//	@Description	Champs modifiables : name, description, license_id, tags, links. Le contenu blockchain déjà soumis reste immuable (règle 7) — seuls les champs hors ledger sont mis à jour.
//	@Tags			components
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string							true	"identifiant du composant"
//	@Param			body	body		object	true	"champs à modifier (name, description, license_id, tags, links)"
//	@Success		200	{object}	componentDTO
//	@Failure		400	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		MyrToken
//	@Router			/components/{id} [patch]
func (h *Handler) patchComponent(w http.ResponseWriter, r *http.Request, id string) {
	var body struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		LicenseID   string   `json:"license_id"`
		Tags        []string `json:"tags"`
		Links       []string `json:"links"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "JSON invalide: "+err.Error(), http.StatusBadRequest)
		return
	}
	m, err := h.svcFor(r).UpdateAsset(model.UpdateRequest{
		ID: id, Name: body.Name, Description: body.Description,
		LicenseID: body.LicenseID, Tags: body.Tags, Links: body.Links,
	})
	if err != nil {
		internalErr(w, err)
		return
	}
	thumb, _ := h.svcFor(r).GetThumbnail(m.ID)
	jsonOK(w, toComponentDTO(m, thumb))
}

// getComponent renvoie le détail d'un composant.
//
//	@Summary	Détail d'un composant
//	@Tags		components
//	@Produce	json
//	@Param		id	path		string	true	"identifiant du composant"
//	@Success	200	{object}	componentDTO
//	@Failure	404	{object}	map[string]string
//	@Security	MyrToken
//	@Router		/components/{id} [get]
func (h *Handler) getComponent(w http.ResponseWriter, r *http.Request, id string) {
	m, err := h.svcFor(r).Get(id, h.channelFor(r))
	if err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}
	thumb, _ := h.svcFor(r).GetThumbnail(m.ID)
	jsonOK(w, toComponentDTO(m, thumb))
}

// deleteComponent supprime un composant du stockage local.
//
//	@Summary		Supprimer un composant
//	@Description	Suppression locale uniquement — la blockchain Fabric ne supporte pas la suppression (règle 9) ; un composant déjà soumis reste sur le ledger.
//	@Tags			components
//	@Param			id	path	string	true	"identifiant du composant"
//	@Success		204	"pas de contenu"
//	@Failure		500	{object}	map[string]string
//	@Security		MyrToken
//	@Router			/components/{id} [delete]
func (h *Handler) deleteComponent(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.svcFor(r).Remove(id); err != nil {
		internalErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleComponentTree renvoie l'arbre d'évolution complet d'un asset.
//
//	@Summary		Arbre d'évolution d'un composant
//	@Description	Tous les ancêtres (via parent_id) et descendants (parcours en largeur) de l'asset, plus les connexions internes à cet ensemble.
//	@Tags			components
//	@Produce		json
//	@Param			id	path		string	true	"identifiant du composant"
//	@Success		200	{object}	graphResponse
//	@Failure		500	{object}	map[string]string
//	@Security		MyrToken
//	@Router			/components/{id}/tree [get]
func (h *Handler) handleComponentTree(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		http.Error(w, "méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}
	all, err := h.svcFor(r).List(h.channelFor(r))
	if err != nil {
		internalErr(w, err)
		return
	}

	// Index par ID
	byID := make(map[string]*model.Model3D, len(all))
	for _, m := range all {
		byID[m.ID] = m
	}

	// Index enfants : parentID → liste d'enfants
	children := make(map[string][]*model.Model3D)
	for _, m := range all {
		if m.ParentID != "" {
			children[m.ParentID] = append(children[m.ParentID], m)
		}
	}

	treeIDs := make(map[string]bool)

	// Ancêtres (chemin vers la racine)
	cur := id
	for cur != "" {
		treeIDs[cur] = true
		if m, ok := byID[cur]; ok {
			cur = m.ParentID
		} else {
			break
		}
	}

	// Descendants (BFS)
	queue := []string{id}
	for len(queue) > 0 {
		cur, queue = queue[0], queue[1:]
		for _, child := range children[cur] {
			if !treeIDs[child.ID] {
				treeIDs[child.ID] = true
				queue = append(queue, child.ID)
			}
		}
	}

	// Construire les DTOs
	dtos := make([]componentDTO, 0, len(treeIDs))
	for aid := range treeIDs {
		m, ok := byID[aid]
		if !ok {
			continue
		}
		thumb, _ := h.svcFor(r).GetThumbnail(m.ID)
		dtos = append(dtos, toComponentDTO(m, thumb))
	}

	// Connexions entre membres de l'arbre
	conns, _ := h.svcFor(r).ListConnections()
	connDTOs := make([]connectionDTO, 0)
	for _, c := range conns {
		if treeIDs[c.From] && treeIDs[c.To] {
			connDTOs = append(connDTOs, connectionDTO{
				ID: c.ID, From: c.From, To: c.To, Label: c.Label,
				FromIfaceID: c.FromIfaceID, ToIfaceID: c.ToIfaceID,
				FromInstanceID: c.FromInstanceID, ToInstanceID: c.ToInstanceID,
				FastenerAssetID: c.FastenerAssetID, Incompatible: c.Incompatible,
			})
		}
	}

	jsonOK(w, graphResponse{Components: dtos, Connections: connDTOs, Total: len(dtos)})
}

// ── /api/connections ─────────────────────────────────────────────────────────

func (h *Handler) handleConnections(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.createConnection(w, r)
	default:
		http.Error(w, "méthode non autorisée", http.StatusMethodNotAllowed)
	}
}

// createConnection crée une connexion simple entre deux assets.
//
//	@Summary	Créer une connexion
//	@Tags		connections
//	@Accept		json
//	@Produce	json
//	@Param		body	body		connectionDTO	true	"from et to requis ; label optionnel"
//	@Success	201	{object}	connectionDTO
//	@Failure	400	{object}	map[string]string
//	@Security	MyrToken
//	@Router		/connections [post]
func (h *Handler) createConnection(w http.ResponseWriter, r *http.Request) {
	var body connectionDTO
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "JSON invalide: "+err.Error(), http.StatusBadRequest)
		return
	}
	if body.From == "" || body.To == "" {
		jsonError(w, "les champs 'from' et 'to' sont requis", http.StatusBadRequest)
		return
	}
	conn, err := h.svcFor(r).AddConnection(body.From, body.To, body.Label)
	if err != nil {
		internalErr(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	jsonOK(w, connectionDTO{ID: conn.ID, From: conn.From, To: conn.To, Label: conn.Label})
}

// ── /api/connections/:id ─────────────────────────────────────────────────────

// handleConnection supprime une connexion existante.
//
//	@Summary	Supprimer une connexion
//	@Tags		connections
//	@Param		id	path	string	true	"identifiant de la connexion"
//	@Success	204	"pas de contenu"
//	@Failure	500	{object}	map[string]string
//	@Security	MyrToken
//	@Router		/connections/{id} [delete]
func (h *Handler) handleConnection(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/connections/")
	if id == "" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodDelete {
		http.Error(w, "méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}
	if err := h.svcFor(r).RemoveConnection(id); err != nil {
		internalErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── /api/assembly-links ──────────────────────────────────────────────────────

type assemblyLinkRequest struct {
	FromIfaceID     string `json:"from_iface_id"`
	ToIfaceID       string `json:"to_iface_id"`
	Label           string `json:"label,omitempty"`
	FromInstanceID  string `json:"from_instance_id,omitempty"`
	ToInstanceID    string `json:"to_instance_id,omitempty"`
	FastenerAssetID string `json:"fastener_asset_id,omitempty"`
}

// handleAssemblyLinks crée une liaison interface→interface (assemblage).
//
//	@Summary		Créer une liaison entre deux interfaces
//	@Description	Déclenche automatiquement la vérification de compatibilité (règle 11) ; les liaisons incompatibles portent incompatible:true et ne se suppriment pas automatiquement (règle 12). Une interface est à usage unique dans une liaison (règle 10).
//	@Tags			connections
//	@Accept			json
//	@Produce		json
//	@Param			body	body		assemblyLinkRequest	true	"from_iface_id et to_iface_id requis"
//	@Success		201	{object}	connectionDTO
//	@Failure		400	{object}	map[string]string
//	@Security		MyrToken
//	@Router			/assembly-links [post]
func (h *Handler) handleAssemblyLinks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}
	var body assemblyLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "JSON invalide: "+err.Error(), http.StatusBadRequest)
		return
	}
	if body.FromIfaceID == "" || body.ToIfaceID == "" {
		jsonError(w, "les champs 'from_iface_id' et 'to_iface_id' sont requis", http.StatusBadRequest)
		return
	}
	conn, err := h.svcFor(r).AddAssemblyLink(body.FromIfaceID, body.ToIfaceID, body.Label, body.FromInstanceID, body.ToInstanceID, body.FastenerAssetID)
	if err != nil {
		internalErr(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	jsonOK(w, connectionDTO{
		ID: conn.ID, From: conn.From, To: conn.To, Label: conn.Label,
		FromIfaceID: conn.FromIfaceID, ToIfaceID: conn.ToIfaceID,
		FromInstanceID: conn.FromInstanceID, ToInstanceID: conn.ToInstanceID,
		FastenerAssetID: conn.FastenerAssetID,
	})
}

// ── /api/virtual-connect ─────────────────────────────────────────────────────

type virtualConnectRequest struct {
	VirtualIfaceID   string  `json:"virtual_iface_id"`
	PhysicalIfaceID  string  `json:"physical_iface_id"`
	Name             string  `json:"name,omitempty"`
	ValueMin         float64 `json:"value_min"`
	ValueMax         float64 `json:"value_max"`
	IsRange          bool    `json:"is_range"`
	Unit             string  `json:"unit,omitempty"`
	FromInstanceID   string  `json:"from_instance_id,omitempty"`
	ToInstanceID     string  `json:"to_instance_id,omitempty"`
}

// handleVirtualConnect relie une interface virtuelle à une interface physique.
//
//	@Summary	Relier une interface virtuelle à une interface physique
//	@Tags		connections
//	@Accept		json
//	@Produce	json
//	@Param		body	body		virtualConnectRequest	true	"virtual_iface_id et physical_iface_id requis"
//	@Success	201	{object}	connectionDTO
//	@Failure	400	{object}	map[string]string
//	@Security	MyrToken
//	@Router		/virtual-connect [post]
func (h *Handler) handleVirtualConnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}
	var body virtualConnectRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "JSON invalide: "+err.Error(), http.StatusBadRequest)
		return
	}
	if body.VirtualIfaceID == "" || body.PhysicalIfaceID == "" {
		jsonError(w, "les champs 'virtual_iface_id' et 'physical_iface_id' sont requis", http.StatusBadRequest)
		return
	}
	popup := model.AssetInterface{
		Name:     body.Name,
		ValueMin: body.ValueMin,
		ValueMax: body.ValueMax,
		IsRange:  body.IsRange,
		Unit:     body.Unit,
	}
	conn, err := h.svcFor(r).ConnectVirtualToPhysical(body.VirtualIfaceID, body.PhysicalIfaceID, popup, body.FromInstanceID, body.ToInstanceID)
	if err != nil {
		internalErr(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	jsonOK(w, connectionDTO{
		ID: conn.ID, From: conn.From, To: conn.To, Label: conn.Label,
		FromIfaceID: conn.FromIfaceID, ToIfaceID: conn.ToIfaceID,
		FromInstanceID: conn.FromInstanceID, ToInstanceID: conn.ToInstanceID,
	})
}

// ── /api/modules ────────────────────────────────────────────────────────────

type moduleDTO struct {
	ID                 string                    `json:"id"`
	Name               string                    `json:"name"`
	Description        string                    `json:"description,omitempty"`
	OwnerID            string                    `json:"owner_id,omitempty"`
	LicenseID          string                    `json:"license_id,omitempty"`
	Status             model.ModuleStatus        `json:"status"`
	Assemblies         []string                  `json:"assemblies"`
	WorkspaceInstances []model.WorkspaceInstance `json:"instances"`
	Versions           []model.ModuleVersion     `json:"versions"`
	Thumbnail          string                    `json:"thumbnail,omitempty"`
	CreatedAt          time.Time                 `json:"created_at"`
}

func (h *Handler) toModuleDTO(m *model.Model3D) moduleDTO {
	assemblies := m.Assemblies
	if assemblies == nil {
		assemblies = []string{}
	}
	wsInsts := m.WorkspaceInstances
	if wsInsts == nil {
		wsInsts = []model.WorkspaceInstance{}
	}
	versions := m.ModuleVersions
	if versions == nil {
		versions = []model.ModuleVersion{}
	}
	thumb, _ := h.thumbStore.GetThumbnail(m.ID)
	return moduleDTO{
		ID: m.ID, Name: m.Name, Description: m.Description,
		OwnerID: m.OwnerID, LicenseID: m.LicenseID, Status: m.Status,
		Assemblies: assemblies, WorkspaceInstances: wsInsts, Versions: versions,
		Thumbnail: thumb, CreatedAt: m.CreatedAt,
	}
}

// handleModules liste ou crée des modules.
//
//	@Summary		Lister ou créer des modules
//	@Description	GET liste les modules (filtre texte "q", pagination "limit"). POST crée un module en brouillon (status "draft") — un module exige au moins un assemblage avant d'être soumis (règle 14).
//	@Tags			modules
//	@Accept			json
//	@Produce		json
//	@Param			q		query		string	false	"recherche texte (nom, description) — GET uniquement"
//	@Param			limit	query		int		false	"nombre maximum de résultats (défaut 200, max 1000) — GET uniquement"
//	@Param			body	body		object	false	"name (requis), description, owner_id, channel_id, license_id — POST uniquement"
//	@Success		200	{object}	map[string]interface{}	"GET : {items, total}"
//	@Success		201	{object}	moduleDTO				"POST : module créé"
//	@Failure		400	{object}	map[string]string
//	@Security		MyrToken
//	@Router			/modules [get]
//	@Router			/modules [post]
func (h *Handler) handleModules(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		q := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("q")))
		limit := 200
		if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
		all, err := h.svcFor(r).ListModules(h.channelFor(r))
		if err != nil {
			internalErr(w, err)
			return
		}
		var filtered []*model.Model3D
		for _, p := range all {
			if q != "" && !strings.Contains(strings.ToLower(p.Name), q) &&
				!strings.Contains(strings.ToLower(p.Description), q) {
				continue
			}
			filtered = append(filtered, p)
		}
		total := len(filtered)
		if limit < len(filtered) {
			filtered = filtered[:limit]
		}
		dtos := make([]moduleDTO, 0, len(filtered))
		for _, p := range filtered {
			dtos = append(dtos, h.toModuleDTO(p))
		}
		jsonOK(w, map[string]any{"items": dtos, "total": total})
	case http.MethodPost:
		var req struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			OwnerID     string `json:"owner_id"`
			ChannelID   string `json:"channel_id"`
			LicenseID   string `json:"license_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
			jsonError(w, "le champ 'name' est requis", http.StatusBadRequest)
			return
		}
		if req.ChannelID == "" {
			req.ChannelID = h.channelFor(r)
		}
		p, err := h.svcFor(r).CreateModule(model.ModuleRequest{
			Name: req.Name, Description: req.Description,
			OwnerID: req.OwnerID, ChannelID: req.ChannelID, LicenseID: req.LicenseID,
		})
		if err != nil {
			internalErr(w, err)
			return
		}
		w.WriteHeader(http.StatusCreated)
		jsonOK(w, h.toModuleDTO(p))
	default:
		http.Error(w, "méthode non autorisée", http.StatusMethodNotAllowed)
	}
}

// handleModule route toutes les opérations sur un module identifié par :id
// et ses sous-ressources. Un seul handler Go dessert plusieurs routes REST
// distinctes (dispatch interne sur le suffixe du chemin) — voir le détail de
// chaque sous-route dans la description ci-dessous ; la spec générée
// (api/swagger.json) partage donc les mêmes réponses génériques pour toutes.
//
//	@Summary		Opérations sur un module et ses sous-ressources
//	@Description	GET/DELETE /api/modules/{id} : détail / suppression (brouillon uniquement, règle 14). GET /api/modules/{id}/interfaces : interfaces exposées. GET /api/modules/{id}/connections : connexions internes. POST/DELETE /api/modules/{id}/assemblies(/{connID}) : ajouter/retirer un assemblage. GET/POST /api/modules/{id}/thumbnail : miniature. POST /api/modules/{id}/thumbnail/regenerate : redériver la miniature depuis le lien source (og:image). GET/POST /api/modules/{id}/instances : lister / ajouter une instance de composant. DELETE/PATCH /api/modules/{id}/instances/{instanceId} : retirer (cascade des connexions, règle 15) / repositionner une instance. POST /api/modules/{id}/submit : soumettre le module à la blockchain (règle 14).
//	@Tags			modules
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"identifiant du module"
//	@Success		200	{object}	moduleDTO
//	@Success		201	{object}	moduleDTO
//	@Success		204	"pas de contenu (DELETE)"
//	@Failure		400	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Security		MyrToken
//	@Router			/modules/{id} [get]
//	@Router			/modules/{id} [delete]
//	@Router			/modules/{id}/interfaces [get]
//	@Router			/modules/{id}/connections [get]
//	@Router			/modules/{id}/assemblies/{connID} [delete]
//	@Router			/modules/{id}/thumbnail [get]
//	@Router			/modules/{id}/thumbnail [post]
//	@Router			/modules/{id}/thumbnail/regenerate [post]
//	@Router			/modules/{id}/instances [get]
//	@Router			/modules/{id}/instances [post]
//	@Router			/modules/{id}/instances/{instanceId} [delete]
//	@Router			/modules/{id}/instances/{instanceId} [patch]
//	@Router			/modules/{id}/submit [post]
func (h *Handler) handleModule(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/modules/")
	if rest == "" {
		http.NotFound(w, r)
		return
	}

	// /api/modules/:id/interfaces — interfaces exposées (non connectées en interne)
	if strings.HasSuffix(rest, "/interfaces") {
		id := strings.TrimSuffix(rest, "/interfaces")
		ifaces, err := h.svcFor(r).GetModuleInterfaces(id)
		if err != nil {
			internalErr(w, err)
			return
		}
		if ifaces == nil {
			ifaces = []*model.AssetInterface{}
		}
		jsonOK(w, ifaces)
		return
	}

	// /api/modules/:id/connections — connexions internes (assembly) du module
	if strings.HasSuffix(rest, "/connections") && r.Method == http.MethodGet {
		id := strings.TrimSuffix(rest, "/connections")
		mod, err := h.svcFor(r).GetModule(id)
		if err != nil {
			jsonError(w, err.Error(), http.StatusNotFound)
			return
		}
		asmSet := map[string]bool{}
		for _, aid := range mod.Assemblies {
			asmSet[aid] = true
		}
		all, _ := h.svcFor(r).ListConnections()
		var dtos []connectionDTO
		for _, c := range all {
			if asmSet[c.ID] {
				dtos = append(dtos, connectionDTO{
					ID: c.ID, From: c.From, To: c.To, Label: c.Label,
					FromIfaceID: c.FromIfaceID, ToIfaceID: c.ToIfaceID,
					FromInstanceID: c.FromInstanceID, ToInstanceID: c.ToInstanceID,
					FastenerAssetID: c.FastenerAssetID, Incompatible: c.Incompatible,
				})
			}
		}
		if dtos == nil {
			dtos = []connectionDTO{}
		}
		jsonOK(w, dtos)
		return
	}

	// /api/modules/:id/assemblies
	if strings.HasSuffix(rest, "/assemblies") {
		id := strings.TrimSuffix(rest, "/assemblies")
		h.handleModuleAssemblies(w, r, id)
		return
	}
	// /api/modules/:id/assemblies/:connID
	if idx := strings.Index(rest, "/assemblies/"); idx != -1 {
		productID := rest[:idx]
		connID := rest[idx+len("/assemblies/"):]
		if r.Method != http.MethodDelete {
			http.Error(w, "méthode non autorisée", http.StatusMethodNotAllowed)
			return
		}
		if err := h.svcFor(r).RemoveAssemblyFromModule(productID, connID); err != nil {
			internalErr(w, err)
			return
		}
		p, err := h.svcFor(r).GetModule(productID)
		if err != nil {
			internalErr(w, err)
			return
		}
		jsonOK(w, h.toModuleDTO(p))
		return
	}
	// /api/modules/:id/thumbnail/regenerate — redérive la miniature depuis le lien source
	if strings.HasSuffix(rest, "/thumbnail/regenerate") {
		id := strings.TrimSuffix(rest, "/thumbnail/regenerate")
		h.regenerateThumbnail(w, r, id)
		return
	}
	// /api/modules/:id/thumbnail
	if strings.HasSuffix(rest, "/thumbnail") {
		id := strings.TrimSuffix(rest, "/thumbnail")
		switch r.Method {
		case http.MethodGet:
			thumb, _ := h.thumbStore.GetThumbnail(id)
			jsonOK(w, map[string]string{"thumbnail": thumb})
		case http.MethodPost:
			var body struct{ Thumbnail string `json:"thumbnail"` }
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				jsonError(w, "body JSON invalide", http.StatusBadRequest)
				return
			}
			if err := h.thumbStore.SaveThumbnail(id, body.Thumbnail); err != nil {
				internalErr(w, err)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "méthode non autorisée", http.StatusMethodNotAllowed)
		}
		return
	}
	// /api/modules/:id/instances — lister ou ajouter une instance d'un module
	if strings.HasSuffix(rest, "/instances") {
		id := strings.TrimSuffix(rest, "/instances")
		switch r.Method {
		case http.MethodGet:
			p, err := h.svcFor(r).GetModule(id)
			if err != nil {
				jsonError(w, err.Error(), http.StatusNotFound)
				return
			}
			jsonOK(w, h.toModuleDTO(p))
		case http.MethodPost:
			var body struct {
				AssetID string `json:"asset_id"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.AssetID == "" {
				jsonError(w, "le champ 'asset_id' est requis", http.StatusBadRequest)
				return
			}
			p, err := h.svcFor(r).AddAssetToWorkspace(id, body.AssetID)
			if err != nil {
				internalErr(w, err)
				return
			}
			jsonOK(w, h.toModuleDTO(p))
		default:
			http.Error(w, "méthode non autorisée", http.StatusMethodNotAllowed)
		}
		return
	}
	// /api/modules/:id/instances/:instanceID — retirer ou mettre à jour une instance d'un module
	if idx := strings.Index(rest, "/instances/"); idx != -1 {
		productID := rest[:idx]
		instanceID := rest[idx+len("/instances/"):]
		switch r.Method {
		case http.MethodDelete:
			p, err := h.svcFor(r).RemoveAssetFromWorkspace(productID, instanceID)
			if err != nil {
				internalErr(w, err)
				return
			}
			jsonOK(w, h.toModuleDTO(p))
		case http.MethodPatch:
			var body struct {
				X float64 `json:"x"`
				Y float64 `json:"y"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				jsonError(w, "body JSON invalide", http.StatusBadRequest)
				return
			}
			p, err := h.svcFor(r).UpdateInstancePosition(productID, instanceID, body.X, body.Y)
			if err != nil {
				internalErr(w, err)
				return
			}
			jsonOK(w, h.toModuleDTO(p))
		default:
			http.Error(w, "méthode non autorisée", http.StatusMethodNotAllowed)
		}
		return
	}
	// /api/modules/:id/submit
	if strings.HasSuffix(rest, "/submit") {
		id := strings.TrimSuffix(rest, "/submit")
		if r.Method != http.MethodPost {
			http.Error(w, "méthode non autorisée", http.StatusMethodNotAllowed)
			return
		}
		var body struct{ Note string `json:"note"` }
		json.NewDecoder(r.Body).Decode(&body)
		p, err := h.svcFor(r).SubmitModule(id, body.Note)
		if err != nil {
			internalErr(w, err)
			return
		}
		jsonOK(w, h.toModuleDTO(p))
		return
	}

	id := rest
	switch r.Method {
	case http.MethodGet:
		p, err := h.svcFor(r).GetModule(id)
		if err != nil {
			jsonError(w, err.Error(), http.StatusNotFound)
			return
		}
		jsonOK(w, h.toModuleDTO(p))
	case http.MethodDelete:
		if err := h.svcFor(r).RemoveModule(id); err != nil {
			internalErr(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "méthode non autorisée", http.StatusMethodNotAllowed)
	}
}

// handleModuleAssemblies ajoute une connexion existante à la liste d'assemblage d'un module.
//
//	@Summary	Ajouter un assemblage à un module
//	@Tags		modules
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string	true	"identifiant du module"
//	@Param		body	body		object	true	"connection_id requis"
//	@Success	200	{object}	moduleDTO
//	@Failure	400	{object}	map[string]string
//	@Security	MyrToken
//	@Router		/modules/{id}/assemblies [post]
func (h *Handler) handleModuleAssemblies(w http.ResponseWriter, r *http.Request, productID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}
	var body struct{ ConnectionID string `json:"connection_id"` }
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.ConnectionID == "" {
		jsonError(w, "le champ 'connection_id' est requis", http.StatusBadRequest)
		return
	}
	if err := h.svcFor(r).AddAssemblyToModule(productID, body.ConnectionID); err != nil {
		internalErr(w, err)
		return
	}
	p, err := h.svcFor(r).GetModule(productID)
	if err != nil {
		internalErr(w, err)
		return
	}
	jsonOK(w, h.toModuleDTO(p))
}

// ── /api/channels ────────────────────────────────────────────────────────────

// handleChannels lit ou change le canal Fabric actif de la session.
//
//	@Summary		Lire ou changer le canal actif
//	@Description	GET est public. PUT accepte X-Myr-Token ou une session déjà authentifiée et valide le canal contre la liste des canaux connus du réseau actif.
//	@Tags			admin
//	@Accept			json
//	@Produce		json
//	@Param			body	body		object	false	"{ \"channel\": \"sandbox\" } — PUT uniquement"
//	@Success		200	{object}	map[string]interface{}
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Security		MyrToken
//	@Router			/channels [get]
//	@Router			/channels [put]
func (h *Handler) handleChannels(w http.ResponseWriter, r *http.Request) {
	sess := sessionFromCtx(r)
	active := h.netInfo.Channel
	if sess != nil && sess.Channel != "" {
		active = sess.Channel
	}

	// Liste des canaux : depuis le réseau actif (multi-réseau) ou depuis netInfo (legacy)
	channels := h.channelsForNetwork(r)

	switch r.Method {
	case http.MethodGet:
		jsonOK(w, map[string]any{
			"channels":   channels,
			"active":     active,
			"network_id": h.networkFor(r),
		})

	case http.MethodPut:
		// Accepte X-Myr-Token (legacy) ou session déjà injectée par requireAuth (JWT)
		token := r.Header.Get("X-Myr-Token")
		if sess == nil {
			if h.sessions.get(token) == nil {
				jsonError(w, "authentification requise", http.StatusUnauthorized)
				return
			}
		} else {
			token = sess.Token
		}
		var body struct {
			Channel string `json:"channel"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Channel == "" {
			jsonError(w, "champ 'channel' requis", http.StatusBadRequest)
			return
		}
		// Valider que le canal est dans la liste connue (si liste non vide)
		if len(channels) > 0 {
			known := false
			for _, ch := range channels {
				if ch == body.Channel {
					known = true
					break
				}
			}
			if !known {
				jsonError(w, fmt.Sprintf("canal '%s' inconnu sur ce réseau", body.Channel), http.StatusBadRequest)
				return
			}
		}
		h.sessions.setChannel(token, body.Channel)
		jsonOK(w, map[string]any{"channel": body.Channel})

	default:
		http.Error(w, "méthode non autorisée", http.StatusMethodNotAllowed)
	}
}

// channelsForNetwork retourne la liste des canaux du réseau actif de la requête.
// Multi-réseau : depuis networkSvc. Legacy : depuis netInfo.
func (h *Handler) channelsForNetwork(r *http.Request) []string {
	if h.networkSvc != nil {
		networkID := h.networkFor(r)
		if profiles, err := h.networkSvc.List(); err == nil {
			for _, p := range profiles {
				if p.ID == networkID {
					return p.Channels
				}
			}
		}
	}
	if len(h.netInfo.Channels) > 0 {
		return h.netInfo.Channels
	}
	// Pas de réseau configuré → canal unique ou aucune contrainte
	if h.netInfo.Channel != "" {
		return []string{h.netInfo.Channel}
	}
	return nil // aucun réseau → pas de validation du canal
}

// ── /api/ping ────────────────────────────────────────────────────────────────

// handlePing est un endpoint de liveness minimal (aucun appel Fabric).
//
//	@Summary	Liveness
//	@Tags		infra
//	@Produce	json
//	@Success	200	{object}	map[string]interface{}
//	@Router		/ping [get]
func (h *Handler) handlePing(w http.ResponseWriter, r *http.Request) {
	jsonOK(w, map[string]any{"ok": true, "version": h.version, "build_date": h.buildDate})
}

// ── /api/status ──────────────────────────────────────────────────────────────

// handleStatus renvoie l'état du serveur (mode, connexion Fabric, nombre d'assets).
//
//	@Summary	Statut du serveur
//	@Tags		infra
//	@Produce	json
//	@Success	200	{object}	statusResponse
//	@Router		/status [get]
func (h *Handler) handleStatus(w http.ResponseWriter, r *http.Request) {
	channel := h.channelFor(r)
	var assetCount int
	if all, err := h.svcFor(r).List(channel); err == nil {
		assetCount = len(all)
	}
	jsonOK(w, statusResponse{
		Mode:      "blockchain",
		Connected: h.fabricConnected,
		Assets:    assetCount,
		Channel:   channel,
		Peer:      h.netInfo.Peer,
		Network:   h.netInfo.Network,
	})
}

// ── /api/components/:id/interfaces ───────────────────────────────────────────────

// handleComponentInterfaces liste ou ajoute les interfaces physiques d'un composant.
//
//	@Summary		Interfaces d'un composant
//	@Description	Tant que l'asset porteur est en brouillon, les interfaces sont éditées librement en local et ne rejoignent la blockchain qu'à la soumission de l'asset (règle 27).
//	@Tags			components
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string				true	"identifiant du composant"
//	@Param			body	body		model.AssetInterface	false	"interface à ajouter — POST uniquement"
//	@Success		200	{array}		model.AssetInterface
//	@Success		201	{object}	model.AssetInterface
//	@Failure		400	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		MyrToken
//	@Router			/components/{id}/interfaces [get]
//	@Router			/components/{id}/interfaces [post]
func (h *Handler) handleComponentInterfaces(w http.ResponseWriter, r *http.Request, assetID string) {
	switch r.Method {
	case http.MethodGet:
		ifaces, err := h.svcFor(r).ListInterfacesForAsset(assetID)
		if err != nil {
			internalErr(w, err)
			return
		}
		if ifaces == nil {
			ifaces = []*model.AssetInterface{}
		}
		jsonOK(w, ifaces)
	case http.MethodPost:
		var iface model.AssetInterface
		if err := json.NewDecoder(r.Body).Decode(&iface); err != nil {
			jsonError(w, "JSON invalide: "+err.Error(), http.StatusBadRequest)
			return
		}
		iface.AssetID = assetID
		if err := h.svcFor(r).AddInterface(&iface); err != nil {
			internalErr(w, err)
			return
		}
		w.WriteHeader(http.StatusCreated)
		jsonOK(w, &iface)
	default:
		http.Error(w, "méthode non autorisée", http.StatusMethodNotAllowed)
	}
}

// ── /api/interfaces/:id ──────────────────────────────────────────────────────

// handleInterface modifie ou supprime une interface existante.
//
//	@Summary		Modifier ou supprimer une interface
//	@Description	DELETE maintient l'invariant qu'un asset possède toujours au moins un slot virtuel (règle 13) : un nouveau slot virtuel est recréé si l'interface supprimée était physique et la dernière restante.
//	@Tags			components
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"identifiant de l'interface"
//	@Param			body	body		model.AssetInterface	false	"interface mise à jour — PATCH uniquement"
//	@Success		200	{object}	model.AssetInterface
//	@Success		204	"pas de contenu (DELETE)"
//	@Failure		400	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Security		MyrToken
//	@Router			/interfaces/{id} [patch]
//	@Router			/interfaces/{id} [delete]
func (h *Handler) handleInterface(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/interfaces/")
	if id == "" {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodDelete:
		// Récupérer l'interface avant suppression pour connaître l'assetID
		existing, _ := h.svcFor(r).GetInterface(id)
		if err := h.svcFor(r).RemoveInterface(id); err != nil {
			internalErr(w, err)
			return
		}
		// Maintenir l'invariant : au moins un slot virtuel par asset
		if existing != nil && !existing.Virtual {
			h.svcFor(r).EnsureVirtualSlot(existing.AssetID)
		}
		w.WriteHeader(http.StatusNoContent)
	case http.MethodPatch:
		existing, err := h.svcFor(r).GetInterface(id)
		if err != nil {
			jsonError(w, err.Error(), http.StatusNotFound)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(existing); err != nil {
			jsonError(w, "JSON invalide: "+err.Error(), http.StatusBadRequest)
			return
		}
		existing.ID = id // prevent ID overwrite
		if err := h.svcFor(r).UpdateInterface(existing); err != nil {
			internalErr(w, err)
			return
		}
		jsonOK(w, existing)
	default:
		http.Error(w, "méthode non autorisée", http.StatusMethodNotAllowed)
	}
}

// ── /api/refs ────────────────────────────────────────────────────────────────

// handleRefs renvoie le vocabulaire de référence (catégories, types, unités).
//
//	@Summary	Vocabulaire de référence des interfaces
//	@Tags		refs
//	@Produce	json
//	@Success	200	{object}	map[string]interface{}
//	@Failure	500	{object}	map[string]string
//	@Security	MyrToken
//	@Router		/refs [get]
func (h *Handler) handleRefs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}
	refs, err := h.svcFor(r).GetRefs()
	if err != nil {
		internalErr(w, err)
		return
	}
	jsonOK(w, refs)
}

// handleRefCategories ajoute une catégorie au référentiel d'interfaces.
//
//	@Summary	Ajouter une catégorie de référence
//	@Tags		refs
//	@Accept		json
//	@Produce	json
//	@Param		body	body		object	true	"name requis"
//	@Success	201	{object}	map[string]interface{}
//	@Failure	400	{object}	map[string]string
//	@Security	MyrToken
//	@Router		/refs/categories [post]
func (h *Handler) handleRefCategories(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		jsonError(w, "le champ 'name' est requis", http.StatusBadRequest)
		return
	}
	if err := h.svcFor(r).AddRefCategory(body.Name); err != nil {
		internalErr(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	refs, _ := h.svcFor(r).GetRefs()
	jsonOK(w, refs)
}

// handleRefTypes ajoute un type au référentiel d'interfaces, pour une catégorie donnée.
//
//	@Summary	Ajouter un type de référence
//	@Tags		refs
//	@Accept		json
//	@Produce	json
//	@Param		body	body		object	true	"category et name requis"
//	@Success	201	{object}	map[string]interface{}
//	@Failure	400	{object}	map[string]string
//	@Security	MyrToken
//	@Router		/refs/types [post]
func (h *Handler) handleRefTypes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Category string `json:"category"`
		Name     string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Category == "" || body.Name == "" {
		jsonError(w, "les champs 'category' et 'name' sont requis", http.StatusBadRequest)
		return
	}
	if err := h.svcFor(r).AddRefType(body.Category, body.Name); err != nil {
		internalErr(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	refs, _ := h.svcFor(r).GetRefs()
	jsonOK(w, refs)
}

// handleRefUnits ajoute une unité au référentiel d'interfaces, pour une catégorie donnée.
//
//	@Summary	Ajouter une unité de référence
//	@Tags		refs
//	@Accept		json
//	@Produce	json
//	@Param		body	body		object	true	"category et name requis"
//	@Success	201	{object}	map[string]interface{}
//	@Failure	400	{object}	map[string]string
//	@Security	MyrToken
//	@Router		/refs/units [post]
func (h *Handler) handleRefUnits(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Category string `json:"category"`
		Name     string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Category == "" || body.Name == "" {
		jsonError(w, "les champs 'category' et 'name' sont requis", http.StatusBadRequest)
		return
	}
	if err := h.svcFor(r).AddRefUnit(body.Category, body.Name); err != nil {
		internalErr(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	refs, _ := h.svcFor(r).GetRefs()
	jsonOK(w, refs)
}

// ── helpers ───────────────────────────────────────────────────────────────────

func readJSON(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

func jsonOK(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func methodNotAllowed(w http.ResponseWriter) {
	jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// internalErr logue l'erreur complète côté serveur et renvoie un message générique au client.
// Distingue l'indisponibilité de la blockchain (503, transitoire — pas une erreur de
// données) des autres échecs du service domaine (500).
func internalErr(w http.ResponseWriter, err error) {
	log.Printf("erreur interne : %v", err)
	if errors.Is(err, model.ErrBlockchainUnavailable) {
		jsonError(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	jsonError(w, err.Error(), http.StatusInternalServerError)
}

func parseTags(raw string) []string {
	if raw == "" {
		return []string{}
	}
	parts := strings.Split(raw, ",")
	tags := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			tags = append(tags, t)
		}
	}
	return tags
}

// ── /api/licenses ─────────────────────────────────────────────────────────────

// GET  /api/licenses           → liste complète du catalogue
// GET  /api/licenses/:id       → une licence
// POST /api/licenses/check     → { parent_license_id, proposed_license_id } → LicenseCheck
// POST /api/licenses/check-product → { component_license_ids[], proposed_module_license_id } → LicenseCheck

// handleLicenses liste le catalogue complet des licences.
//
//	@Summary	Catalogue des licences
//	@Tags		licenses
//	@Produce	json
//	@Success	200	{array}	model.License
//	@Security	MyrToken
//	@Router		/licenses [get]
func (h *Handler) handleLicenses(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}
	jsonOK(w, h.svcFor(r).ListLicenses())
}

// handleLicense route le détail d'une licence ainsi que les vérifications de
// compatibilité (règle 8) sous /api/licenses/{...}.
//
//	@Summary		Détail d'une licence ou vérification de compatibilité
//	@Description	GET /api/licenses/{id} : détail. POST /api/licenses/check : { parent_license_id, proposed_license_id } → compatibilité pour une dérivation de composant. POST /api/licenses/check-product : { component_license_ids[], proposed_module_license_id } → compatibilité pour un module assemblant plusieurs composants.
//	@Tags			licenses
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string	false	"identifiant de la licence — GET /api/licenses/{id}"
//	@Param			body	body		object	false	"corps de vérification — POST /api/licenses/check(-product)"
//	@Success		200	{object}	model.License
//	@Failure		400	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Security		MyrToken
//	@Router			/licenses/{id} [get]
//	@Router			/licenses/check [post]
//	@Router			/licenses/check-product [post]
func (h *Handler) handleLicense(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/licenses/")
	if rest == "" {
		http.NotFound(w, r)
		return
	}
	// /api/licenses/check
	if rest == "check" {
		if r.Method != http.MethodPost {
			http.Error(w, "méthode non autorisée", http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			ParentLicenseID   string `json:"parent_license_id"`
			ProposedLicenseID string `json:"proposed_license_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			jsonError(w, "JSON invalide: "+err.Error(), http.StatusBadRequest)
			return
		}
		jsonOK(w, h.svcFor(r).CheckLicenseCompatibility(body.ParentLicenseID, body.ProposedLicenseID))
		return
	}
	// /api/licenses/check-product
	if rest == "check-product" {
		if r.Method != http.MethodPost {
			http.Error(w, "méthode non autorisée", http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			ComponentLicenseIDs      []string `json:"component_license_ids"`
			ProposedModuleLicenseID string   `json:"proposed_module_license_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			jsonError(w, "JSON invalide: "+err.Error(), http.StatusBadRequest)
			return
		}
		jsonOK(w, h.svcFor(r).CheckModuleLicenseCompatibility(body.ComponentLicenseIDs, body.ProposedModuleLicenseID))
		return
	}
	// /api/licenses/:id
	if r.Method != http.MethodGet {
		http.Error(w, "méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}
	l, err := h.svcFor(r).GetLicense(rest)
	if err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}
	jsonOK(w, l)
}

