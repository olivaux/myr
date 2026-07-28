// domain/model/service.go — logique métier, ne connaît QUE les interfaces
package model

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"
)

type Service struct {
	blockchain     BlockchainPort // nullable — nil si Fabric indisponible (voir ErrBlockchainUnavailable)
	fileStorage    FileStoragePort
	connStore      ConnectionStore // optionnel — nil hors mode GUI
	thumbStore     ThumbnailStore  // optionnel — nil hors mode GUI
	ifaceStore     InterfaceStore  // optionnel — nil hors mode GUI
	draftStore     DraftStore      // optionnel — requis uniquement pour AddRequest.Draft / Submit
	ogImageFetcher OGImageFetcher  // optionnel — nil si non câblé (pas d'accès réseau depuis les tests domaine)
}

func NewService(bc BlockchainPort, fs FileStoragePort) *Service {
	return &Service{blockchain: bc, fileStorage: fs}
}

// SetBlockchain remplace la blockchain active (utilisé après reconnexion).
func (s *Service) SetBlockchain(bc BlockchainPort) { s.blockchain = bc }

// WithConnStore attache un store de connexions (mode GUI).
func (s *Service) WithConnStore(cs ConnectionStore) *Service {
	s.connStore = cs
	return s
}

// WithThumbStore attache un store de miniatures (mode GUI).
func (s *Service) WithThumbStore(ts ThumbnailStore) *Service {
	s.thumbStore = ts
	return s
}

// WithIfaceStore attache un store d'interfaces physiques (mode GUI).
func (s *Service) WithIfaceStore(is InterfaceStore) *Service {
	s.ifaceStore = is
	return s
}

// WithDraftStore attache un store de brouillons (requis pour AddRequest.Draft et Submit).
func (s *Service) WithDraftStore(ds DraftStore) *Service {
	s.draftStore = ds
	return s
}

// WithOGImageFetcher attache la source de régénération de miniature depuis un lien web.
func (s *Service) WithOGImageFetcher(f OGImageFetcher) *Service {
	s.ogImageFetcher = f
	return s
}

// Add crée un asset minimal (compatibilité CLI / TUI).
func (s *Service) Add(filePath, name, channelID, ownerID string, tags []string) (*Model3D, error) {
	return s.AddFull(AddRequest{
		FilePath:  filePath,
		Name:      name,
		ChannelID: channelID,
		OwnerID:   ownerID,
		Tags:      tags,
		Category:  CategoryBase,
	})
}

// AddFull crée un asset avec tous ses champs (mode GUI).
// Si req.Draft, l'asset est stocké dans DraftStore (aucune transaction blockchain) —
// il ne rejoint la blockchain qu'à un appel explicite à Submit.
func (s *Service) AddFull(req AddRequest) (*Model3D, error) {
	if !req.Draft && s.blockchain == nil {
		return nil, ErrBlockchainUnavailable
	}
	// Vérification de compatibilité de licence avec le parent (si renseigné).
	if req.ParentID != "" && req.LicenseID != "" {
		if s.blockchain == nil {
			return nil, ErrBlockchainUnavailable
		}
		parent, err := s.blockchain.GetModelRecord(req.ParentID, req.ChannelID)
		if err == nil && parent.LicenseID != "" {
			check := CheckLicenseCompatibility(parent.LicenseID, req.LicenseID)
			if !check.Compatible {
				return nil, fmt.Errorf("incompatibilité de licence : %s", check.Reason)
			}
		}
	}

	m := &Model3D{
		ID:          generateID(),
		Name:        req.Name,
		Description: req.Description,
		Category:    req.Category,
		ParentID:    req.ParentID,
		ChannelID:   req.ChannelID,
		OwnerID:     req.OwnerID,
		LicenseID:   req.LicenseID,
		Tags:        req.Tags,
		Links:       req.Links,
		CreatedAt:   time.Now(),
	}

	if req.FilePath != "" {
		hash, err := s.hashFile(req.FilePath)
		if err != nil {
			return nil, fmt.Errorf("hashing file: %w", err)
		}
		storageRef, err := s.fileStorage.Upload(req.FilePath)
		if err != nil {
			return nil, fmt.Errorf("uploading file: %w", err)
		}
		m.Hash = hash
		m.Versions = []Version{{Number: 1, Hash: storageRef, CreatedAt: time.Now()}}
	}

	if req.Draft {
		if s.draftStore == nil {
			return nil, fmt.Errorf("stockage de brouillons non configuré")
		}
		m.Status = ModuleDraft
		if err := s.draftStore.SaveDraft(m); err != nil {
			return nil, fmt.Errorf("saving draft: %w", err)
		}
		return m, nil
	}

	if err := s.blockchain.StoreModelRecord(m); err != nil {
		return nil, fmt.Errorf("storing on blockchain: %w", err)
	}

	return m, nil
}

// Submit engage sur la blockchain un composant créé en brouillon (AddRequest.Draft) :
// une seule transaction committe les métadonnées et les interfaces locales
// (Model3D.Interfaces, depuis InterfaceStore), puis Status passe à submitted.
// Généralise SubmitModule à tout Model3D, sans la vérification d'assemblage
// (RM17, propre aux modules — voir DC_CLI_Model.md §3.6bis).
func (s *Service) Submit(assetID string) (*Model3D, error) {
	if s.draftStore == nil {
		return nil, fmt.Errorf("stockage de brouillons non configuré")
	}
	if s.blockchain == nil {
		return nil, ErrBlockchainUnavailable
	}
	m, err := s.draftStore.GetDraft(assetID)
	if err != nil {
		return nil, fmt.Errorf("brouillon introuvable : %w", err)
	}
	if s.ifaceStore != nil {
		ifaces, err := s.ifaceStore.ListInterfacesForAsset(assetID)
		if err != nil {
			return nil, fmt.Errorf("listing interfaces: %w", err)
		}
		m.Interfaces = make([]AssetInterface, len(ifaces))
		for i, iface := range ifaces {
			m.Interfaces[i] = *iface
		}
	}
	m.Status = ModuleSubmitted
	if err := s.blockchain.StoreModelRecord(m); err != nil {
		return nil, fmt.Errorf("storing on blockchain: %w", err)
	}
	_ = s.draftStore.RemoveDraft(assetID)
	return m, nil
}

// Get récupère un modèle par son ID sur le canal indiqué.
// channelID="" utilise le canal par défaut configuré dans l'adapter.
// Cherche d'abord dans DraftStore (asset pas encore soumis), puis sur la blockchain.
func (s *Service) Get(id, channelID string) (*Model3D, error) {
	if s.draftStore != nil {
		if m, err := s.draftStore.GetDraft(id); err == nil {
			return m, nil
		}
	}
	if s.blockchain == nil {
		return nil, ErrBlockchainUnavailable
	}
	return s.blockchain.GetModelRecord(id, channelID)
}

// List retourne les brouillons locaux (DraftStore) suivis des assets de la blockchain.
func (s *Service) List(channelID string) ([]*Model3D, error) {
	var drafts []*Model3D
	if s.draftStore != nil {
		drafts, _ = s.draftStore.ListDrafts(channelID)
	}
	if s.blockchain == nil {
		if len(drafts) > 0 {
			return drafts, nil
		}
		return nil, ErrBlockchainUnavailable
	}
	onChain, err := s.blockchain.ListModelRecords(channelID)
	if err != nil {
		return nil, err
	}
	return append(drafts, onChain...), nil
}

// Verify vérifie l'intégrité d'un modèle sur le canal indiqué.
// channelID="" utilise le canal par défaut configuré dans l'adapter.
func (s *Service) Verify(id, channelID string) (bool, error) {
	if s.blockchain == nil {
		return false, ErrBlockchainUnavailable
	}
	record, err := s.blockchain.GetModelRecord(id, channelID)
	if err != nil {
		return false, err
	}
	return s.blockchain.VerifyIntegrity(id, record.Hash, channelID)
}

// ── Connexions d'assemblage (requiert connStore) ────────────────────────────

func (s *Service) AddConnection(from, to, label string) (*Connection, error) {
	if s.connStore == nil {
		return nil, fmt.Errorf("connection store non configuré")
	}
	conn := &Connection{ID: generateID(), From: from, To: to, Label: label}
	return conn, s.connStore.SaveConnection(conn)
}

// AddAssemblyLink crée un lien d'assemblage précis entre deux interfaces physiques.
// fastenerAssetID est l'asset d'accroche (vis, câble, clip…) ; vide si connexion directe.
// Les assets source et cible sont déduits des interfaces — le domaine reste l'autorité.
func (s *Service) AddAssemblyLink(fromIfaceID, toIfaceID, label, fromInstanceID, toInstanceID, fastenerAssetID string) (*Connection, error) {
	if s.ifaceStore == nil {
		return nil, fmt.Errorf("interface store non configuré")
	}
	if s.connStore == nil {
		return nil, fmt.Errorf("connection store non configuré")
	}
	fromIface, err := s.ifaceStore.GetInterface(fromIfaceID)
	if err != nil {
		return nil, fmt.Errorf("interface source: %w", err)
	}
	toIface, err := s.ifaceStore.GetInterface(toIfaceID)
	if err != nil {
		return nil, fmt.Errorf("interface cible: %w", err)
	}

	// Validation de l'accroche : elle doit avoir au moins une interface
	// compatible avec chacun des deux endpoints.
	if fastenerAssetID != "" {
		if err := s.validateFastener(fastenerAssetID, fromIface, toIface); err != nil {
			return nil, fmt.Errorf("asset d'accroche invalide : %w", err)
		}
	}

	conn := &Connection{
		ID:              generateID(),
		From:            fromIface.AssetID,
		To:              toIface.AssetID,
		Label:           label,
		FromIfaceID:     fromIfaceID,
		ToIfaceID:       toIfaceID,
		FromInstanceID:  fromInstanceID,
		ToInstanceID:    toInstanceID,
		FastenerAssetID: fastenerAssetID,
	}
	return conn, s.connStore.SaveConnection(conn)
}

// validateFastener vérifie qu'un asset d'accroche est physiquement compatible
// avec les deux interfaces qu'il doit relier.
func (s *Service) validateFastener(fastenerID string, fromIface, toIface *AssetInterface) error {
	if s.ifaceStore == nil {
		return nil
	}
	fasIfaces, err := s.ifaceStore.ListInterfacesForAsset(fastenerID)
	if err != nil {
		return fmt.Errorf("interfaces introuvables : %w", err)
	}
	if len(fasIfaces) == 0 {
		return fmt.Errorf("l'asset d'accroche n'a aucune interface définie")
	}
	fromOK, toOK := false, false
	for _, fi := range fasIfaces {
		if ifacesCompatible(fi, fromIface) {
			fromOK = true
		}
		if ifacesCompatible(fi, toIface) {
			toOK = true
		}
	}
	if !fromOK {
		return fmt.Errorf("aucune interface de l'accroche n'est compatible avec l'interface source")
	}
	if !toOK {
		return fmt.Errorf("aucune interface de l'accroche n'est compatible avec l'interface cible")
	}
	return nil
}

// ifacesCompatible retourne true si deux interfaces physiques peuvent être reliées.
// Règles : même catégorie + même type ; sens cohérent (out↔in ou bidir) ; plages de valeurs compatibles.
func ifacesCompatible(a, b *AssetInterface) bool {
	if a.Category != b.Category || a.Type != b.Type {
		return false
	}
	dirOK := (a.Direction == IfaceOut && b.Direction == IfaceIn) ||
		(a.Direction == IfaceIn && b.Direction == IfaceOut) ||
		a.Direction == IfaceBidi || b.Direction == IfaceBidi
	if !dirOK {
		return false
	}
	// Choisir qui est "out" pour le contrôle de plage
	if a.Direction == IfaceOut || a.Direction == IfaceBidi {
		return ifaceValuesOverlap(a, b)
	}
	return ifaceValuesOverlap(b, a)
}

// ifaceValuesOverlap vérifie que les plages de valeurs de deux interfaces se chevauchent.
// out → fournisseur, in → consommateur.
func ifaceValuesOverlap(out, in *AssetInterface) bool {
	if (out.ValueMin == 0 && out.ValueMax == 0) || (in.ValueMin == 0 && in.ValueMax == 0) {
		return true // pas de valeur définie = pas de contrainte
	}
	return out.ValueMin <= in.ValueMax && in.ValueMin <= out.ValueMax
}

func (s *Service) RemoveConnection(id string) error {
	if s.connStore == nil {
		return fmt.Errorf("connection store non configuré")
	}
	return s.connStore.RemoveConnection(id)
}

func (s *Service) ListConnections() ([]*Connection, error) {
	if s.connStore == nil {
		return nil, nil
	}
	return s.connStore.ListConnections()
}

func (s *Service) GetChildren(parentID string) ([]*Model3D, error) {
	if s.blockchain == nil {
		return nil, ErrBlockchainUnavailable
	}
	all, err := s.blockchain.ListModelRecords("")
	if err != nil {
		return nil, err
	}
	var children []*Model3D
	for _, m := range all {
		if m.ParentID == parentID {
			children = append(children, m)
		}
	}
	return children, nil
}

// ── Miniatures STL (requiert thumbStore) ────────────────────────────────────

func (s *Service) SaveThumbnail(assetID, dataURL string) error {
	if s.thumbStore == nil {
		return fmt.Errorf("thumbnail store non configuré")
	}
	return s.thumbStore.SaveThumbnail(assetID, dataURL)
}

func (s *Service) GetThumbnail(assetID string) (string, error) {
	if s.thumbStore == nil {
		return "", nil
	}
	return s.thumbStore.GetThumbnail(assetID)
}

// RegenerateThumbnail redérive la miniature d'un asset depuis sa seule source durable
// accessible côté serveur : l'og:image du premier lien externe enregistré (Links).
// Un modèle 3D n'a pas de source régénérable côté serveur (pas de moteur de rendu 3D
// dans le domaine) — sa miniature est produite par le rendu client (GUI) et poussée
// via SaveThumbnail ; RegenerateThumbnail échoue avec ErrNoThumbnailSource si l'asset
// n'a aucun lien externe.
func (s *Service) RegenerateThumbnail(assetID string) (string, error) {
	if s.thumbStore == nil {
		return "", fmt.Errorf("thumbnail store non configuré")
	}
	if s.ogImageFetcher == nil {
		return "", fmt.Errorf("aucune source de régénération de miniature configurée")
	}
	m, err := s.Get(assetID, "")
	if err != nil {
		return "", err
	}
	if len(m.Links) == 0 {
		return "", ErrNoThumbnailSource
	}
	dataURL, err := s.ogImageFetcher.FetchOGImage(m.Links[0])
	if err != nil {
		return "", fmt.Errorf("récupération og:image: %w", err)
	}
	if err := s.thumbStore.SaveThumbnail(assetID, dataURL); err != nil {
		return "", err
	}
	return dataURL, nil
}

// ── Interfaces physiques (requiert ifaceStore) ───────────────────────────────

// #incoherence — AddInterface/UpdateInterface/RemoveInterface ne vérifient pas
// si l'asset porteur est déjà submitted avant de muter ses interfaces (RM19) —
// voir specs/roadmap_dev.md § Écarts — revue de code, E3.
func (s *Service) AddInterface(iface *AssetInterface) error {
	if s.ifaceStore == nil {
		return fmt.Errorf("interface store non configuré")
	}
	if iface.ID == "" {
		iface.ID = generateID()
	}
	return s.ifaceStore.SaveInterface(iface)
}

// UpdateInterface met à jour une interface existante et marque les connexions
// qui l'utilisent comme incompatibles si elles ne satisfont plus les contraintes.
func (s *Service) UpdateInterface(iface *AssetInterface) error {
	if s.ifaceStore == nil {
		return fmt.Errorf("interface store non configuré")
	}
	if err := s.ifaceStore.SaveInterface(iface); err != nil {
		return err
	}
	if s.connStore == nil {
		return nil
	}
	conns, err := s.connStore.ListConnections()
	if err != nil {
		return nil // non bloquant
	}
	for _, c := range conns {
		if c.FromIfaceID != iface.ID && c.ToIfaceID != iface.ID {
			continue
		}
		fromIface, err := s.ifaceStore.GetInterface(c.FromIfaceID)
		if err != nil {
			continue
		}
		toIface, err := s.ifaceStore.GetInterface(c.ToIfaceID)
		if err != nil {
			continue
		}
		wasIncompatible := c.Incompatible
		c.Incompatible = !ifacesCompatible(fromIface, toIface)
		if c.Incompatible != wasIncompatible {
			_ = s.connStore.UpdateConnection(c)
		}
	}
	return nil
}

func (s *Service) RemoveInterface(id string) error {
	if s.ifaceStore == nil {
		return fmt.Errorf("interface store non configuré")
	}
	return s.ifaceStore.RemoveInterface(id)
}

func (s *Service) ListInterfacesForAsset(assetID string) ([]*AssetInterface, error) {
	if s.ifaceStore == nil {
		return nil, nil
	}
	return s.ifaceStore.ListInterfacesForAsset(assetID)
}

func (s *Service) GetInterface(id string) (*AssetInterface, error) {
	if s.ifaceStore == nil {
		return nil, fmt.Errorf("interface store non configuré")
	}
	return s.ifaceStore.GetInterface(id)
}

func (s *Service) GetRefs() (*InterfaceRefs, error) {
	if s.ifaceStore == nil {
		return DefaultInterfaceRefs(), nil
	}
	return s.ifaceStore.GetRefs()
}

func (s *Service) AddRefCategory(cat string) error {
	if s.ifaceStore == nil {
		return fmt.Errorf("interface store non configuré")
	}
	return s.ifaceStore.AddRefCategory(cat)
}

func (s *Service) AddRefType(cat, typeName string) error {
	if s.ifaceStore == nil {
		return fmt.Errorf("interface store non configuré")
	}
	return s.ifaceStore.AddRefType(cat, typeName)
}

func (s *Service) AddRefUnit(cat, unit string) error {
	if s.ifaceStore == nil {
		return fmt.Errorf("interface store non configuré")
	}
	return s.ifaceStore.AddRefUnit(cat, unit)
}

// ── Suppression ─────────────────────────────────────────────────────────────

// removeConnectionsFor supprime en cascade les connexions impliquant un asset (règle 15).
func (s *Service) removeConnectionsFor(id string) {
	if s.connStore == nil {
		return
	}
	conns, _ := s.connStore.ListConnections()
	for _, c := range conns {
		if c.From == id || c.To == id {
			_ = s.connStore.RemoveConnection(c.ID)
		}
	}
}

func (s *Service) Remove(id string) error {
	if s.draftStore != nil {
		if _, err := s.draftStore.GetDraft(id); err == nil {
			s.removeConnectionsFor(id)
			return s.draftStore.RemoveDraft(id)
		}
	}
	if s.blockchain == nil {
		return ErrBlockchainUnavailable
	}
	s.removeConnectionsFor(id)
	// Fabric ne supporte pas la suppression — on délègue à l'adapter
	// qui peut retourner ErrNotSupported si besoin.
	// #incoherence — ErrNotSupported n'existe nulle part dans domain/model
	// malgré ce commentaire et UCAM05.md — voir specs/roadmap_dev.md § Écarts
	// — revue de code, E4.
	type remover interface{ RemoveModelRecord(id string) error }
	if r, ok := s.blockchain.(remover); ok {
		return r.RemoveModelRecord(id)
	}
	return fmt.Errorf("suppression non supportée par cet adapter blockchain")
}

// applyAssetPatch applique sur m les champs non-vides d'un UpdateRequest.
// #incoherence — ne réapplique pas CheckLicenseCompatibility (RM03) quand
// LicenseID change, contrairement à AddFull — voir specs/roadmap_dev.md
// § Écarts — revue de code, E2.
func applyAssetPatch(m *Model3D, req UpdateRequest) {
	if req.Name != "" {
		m.Name = req.Name
	}
	if req.Description != "" {
		m.Description = req.Description
	}
	if req.LicenseID != "" {
		m.LicenseID = req.LicenseID
	}
	if req.Tags != nil {
		m.Tags = req.Tags
	}
	if req.Links != nil {
		m.Links = req.Links
	}
}

// UpdateAsset applique un patch partiel sur un asset existant (brouillon local ou blockchain).
func (s *Service) UpdateAsset(req UpdateRequest) (*Model3D, error) {
	if s.draftStore != nil {
		if m, err := s.draftStore.GetDraft(req.ID); err == nil {
			applyAssetPatch(m, req)
			return m, s.draftStore.SaveDraft(m)
		}
	}
	if s.blockchain == nil {
		return nil, ErrBlockchainUnavailable
	}
	m, err := s.blockchain.GetModelRecord(req.ID, "")
	if err != nil {
		return nil, fmt.Errorf("asset introuvable: %w", err)
	}
	applyAssetPatch(m, req)
	return m, s.blockchain.StoreModelRecord(m)
}

// ── Modules ───────────────────────────────────────────────────────────────────

func (s *Service) CreateModule(req ModuleRequest) (*Model3D, error) {
	if s.blockchain == nil {
		return nil, ErrBlockchainUnavailable
	}
	m := &Model3D{
		ID:                 generateID(),
		Name:               req.Name,
		Description:        req.Description,
		OwnerID:            req.OwnerID,
		ChannelID:          req.ChannelID,
		LicenseID:          req.LicenseID,
		Status:             ModuleDraft,
		CreatedAt:          time.Now(),
		Assemblies:         []string{},
		WorkspaceInstances: []WorkspaceInstance{},
		ModuleVersions:     []ModuleVersion{},
	}
	return m, s.blockchain.StoreModelRecord(m)
}

func (s *Service) GetModule(id string) (*Model3D, error) {
	if s.blockchain == nil {
		return nil, ErrBlockchainUnavailable
	}
	return s.blockchain.GetModelRecord(id, "")
}

func (s *Service) ListModules(channelID string) ([]*Model3D, error) {
	if s.blockchain == nil {
		return nil, ErrBlockchainUnavailable
	}
	all, err := s.blockchain.ListModelRecords(channelID)
	if err != nil {
		return nil, err
	}
	var modules []*Model3D
	for _, m := range all {
		if m.IsModule() {
			modules = append(modules, m)
		}
	}
	return modules, nil
}

func (s *Service) AddAssemblyToModule(moduleID, connID string) error {
	if s.blockchain == nil {
		return ErrBlockchainUnavailable
	}
	m, err := s.blockchain.GetModelRecord(moduleID, "")
	if err != nil {
		return err
	}
	for _, id := range m.Assemblies {
		if id == connID {
			return nil // déjà présent
		}
	}
	m.Assemblies = append(m.Assemblies, connID)
	m.Status = ModuleDraft
	return s.blockchain.StoreModelRecord(m)
}

func (s *Service) RemoveAssemblyFromModule(moduleID, connID string) error {
	if s.blockchain == nil {
		return ErrBlockchainUnavailable
	}
	m, err := s.blockchain.GetModelRecord(moduleID, "")
	if err != nil {
		return err
	}
	filtered := m.Assemblies[:0]
	for _, id := range m.Assemblies {
		if id != connID {
			filtered = append(filtered, id)
		}
	}
	m.Assemblies = filtered
	m.Status = ModuleDraft
	return s.blockchain.StoreModelRecord(m)
}

// SubmitModule soumet le module à la blockchain et crée une version immuable.
func (s *Service) SubmitModule(moduleID, note string) (*Model3D, error) {
	if s.blockchain == nil {
		return nil, ErrBlockchainUnavailable
	}
	m, err := s.blockchain.GetModelRecord(moduleID, "")
	if err != nil {
		return nil, err
	}
	if len(m.Assemblies) == 0 {
		return nil, fmt.Errorf("le module ne contient aucun assemblage")
	}

	hash := computeModuleHash(m.ID, len(m.ModuleVersions)+1, m.Assemblies)
	blockID := simModuleBlockID()

	v := ModuleVersion{
		Number:     len(m.ModuleVersions) + 1,
		Assemblies: append([]string{}, m.Assemblies...),
		Hash:       hash,
		Note:       note,
		CreatedAt:  time.Now(),
		BlockID:    blockID,
	}
	m.ModuleVersions = append(m.ModuleVersions, v)
	m.Status = ModuleSubmitted
	return m, s.blockchain.StoreModelRecord(m)
}

// AddAssetToWorkspace ajoute une nouvelle instance de l'asset au module.
// Invariant : chaque asset dispose toujours d'au moins une interface virtuelle
// (slot utilisable pour créer une liaison). Un slot est créé automatiquement
// si aucune interface virtuelle n'existe encore pour cet asset.
// #incoherence — aucune garde contre assetID == moduleID ni contre un cycle
// A→B→A ; combiné à getModuleInterfacesInto ci-dessous, une composition
// cyclique fait récurser indéfiniment GetModuleInterfaces (stack overflow) —
// voir specs/roadmap_dev.md § Écarts — revue de code, E1.
func (s *Service) AddAssetToWorkspace(moduleID, assetID string) (*Model3D, error) {
	if s.blockchain == nil {
		return nil, ErrBlockchainUnavailable
	}
	m, err := s.blockchain.GetModelRecord(moduleID, "")
	if err != nil {
		return nil, err
	}
	inst := WorkspaceInstance{ID: generateID(), AssetID: assetID}
	m.WorkspaceInstances = append(m.WorkspaceInstances, inst)
	if err := s.blockchain.StoreModelRecord(m); err != nil {
		return nil, err
	}
	s.ensureVirtualSlot(assetID)
	return m, nil
}

// EnsureVirtualSlot garantit qu'au moins une interface virtuelle existe pour cet asset.
// Idempotent. Sans effet si le store n'est pas configuré ou si un slot existe déjà.
func (s *Service) EnsureVirtualSlot(assetID string) {
	if s.ifaceStore == nil {
		return
	}
	ifaces, err := s.ifaceStore.ListInterfacesForAsset(assetID)
	if err != nil {
		return
	}
	for _, iface := range ifaces {
		if iface.Virtual {
			return // slot déjà présent
		}
	}
	_ = s.ifaceStore.SaveInterface(&AssetInterface{
		ID:      generateID(),
		AssetID: assetID,
		Virtual: true,
	})
}

// ensureVirtualSlot est l'alias interne non-exporté.
func (s *Service) ensureVirtualSlot(assetID string) { s.EnsureVirtualSlot(assetID) }

// RemoveAssetFromWorkspace retire une instance spécifique (par instanceID) du module.
// Supprime en cascade les connexions de cette instance dans le module.
func (s *Service) RemoveAssetFromWorkspace(moduleID, instanceID string) (*Model3D, error) {
	if s.blockchain == nil {
		return nil, ErrBlockchainUnavailable
	}
	m, err := s.blockchain.GetModelRecord(moduleID, "")
	if err != nil {
		return nil, err
	}

	if s.connStore != nil {
		asmSet := make(map[string]bool, len(m.Assemblies))
		for _, id := range m.Assemblies {
			asmSet[id] = true
		}
		conns, _ := s.connStore.ListConnections()
		survivingAsm := append([]string{}, m.Assemblies...)
		for _, c := range conns {
			if !asmSet[c.ID] {
				continue
			}
			if c.FromInstanceID == instanceID || c.ToInstanceID == instanceID {
				next := survivingAsm[:0]
				for _, id := range survivingAsm {
					if id != c.ID {
						next = append(next, id)
					}
				}
				survivingAsm = next
				_ = s.connStore.RemoveConnection(c.ID)
			}
		}
		m.Assemblies = survivingAsm
	}

	filtered := m.WorkspaceInstances[:0]
	for _, inst := range m.WorkspaceInstances {
		if inst.ID != instanceID {
			filtered = append(filtered, inst)
		}
	}
	m.WorkspaceInstances = filtered
	return m, s.blockchain.StoreModelRecord(m)
}

// UpdateInstancePosition met à jour la position persistée d'une instance d'un module.
func (s *Service) UpdateInstancePosition(moduleID, instanceID string, x, y float64) (*Model3D, error) {
	if s.blockchain == nil {
		return nil, ErrBlockchainUnavailable
	}
	m, err := s.blockchain.GetModelRecord(moduleID, "")
	if err != nil {
		return nil, err
	}
	for i, inst := range m.WorkspaceInstances {
		if inst.ID == instanceID {
			m.WorkspaceInstances[i].X = x
			m.WorkspaceInstances[i].Y = y
			return m, s.blockchain.StoreModelRecord(m)
		}
	}
	return nil, fmt.Errorf("instance %s introuvable dans le module %s", instanceID, moduleID)
}

func (s *Service) RemoveModule(id string) error {
	if s.blockchain == nil {
		return ErrBlockchainUnavailable
	}
	type remover interface{ RemoveModelRecord(id string) error }
	if r, ok := s.blockchain.(remover); ok {
		return r.RemoveModelRecord(id)
	}
	return fmt.Errorf("suppression non supportée par cet adapter blockchain")
}

// GetModuleInterfaces retourne les interfaces d'un composant ou d'un module :
// - Composant simple : toutes ses interfaces directes.
// - Module : interfaces exposées = interfaces des sous-composants/modules non reliées en interne.
// La fonction est récursive ; le cache evite les appels blockchain redondants.
func (s *Service) GetModuleInterfaces(moduleID string) ([]*AssetInterface, error) {
	if s.ifaceStore == nil {
		return nil, fmt.Errorf("interface store non configuré")
	}
	cache := make(map[string][]*AssetInterface)
	return s.getModuleInterfacesInto(moduleID, cache)
}

// #incoherence — cache[moduleID] n'est écrit qu'à la fin du calcul, pas avant
// de récurser dans les WorkspaceInstances : un cycle encore en cours de calcul
// n'est jamais détecté — voir specs/roadmap_dev.md § Écarts — revue de code, E1.
func (s *Service) getModuleInterfacesInto(moduleID string, cache map[string][]*AssetInterface) ([]*AssetInterface, error) {
	if ifaces, ok := cache[moduleID]; ok {
		return ifaces, nil
	}
	var mod *Model3D
	var err error
	if s.blockchain != nil {
		mod, err = s.blockchain.GetModelRecord(moduleID, "")
	} else {
		err = ErrBlockchainUnavailable
	}
	if err != nil {
		// L'enregistrement est absent de la blockchain (asset non encore soumis, Fabric
		// indisponible, ou module local). On suppose un composant simple : on retourne
		// les interfaces locales directes, et on crée un slot virtuel seulement si
		// l'asset n'a encore aucune interface définie.
		// Cela permet d'afficher les points d'interface même sans connexion Fabric.
		ifaces, err2 := s.ifaceStore.ListInterfacesForAsset(moduleID)
		if err2 != nil {
			return nil, err // retourner l'erreur blockchain originale
		}
		if len(ifaces) == 0 {
			s.ensureVirtualSlot(moduleID)
			ifaces, _ = s.ifaceStore.ListInterfacesForAsset(moduleID)
		}
		cache[moduleID] = ifaces
		return ifaces, nil
	}
	// Composant simple : retourner ses interfaces directes.
	// Un slot virtuel est créé uniquement si l'asset n'a encore aucune interface
	// (permet de commencer à créer des liaisons).
	if !mod.IsModule() {
		ifaces, err := s.ifaceStore.ListInterfacesForAsset(moduleID)
		if err != nil {
			return nil, err
		}
		if len(ifaces) == 0 {
			s.ensureVirtualSlot(moduleID)
			ifaces, _ = s.ifaceStore.ListInterfacesForAsset(moduleID)
		}
		cache[moduleID] = ifaces
		return ifaces, err
	}
	// Module : garantir un slot virtuel propre au module (pour les connexions externes)
	// uniquement si le module lui-même n'a pas encore d'interfaces directes.
	ownIfacesCheck, _ := s.ifaceStore.ListInterfacesForAsset(moduleID)
	if len(ownIfacesCheck) == 0 {
		s.ensureVirtualSlot(moduleID)
	}
	// Calculer les interfaces exposées (non connectées en interne)
	internalIfaceIDs := map[string]bool{}
	if s.connStore != nil {
		asmSet := map[string]bool{}
		for _, id := range mod.Assemblies {
			asmSet[id] = true
		}
		conns, _ := s.connStore.ListConnections()
		for _, c := range conns {
			if asmSet[c.ID] {
				if c.FromIfaceID != "" {
					internalIfaceIDs[c.FromIfaceID] = true
				}
				if c.ToIfaceID != "" {
					internalIfaceIDs[c.ToIfaceID] = true
				}
			}
		}
	}
	seen := map[string]bool{}
	var exposed []*AssetInterface
	for _, inst := range mod.WorkspaceInstances {
		if seen[inst.AssetID] {
			continue
		}
		seen[inst.AssetID] = true
		ifaces, err := s.getModuleInterfacesInto(inst.AssetID, cache)
		if err != nil {
			continue
		}
		for _, iface := range ifaces {
			if !internalIfaceIDs[iface.ID] {
				exposed = append(exposed, iface)
			}
		}
	}
	// Inclure les interfaces directes du module lui-même (son slot virtuel propre)
	ownIfaces, _ := s.ifaceStore.ListInterfacesForAsset(moduleID)
	for _, iface := range ownIfaces {
		if !internalIfaceIDs[iface.ID] {
			exposed = append(exposed, iface)
		}
	}
	cache[moduleID] = exposed
	return exposed, nil
}

func computeModuleHash(productID string, vNum int, assemblies []string) string {
	sorted := append([]string{}, assemblies...)
	sort.Strings(sorted)
	payload := fmt.Sprintf("%s|v%d|%s", productID, vNum, strings.Join(sorted, ","))
	h := sha256.Sum256([]byte(payload))
	return fmt.Sprintf("sha256:%x", h[:8])
}

func simModuleBlockID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("P%08X", b)
}

// ── Licences ──────────────────────────────────────────────────────────────────

func (s *Service) ListLicenses() []*License {
	return ListLicenses()
}

func (s *Service) GetLicense(id string) (*License, error) {
	return GetLicense(id)
}

// CheckLicenseCompatibility expose la règle domaine via le service.
func (s *Service) CheckLicenseCompatibility(parentLicenseID, proposedLicenseID string) *LicenseCheck {
	return CheckLicenseCompatibility(parentLicenseID, proposedLicenseID)
}

// CheckModuleLicenseCompatibility vérifie la compatibilité d'une licence de produit
// vis-à-vis des licences de tous ses composants.
func (s *Service) CheckModuleLicenseCompatibility(componentLicenseIDs []string, proposedProductLicenseID string) *LicenseCheck {
	return CheckModuleLicenseCompatibility(componentLicenseIDs, proposedProductLicenseID)
}

// ── Liaison interface virtuelle → interface physique ─────────────────────────

// ConnectVirtualToPhysical matérialise une interface virtuelle en lui appliquant
// la catégorie, le type et la direction opposée de l'interface physique cible,
// puis crée la liaison d'assemblage correspondante.
// popupValues porte les champs saisis par le client (CLI/API)
// (Name, ValueMin, ValueMax, IsRange, Unit) ; les champs à zéro héritent des valeurs physiques.
func (s *Service) ConnectVirtualToPhysical(
	virtualIfaceID, physicalIfaceID string,
	popupValues AssetInterface,
	fromInstanceID, toInstanceID string,
) (*Connection, error) {
	if s.ifaceStore == nil {
		return nil, fmt.Errorf("interface store non configuré")
	}
	if s.connStore == nil {
		return nil, fmt.Errorf("connection store non configuré")
	}

	physical, err := s.ifaceStore.GetInterface(physicalIfaceID)
	if err != nil {
		return nil, fmt.Errorf("interface physique introuvable : %w", err)
	}
	virtual, err := s.ifaceStore.GetInterface(virtualIfaceID)
	if err != nil {
		return nil, fmt.Errorf("interface virtuelle introuvable : %w", err)
	}
	if !virtual.Virtual {
		return nil, fmt.Errorf("l'interface %s n'est pas virtuelle", virtualIfaceID)
	}

	oppositeDir := func(d IfaceDirection) IfaceDirection {
		switch d {
		case IfaceOut:
			return IfaceIn
		case IfaceIn:
			return IfaceOut
		default:
			return IfaceBidi
		}
	}

	// Matérialisation : la virtuelle adopte catégorie+type du physique, direction opposée
	virtual.Virtual = false
	virtual.Category = physical.Category
	virtual.Type = physical.Type
	virtual.Direction = oppositeDir(physical.Direction)

	// Valeurs issues de la popup, avec fallback sur les valeurs du physique
	if popupValues.Name != "" {
		virtual.Name = popupValues.Name
	} else {
		virtual.Name = physical.Name
	}
	if popupValues.ValueMin != 0 || popupValues.ValueMax != 0 {
		virtual.ValueMin = popupValues.ValueMin
		virtual.ValueMax = popupValues.ValueMax
		virtual.IsRange = popupValues.IsRange
	} else {
		virtual.ValueMin = physical.ValueMin
		virtual.ValueMax = physical.ValueMax
		virtual.IsRange = physical.IsRange
	}
	if popupValues.Unit != "" {
		virtual.Unit = popupValues.Unit
	} else {
		virtual.Unit = physical.Unit
	}

	if err := s.ifaceStore.SaveInterface(virtual); err != nil {
		return nil, fmt.Errorf("sauvegarde interface matérialisée : %w", err)
	}

	// fromInstanceID porte l'instance du VIRTUEL, toInstanceID celle du PHYSIQUE.
	// AddAssemblyLink attend (from=physique, to=virtuel) → on inverse les instance IDs.
	conn, err := s.AddAssemblyLink(physicalIfaceID, virtualIfaceID, virtual.Name, toInstanceID, fromInstanceID, "")
	if err != nil {
		return nil, fmt.Errorf("création liaison : %w", err)
	}
	// Maintenir l'invariant : recréer un slot virtuel pour que l'asset reste connectable
	s.ensureVirtualSlot(virtual.AssetID)
	return conn, nil
}

// ── interne ──────────────────────────────────────────────────────────────────

func (s *Service) hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	io.Copy(h, f)
	return fmt.Sprintf("sha256:%x", h.Sum(nil)), nil
}
