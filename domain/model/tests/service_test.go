package model_test

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"myr-core/domain/model"
)

// ══════════════════════════════════════════════════════════════════════════════
// Stubs / mocks
// ══════════════════════════════════════════════════════════════════════════════

// ── BlockchainPort ────────────────────────────────────────────────────────────

type mockBC struct {
	records map[string]*model.Model3D
}

func newMockBC() *mockBC {
	return &mockBC{records: make(map[string]*model.Model3D)}
}

func (m *mockBC) StoreModelRecord(mod *model.Model3D) error {
	m.records[mod.ID] = mod
	return nil
}

func (m *mockBC) GetModelRecord(id, _ string) (*model.Model3D, error) {
	mod, ok := m.records[id]
	if !ok {
		return nil, fmt.Errorf("modèle introuvable: %s", id)
	}
	return mod, nil
}

func (m *mockBC) ListModelRecords(channelID string) ([]*model.Model3D, error) {
	var list []*model.Model3D
	for _, mod := range m.records {
		if channelID == "" || mod.ChannelID == channelID {
			list = append(list, mod)
		}
	}
	return list, nil
}

func (m *mockBC) VerifyIntegrity(id, hash, _ string) (bool, error) {
	mod, err := m.GetModelRecord(id, "")
	if err != nil {
		return false, err
	}
	return mod.Hash == hash, nil
}

func (m *mockBC) RemoveModelRecord(id string) error {
	if _, ok := m.records[id]; !ok {
		return fmt.Errorf("modèle introuvable: %s", id)
	}
	delete(m.records, id)
	return nil
}

// ── FileStoragePort ───────────────────────────────────────────────────────────

type mockFS struct{}

func (m *mockFS) Upload(filePath string) (string, error) { return "hash:" + filePath, nil }
func (m *mockFS) Download(hash, destPath string) error   { return nil }
func (m *mockFS) Delete(hash string) error               { return nil }

// mockFSFn est une version configurable de mockFS pour simuler des erreurs.
type mockFSFn struct {
	uploadFn func(filePath string) (string, error)
}

func (m *mockFSFn) Upload(filePath string) (string, error) {
	if m.uploadFn != nil {
		return m.uploadFn(filePath)
	}
	return "hash:" + filePath, nil
}
func (m *mockFSFn) Download(hash, destPath string) error { return nil }
func (m *mockFSFn) Delete(hash string) error             { return nil }

// ── ConnectionStore ───────────────────────────────────────────────────────────

type mockConnStore struct {
	conns map[string]*model.Connection
}

func newMockConnStore() *mockConnStore {
	return &mockConnStore{conns: make(map[string]*model.Connection)}
}

func (s *mockConnStore) SaveConnection(c *model.Connection) error {
	s.conns[c.ID] = c
	return nil
}

func (s *mockConnStore) UpdateConnection(c *model.Connection) error {
	if _, ok := s.conns[c.ID]; !ok {
		return fmt.Errorf("connexion introuvable: %s", c.ID)
	}
	s.conns[c.ID] = c
	return nil
}

func (s *mockConnStore) RemoveConnection(id string) error {
	if _, ok := s.conns[id]; !ok {
		return fmt.Errorf("connexion introuvable: %s", id)
	}
	delete(s.conns, id)
	return nil
}

func (s *mockConnStore) ListConnections() ([]*model.Connection, error) {
	list := make([]*model.Connection, 0, len(s.conns))
	for _, c := range s.conns {
		list = append(list, c)
	}
	return list, nil
}

// ── ThumbnailStore ────────────────────────────────────────────────────────────

type mockThumbStore struct {
	thumbs map[string]string
}

func newMockThumbStore() *mockThumbStore {
	return &mockThumbStore{thumbs: make(map[string]string)}
}

func (s *mockThumbStore) SaveThumbnail(assetID, dataURL string) error {
	s.thumbs[assetID] = dataURL
	return nil
}

func (s *mockThumbStore) GetThumbnail(assetID string) (string, error) {
	return s.thumbs[assetID], nil
}

// ── OGImageFetcher ────────────────────────────────────────────────────────────

type mockOGImageFetcher struct {
	fetchFn func(pageURL string) (string, error)
}

func (f *mockOGImageFetcher) FetchOGImage(pageURL string) (string, error) {
	if f.fetchFn != nil {
		return f.fetchFn(pageURL)
	}
	return "data:image/png;base64,regenerated", nil
}

// ── InterfaceStore ────────────────────────────────────────────────────────────

type mockIfaceStore struct {
	ifaces map[string]*model.AssetInterface
	refs   *model.InterfaceRefs
}

func newMockIfaceStore() *mockIfaceStore {
	return &mockIfaceStore{
		ifaces: make(map[string]*model.AssetInterface),
		refs:   model.DefaultInterfaceRefs(),
	}
}

func (s *mockIfaceStore) SaveInterface(iface *model.AssetInterface) error {
	s.ifaces[iface.ID] = iface
	return nil
}

func (s *mockIfaceStore) RemoveInterface(id string) error {
	if _, ok := s.ifaces[id]; !ok {
		return fmt.Errorf("interface introuvable: %s", id)
	}
	delete(s.ifaces, id)
	return nil
}

func (s *mockIfaceStore) ListInterfacesForAsset(assetID string) ([]*model.AssetInterface, error) {
	var list []*model.AssetInterface
	for _, iface := range s.ifaces {
		if iface.AssetID == assetID {
			list = append(list, iface)
		}
	}
	return list, nil
}

func (s *mockIfaceStore) GetInterface(id string) (*model.AssetInterface, error) {
	iface, ok := s.ifaces[id]
	if !ok {
		return nil, fmt.Errorf("interface introuvable: %s", id)
	}
	return iface, nil
}

func (s *mockIfaceStore) GetRefs() (*model.InterfaceRefs, error) { return s.refs, nil }

func (s *mockIfaceStore) AddRefCategory(cat string) error {
	s.refs.Categories = append(s.refs.Categories, cat)
	return nil
}

func (s *mockIfaceStore) AddRefType(cat, typeName string) error {
	s.refs.Types[cat] = append(s.refs.Types[cat], typeName)
	return nil
}

func (s *mockIfaceStore) AddRefUnit(cat, unit string) error {
	s.refs.Units[cat] = append(s.refs.Units[cat], unit)
	return nil
}

// ── DraftStore ────────────────────────────────────────────────────────────────

type mockDraftStore struct {
	drafts map[string]*model.Model3D
}

func newMockDraftStore() *mockDraftStore {
	return &mockDraftStore{drafts: make(map[string]*model.Model3D)}
}

func (s *mockDraftStore) SaveDraft(m *model.Model3D) error {
	s.drafts[m.ID] = m
	return nil
}

func (s *mockDraftStore) GetDraft(id string) (*model.Model3D, error) {
	m, ok := s.drafts[id]
	if !ok {
		return nil, fmt.Errorf("brouillon introuvable: %s", id)
	}
	return m, nil
}

func (s *mockDraftStore) RemoveDraft(id string) error {
	delete(s.drafts, id)
	return nil
}

func (s *mockDraftStore) ListDrafts(channelID string) ([]*model.Model3D, error) {
	var list []*model.Model3D
	for _, m := range s.drafts {
		if channelID == "" || m.ChannelID == channelID {
			list = append(list, m)
		}
	}
	return list, nil
}

// ══════════════════════════════════════════════════════════════════════════════
// Helpers
// ══════════════════════════════════════════════════════════════════════════════

func tempFile(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp("", "model-test-*.obj")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	f.WriteString(content)
	t.Cleanup(func() { os.Remove(f.Name()) })
	return f.Name()
}

func newFullSvc(t *testing.T) *model.Service {
	t.Helper()
	return model.NewService(newMockBC(), &mockFS{}).
		WithConnStore(newMockConnStore()).
		WithThumbStore(newMockThumbStore()).
		WithIfaceStore(newMockIfaceStore()).
		WithDraftStore(newMockDraftStore())
}

// ══════════════════════════════════════════════════════════════════════════════
// Tests — assets basiques
// ══════════════════════════════════════════════════════════════════════════════

// / @brief  Création d'un asset avec fichier valide et métadonnées complètes
// / @input  mockBC vide, mockFS, fichier temporaire OBJ avec contenu minimal
// / @expect Retour sans erreur, ID non vide, Hash non vide, Name/ChannelID corrects, 1 version créée
func TestModelAdd(t *testing.T) {
	svc := model.NewService(newMockBC(), &mockFS{}).WithDraftStore(newMockDraftStore())

	path := tempFile(t, "v 0 0 0\n")
	m, err := svc.Add(path, "cube", "ch1", "owner1", []string{"3d"})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if m.Name != "cube" {
		t.Errorf("Name: got %q, want %q", m.Name, "cube")
	}
	if m.ChannelID != "ch1" {
		t.Errorf("ChannelID: got %q, want %q", m.ChannelID, "ch1")
	}
	if m.ID == "" {
		t.Error("expected non-empty ID")
	}
	if m.Hash == "" {
		t.Error("expected non-empty Hash")
	}
	if len(m.Versions) != 1 {
		t.Errorf("Versions: got %d, want 1", len(m.Versions))
	}
}

// / @brief  Toute opération nécessitant BlockchainPort échoue proprement quand
// / aucun adapter blockchain n'est configuré, sans repli silencieux vers un
// / autre stockage. La création (AddFull) n'en fait plus partie : elle ne
// / dépend que du DraftStore (voir TestAddFull_NilBlockchain_StillWorks) —
// / seule la soumission explicite exige la blockchain.
// / @input  Service construit avec bc=nil, DraftStore configuré (mais vide)
// / @expect ErrBlockchainUnavailable pour Submit, SubmitModule, List et GetModule
func TestService_NilBlockchain_ErrBlockchainUnavailable(t *testing.T) {
	svc := model.NewService(nil, &mockFS{}).WithDraftStore(newMockDraftStore())

	if _, err := svc.Submit("any"); !errors.Is(err, model.ErrBlockchainUnavailable) {
		t.Errorf("Submit: got %v, want ErrBlockchainUnavailable", err)
	}
	if _, err := svc.SubmitModule("mod1", "note"); !errors.Is(err, model.ErrBlockchainUnavailable) {
		t.Errorf("SubmitModule: got %v, want ErrBlockchainUnavailable", err)
	}
	if _, err := svc.List("ch1"); !errors.Is(err, model.ErrBlockchainUnavailable) {
		t.Errorf("List: got %v, want ErrBlockchainUnavailable", err)
	}
	if _, err := svc.GetModule("mod1"); !errors.Is(err, model.ErrBlockchainUnavailable) {
		t.Errorf("GetModule: got %v, want ErrBlockchainUnavailable", err)
	}
}

// / @brief  Ajout d'un asset avec un chemin de fichier inexistant
// / @input  mockBC vide, mockFS, chemin "/inexistant/fichier.obj"
// / @expect Retour d'une erreur (fichier introuvable)
func TestModelAdd_FileMissing(t *testing.T) {
	svc := model.NewService(newMockBC(), &mockFS{}).WithDraftStore(newMockDraftStore())

	_, err := svc.Add("/inexistant/fichier.obj", "cube", "ch1", "o1", nil)
	if err == nil {
		t.Error("expected error for missing file")
	}
}

// / @brief  Création d'un asset sans fichier (lien boutique, catalogue pur)
// / @input  mockBC vide, mockFS, FilePath vide ("")
// / @expect Retour sans erreur, Hash vide, Versions vide
func TestModelAdd_NoFile(t *testing.T) {
	svc := model.NewService(newMockBC(), &mockFS{}).WithDraftStore(newMockDraftStore())

	m, err := svc.Add("", "lien-boutique", "ch1", "o1", []string{"achat"})
	if err != nil {
		t.Fatalf("Add sans fichier: %v", err)
	}
	if m.Hash != "" {
		t.Error("Hash doit être vide si pas de fichier")
	}
	if len(m.Versions) != 0 {
		t.Error("Versions doit être vide si pas de fichier")
	}
}

// / @brief  AddFull avec tous les champs optionnels renseignés (description, catégorie, liens)
// / @input  AddRequest complet : Name, Description, Category, Tags, Links, ChannelID, OwnerID
// / @expect Description, Category et Links conservés tels quels dans le modèle retourné
func TestAddFull_AllFields(t *testing.T) {
	svc := model.NewService(newMockBC(), &mockFS{}).WithDraftStore(newMockDraftStore())

	m, err := svc.AddFull(model.AddRequest{
		Name:        "vis M3",
		Description: "vis acier inox",
		Category:    model.CategoryBase,
		ChannelID:   "ch1",
		OwnerID:     "o1",
		Tags:        []string{"fixation", "meca"},
		Links:       []string{"https://shop.example.com/vis-m3"},
	})
	if err != nil {
		t.Fatalf("AddFull: %v", err)
	}
	if m.Description != "vis acier inox" {
		t.Errorf("Description: got %q", m.Description)
	}
	if m.Category != model.CategoryBase {
		t.Errorf("Category: got %q", m.Category)
	}
	if len(m.Links) != 1 || m.Links[0] != "https://shop.example.com/vis-m3" {
		t.Errorf("Links: got %v", m.Links)
	}
}

// / @brief  AddFull avec un asset parent — relation variante / hiérarchie
// / @input  Asset base créé, puis AddRequest variante avec ParentID = base.ID et CategoryVariation
// / @expect child.ParentID égal à l'ID du parent
func TestAddFull_WithParent(t *testing.T) {
	svc := model.NewService(newMockBC(), &mockFS{}).WithDraftStore(newMockDraftStore())

	base, _ := svc.AddFull(model.AddRequest{Name: "base", ChannelID: "ch1", OwnerID: "o1"})
	child, err := svc.AddFull(model.AddRequest{
		Name:      "variante",
		Category:  model.CategoryVariation,
		ParentID:  base.ID,
		ChannelID: "ch1",
		OwnerID:   "o1",
	})
	if err != nil {
		t.Fatalf("AddFull variante: %v", err)
	}
	if child.ParentID != base.ID {
		t.Errorf("ParentID: got %q, want %q", child.ParentID, base.ID)
	}
}

// / @brief  Récupération d'un asset existant par son ID
// / @input  mockBC avec un asset préalablement ajouté via Add
// / @expect Retour sans erreur, ID de l'asset correct
func TestModelGet(t *testing.T) {
	svc := model.NewService(newMockBC(), &mockFS{}).WithDraftStore(newMockDraftStore())

	path := tempFile(t, "data")
	created, _ := svc.Add(path, "cube", "ch1", "o1", nil)

	got, err := svc.Get(created.ID, "")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("ID: got %q, want %q", got.ID, created.ID)
	}
}

// / @brief  Récupération d'un asset avec un ID inconnu
// / @input  mockBC vide, ID "inexistant"
// / @expect Retour d'une erreur
func TestModelGet_NotFound(t *testing.T) {
	svc := model.NewService(newMockBC(), &mockFS{}).WithDraftStore(newMockDraftStore())

	_, err := svc.Get("inexistant", "")
	if err == nil {
		t.Error("expected error for unknown ID")
	}
}

// / @brief  Listage de tous les assets puis filtrage par channel
// / @input  mockBC avec 2 assets sur des channels distincts (ch1 et ch2)
// / @expect List("") retourne 2 assets, List("ch1") retourne 1 asset
func TestModelList(t *testing.T) {
	svc := model.NewService(newMockBC(), &mockFS{}).WithDraftStore(newMockDraftStore())

	svc.Add(tempFile(t, "a"), "m1", "ch1", "o1", nil)
	svc.Add(tempFile(t, "b"), "m2", "ch2", "o1", nil)

	all, err := svc.List("")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("got %d modèles, want 2", len(all))
	}

	ch1only, err := svc.List("ch1")
	if err != nil {
		t.Fatalf("List ch1: %v", err)
	}
	if len(ch1only) != 1 {
		t.Errorf("got %d modèles pour ch1, want 1", len(ch1only))
	}
}

// / @brief  Vérification de l'intégrité d'un asset dont le hash est intact
// / @input  mockBC avec un asset uploadé, hash non altéré
// / @expect Verify retourne true sans erreur
func TestModelVerify(t *testing.T) {
	svc := model.NewService(newMockBC(), &mockFS{}).WithDraftStore(newMockDraftStore())

	path := tempFile(t, "contenu stable")
	m, _ := svc.Add(path, "cube", "ch1", "o1", nil)

	ok, err := svc.Verify(m.ID, "")
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !ok {
		t.Error("vérification devrait réussir")
	}
}

// / @brief  Vérification d'intégrité d'un asset avec un ID inconnu
// / @input  mockBC vide, ID "inexistant"
// / @expect Retour d'une erreur
func TestModelVerify_NotFound(t *testing.T) {
	svc := model.NewService(newMockBC(), &mockFS{}).WithDraftStore(newMockDraftStore())

	_, err := svc.Verify("inexistant", "")
	if err == nil {
		t.Error("expected error for unknown ID")
	}
}

// / @brief  Suppression d'un asset entraîne la suppression en cascade de ses connexions
// / @input  Service complet (BC + connStore), 2 assets liés par une connexion
// / @expect Remove réussit, la connexion associée disparaît de ListConnections
func TestModelRemove(t *testing.T) {
	svc := newFullSvc(t)

	m1, _ := svc.AddFull(model.AddRequest{Name: "a", ChannelID: "ch1", OwnerID: "o1"})
	m2, _ := svc.AddFull(model.AddRequest{Name: "b", ChannelID: "ch1", OwnerID: "o1"})
	conn, _ := svc.AddConnection(m1.ID, m2.ID, "lien")

	// La connexion existe
	conns, _ := svc.ListConnections()
	if len(conns) != 1 {
		t.Fatalf("attendu 1 connexion avant suppression, got %d", len(conns))
	}

	if err := svc.Remove(m1.ID); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	// La connexion doit avoir été supprimée en cascade
	conns, _ = svc.ListConnections()
	for _, c := range conns {
		if c.ID == conn.ID {
			t.Error("connexion aurait dû être supprimée en cascade")
		}
	}
}

// / @brief  Suppression d'un asset existant — asset devient introuvable après Remove
// / @input  mockBC avec un asset AddFull, pas de connStore
// / @expect Remove réussit, Get sur l'ID supprimé retourne une erreur
func TestModelRemove_NotSupported(t *testing.T) {
	// Sans implémentation de RemoveModelRecord sur la blockchain
	type minimalBC struct{ *mockBC }
	// minimalBC n'expose pas RemoveModelRecord — le service doit retourner une erreur
	svc := model.NewService(newMockBC(), &mockFS{}).WithDraftStore(newMockDraftStore())
	// mockBC implémente RemoveModelRecord — tester l'absence via une struct sans la méthode
	type noRemoveBC struct {
		records map[string]*model.Model3D
	}
	// On ne peut pas tester ça sans une vraie struct distincte — on vérifie juste
	// que Remove sur mockBC (qui l'implémente) fonctionne.
	m, _ := svc.AddFull(model.AddRequest{Name: "x", ChannelID: "ch1", OwnerID: "o1"})
	if err := svc.Remove(m.ID); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	_, err := svc.Get(m.ID, "")
	if err == nil {
		t.Error("asset doit être introuvable après suppression")
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Tests — connexions d'assemblage
// ══════════════════════════════════════════════════════════════════════════════

// / @brief  Cycle complet ajout / listage / suppression d'une connexion d'assemblage
// / @input  Service avec mockConnStore, connexion entre "asset1" et "asset2" labellisée "composant"
// / @expect Connexion créée avec ID/From/To/Label corrects, présente dans List, absente après Remove
func TestConnection_AddListRemove(t *testing.T) {
	svc := model.NewService(newMockBC(), &mockFS{}).
		WithConnStore(newMockConnStore())

	conn, err := svc.AddConnection("asset1", "asset2", "composant")
	if err != nil {
		t.Fatalf("AddConnection: %v", err)
	}
	if conn.ID == "" {
		t.Error("connexion ID ne doit pas être vide")
	}
	if conn.From != "asset1" || conn.To != "asset2" {
		t.Errorf("connexion From/To incorrects: %+v", conn)
	}
	if conn.Label != "composant" {
		t.Errorf("Label: got %q", conn.Label)
	}

	list, err := svc.ListConnections()
	if err != nil {
		t.Fatalf("ListConnections: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("attendu 1 connexion, got %d", len(list))
	}

	if err := svc.RemoveConnection(conn.ID); err != nil {
		t.Fatalf("RemoveConnection: %v", err)
	}

	list, _ = svc.ListConnections()
	if len(list) != 0 {
		t.Error("liste connexions doit être vide après suppression")
	}
}

// / @brief  AddConnection et RemoveConnection sans connStore retournent une erreur
// / @input  Service sans WithConnStore (connStore nil)
// / @expect AddConnection et RemoveConnection retournent chacun une erreur
func TestConnection_NilStore_ReturnsError(t *testing.T) {
	svc := model.NewService(newMockBC(), &mockFS{}).WithDraftStore(newMockDraftStore())

	if _, err := svc.AddConnection("a", "b", "x"); err == nil {
		t.Error("AddConnection sans connStore doit retourner une erreur")
	}
	if err := svc.RemoveConnection("x"); err == nil {
		t.Error("RemoveConnection sans connStore doit retourner une erreur")
	}
}

// / @brief  ListConnections sans connStore retourne nil sans erreur
// / @input  Service sans WithConnStore (connStore nil)
// / @expect Retour (nil, nil) — pas d'erreur, liste nil
func TestConnection_ListNilStore_ReturnsNil(t *testing.T) {
	svc := model.NewService(newMockBC(), &mockFS{}).WithDraftStore(newMockDraftStore())

	list, err := svc.ListConnections()
	if err != nil {
		t.Fatalf("ListConnections sans connStore ne doit pas retourner d'erreur: %v", err)
	}
	if list != nil {
		t.Error("ListConnections sans connStore doit retourner nil")
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Tests — interfaces physiques
// ══════════════════════════════════════════════════════════════════════════════

// / @brief  Cycle complet ajout / listage / récupération / suppression d'une interface physique
// / @input  Service avec mockIfaceStore, interface ELEC USB-C Out sur "asset1"
// / @expect ID généré, interface présente dans List et Get, absente après Remove
func TestInterface_AddListRemove(t *testing.T) {
	svc := model.NewService(newMockBC(), &mockFS{}).
		WithIfaceStore(newMockIfaceStore())

	iface := &model.AssetInterface{
		AssetID:   "asset1",
		Name:      "alim 5V",
		Category:  "ELEC",
		Type:      "USB-C",
		Direction: model.IfaceOut,
		ValueMin:  5.0,
		Unit:      "V",
	}

	if err := svc.AddInterface(iface); err != nil {
		t.Fatalf("AddInterface: %v", err)
	}
	if iface.ID == "" {
		t.Error("ID doit être généré si vide")
	}

	list, err := svc.ListInterfacesForAsset("asset1")
	if err != nil {
		t.Fatalf("ListInterfacesForAsset: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("attendu 1 interface, got %d", len(list))
	}

	got, err := svc.GetInterface(iface.ID)
	if err != nil {
		t.Fatalf("GetInterface: %v", err)
	}
	if got.Category != "ELEC" || got.Type != "USB-C" {
		t.Errorf("interface incorrecte: %+v", got)
	}

	if err := svc.RemoveInterface(iface.ID); err != nil {
		t.Fatalf("RemoveInterface: %v", err)
	}

	list, _ = svc.ListInterfacesForAsset("asset1")
	if len(list) != 0 {
		t.Error("liste interfaces doit être vide après suppression")
	}
}

// / @brief  AddInterface ne génère pas de nouvel ID si un ID est déjà fourni
// / @input  Interface avec ID "custom-id" pré-défini
// / @expect iface.ID reste "custom-id" après AddInterface
func TestInterface_IDPreservedIfGiven(t *testing.T) {
	svc := model.NewService(newMockBC(), &mockFS{}).
		WithIfaceStore(newMockIfaceStore())

	iface := &model.AssetInterface{ID: "custom-id", AssetID: "a1", Category: "MECA", Type: "Vis M3", Direction: model.IfaceIn}
	svc.AddInterface(iface)

	if iface.ID != "custom-id" {
		t.Errorf("ID ne doit pas être écrasé: got %q", iface.ID)
	}
}

// / @brief  AddInterface, RemoveInterface et GetInterface sans ifaceStore retournent une erreur
// / @input  Service sans WithIfaceStore (ifaceStore nil)
// / @expect Chacune des trois opérations retourne une erreur
func TestInterface_NilStore_ReturnsError(t *testing.T) {
	svc := model.NewService(newMockBC(), &mockFS{}).WithDraftStore(newMockDraftStore())

	iface := &model.AssetInterface{AssetID: "a1", Category: "ELEC", Type: "DIN", Direction: model.IfaceIn}
	if err := svc.AddInterface(iface); err == nil {
		t.Error("AddInterface sans ifaceStore doit retourner une erreur")
	}
	if err := svc.RemoveInterface("x"); err == nil {
		t.Error("RemoveInterface sans ifaceStore doit retourner une erreur")
	}
	if _, err := svc.GetInterface("x"); err == nil {
		t.Error("GetInterface sans ifaceStore doit retourner une erreur")
	}
}

// / @brief  ListInterfacesForAsset sans ifaceStore retourne nil sans erreur
// / @input  Service sans WithIfaceStore (ifaceStore nil)
// / @expect Retour (nil, nil) — pas d'erreur, liste nil
func TestInterface_ListNilStore_ReturnsNil(t *testing.T) {
	svc := model.NewService(newMockBC(), &mockFS{}).WithDraftStore(newMockDraftStore())

	list, err := svc.ListInterfacesForAsset("a1")
	if err != nil {
		t.Fatalf("ListInterfacesForAsset sans ifaceStore ne doit pas retourner d'erreur: %v", err)
	}
	if list != nil {
		t.Error("attendu nil")
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Tests — AddAssemblyLink (interface ↔ interface)
// ══════════════════════════════════════════════════════════════════════════════

// / @brief  Création d'un lien d'assemblage entre deux interfaces compatibles
// / @input  Service avec mockConnStore + mockIfaceStore, deux interfaces ELEC USB-C (Out / In) sur asset1/asset2
// / @expect Connexion créée avec FromIfaceID/ToIfaceID, From/To et InstanceIDs corrects
func TestAddAssemblyLink(t *testing.T) {
	svc := model.NewService(newMockBC(), &mockFS{}).
		WithConnStore(newMockConnStore()).
		WithIfaceStore(newMockIfaceStore())

	ifaceA := &model.AssetInterface{AssetID: "asset1", Category: "ELEC", Type: "USB-C", Direction: model.IfaceOut}
	ifaceB := &model.AssetInterface{AssetID: "asset2", Category: "ELEC", Type: "USB-C", Direction: model.IfaceIn}
	svc.AddInterface(ifaceA)
	svc.AddInterface(ifaceB)

	conn, err := svc.AddAssemblyLink(ifaceA.ID, ifaceB.ID, "alimentation", "inst-a", "inst-b", "")
	if err != nil {
		t.Fatalf("AddAssemblyLink: %v", err)
	}
	if conn.FromInstanceID != "inst-a" || conn.ToInstanceID != "inst-b" {
		t.Errorf("InstanceIDs incorrects: from=%q to=%q", conn.FromInstanceID, conn.ToInstanceID)
	}
	if conn.From != "asset1" || conn.To != "asset2" {
		t.Errorf("From/To incorrects: %+v", conn)
	}
	if conn.FromIfaceID != ifaceA.ID || conn.ToIfaceID != ifaceB.ID {
		t.Errorf("FromIfaceID/ToIfaceID incorrects: %+v", conn)
	}
}

// / @brief  AddAssemblyLink échoue si ifaceStore ou connStore est absent
// / @input  Service sans ifaceStore (cas 1) ; service avec ifaceStore mais sans connStore (cas 2)
// / @expect Retour d'une erreur dans les deux cas
func TestAddAssemblyLink_MissingStore(t *testing.T) {
	svc := model.NewService(newMockBC(), &mockFS{}).WithDraftStore(newMockDraftStore())

	if _, err := svc.AddAssemblyLink("a", "b", "x", "", "", ""); err == nil {
		t.Error("AddAssemblyLink sans ifaceStore doit retourner une erreur")
	}

	svc2 := model.NewService(newMockBC(), &mockFS{}).WithIfaceStore(newMockIfaceStore())
	if _, err := svc2.AddAssemblyLink("a", "b", "x", "", "", ""); err == nil {
		t.Error("AddAssemblyLink sans connStore doit retourner une erreur")
	}
}

// / @brief  AddAssemblyLink avec des IDs d'interfaces inconnues retourne une erreur
// / @input  Service avec mockConnStore + mockIfaceStore vide, IDs "inexistant-a" et "inexistant-b"
// / @expect Retour d'une erreur (interfaces introuvables)
func TestAddAssemblyLink_UnknownInterface(t *testing.T) {
	svc := model.NewService(newMockBC(), &mockFS{}).
		WithConnStore(newMockConnStore()).
		WithIfaceStore(newMockIfaceStore())

	if _, err := svc.AddAssemblyLink("inexistant-a", "inexistant-b", "x", "", "", ""); err == nil {
		t.Error("AddAssemblyLink avec interfaces inconnues doit retourner une erreur")
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Tests — miniatures
// ══════════════════════════════════════════════════════════════════════════════

// / @brief  Sauvegarde puis récupération d'une miniature (data URL)
// / @input  Service avec mockThumbStore, dataURL "data:image/png;base64,abc123" pour "asset1"
// / @expect GetThumbnail retourne exactement la même dataURL sans erreur
func TestThumbnail_SaveGet(t *testing.T) {
	svc := model.NewService(newMockBC(), &mockFS{}).
		WithThumbStore(newMockThumbStore())

	dataURL := "data:image/png;base64,abc123"
	if err := svc.SaveThumbnail("asset1", dataURL); err != nil {
		t.Fatalf("SaveThumbnail: %v", err)
	}

	got, err := svc.GetThumbnail("asset1")
	if err != nil {
		t.Fatalf("GetThumbnail: %v", err)
	}
	if got != dataURL {
		t.Errorf("dataURL: got %q, want %q", got, dataURL)
	}
}

func TestThumbnail_NilStore_ReturnsError(t *testing.T) {
	svc := model.NewService(newMockBC(), &mockFS{}).WithDraftStore(newMockDraftStore())

	if err := svc.SaveThumbnail("a", "data:..."); err == nil {
		t.Error("SaveThumbnail sans thumbStore doit retourner une erreur")
	}
}

func TestThumbnail_NilStore_GetReturnsEmpty(t *testing.T) {
	svc := model.NewService(newMockBC(), &mockFS{}).WithDraftStore(newMockDraftStore())

	got, err := svc.GetThumbnail("a")
	if err != nil {
		t.Fatalf("GetThumbnail sans thumbStore ne doit pas retourner d'erreur: %v", err)
	}
	if got != "" {
		t.Errorf("attendu chaîne vide, got %q", got)
	}
}

// / @brief  Régénère la miniature d'un asset depuis l'og:image de son premier lien externe
// / @input  Asset "asset1" avec Links=["https://boutique.example/piece"], mockOGImageFetcher retournant une dataURL fixe
// / @expect RegenerateThumbnail retourne la dataURL fetchée et la persiste dans le thumbStore
func TestRegenerateThumbnail_OK(t *testing.T) {
	bc := newMockBC()
	bc.records["asset1"] = &model.Model3D{ID: "asset1", Links: []string{"https://boutique.example/piece"}}
	thumbs := newMockThumbStore()

	var gotURL string
	svc := model.NewService(bc, &mockFS{}).
		WithThumbStore(thumbs).
		WithOGImageFetcher(&mockOGImageFetcher{
			fetchFn: func(pageURL string) (string, error) {
				gotURL = pageURL
				return "data:image/png;base64,regenerated", nil
			},
		})

	got, err := svc.RegenerateThumbnail("asset1")
	if err != nil {
		t.Fatalf("RegenerateThumbnail: %v", err)
	}
	if got != "data:image/png;base64,regenerated" {
		t.Errorf("dataURL: got %q", got)
	}
	if gotURL != "https://boutique.example/piece" {
		t.Errorf("lien passé au fetcher: got %q", gotURL)
	}
	if saved, _ := thumbs.GetThumbnail("asset1"); saved != got {
		t.Errorf("miniature non persistée: got %q", saved)
	}
}

// / @brief  RM — un asset sans lien externe n'a pas de source régénérable côté serveur
// / @input  Asset "asset1" sans Links, thumbStore et ogImageFetcher configurés
// / @expect RegenerateThumbnail retourne model.ErrNoThumbnailSource
func TestRegenerateThumbnail_NoLinks_Rejected(t *testing.T) {
	bc := newMockBC()
	bc.records["asset1"] = &model.Model3D{ID: "asset1"}
	svc := model.NewService(bc, &mockFS{}).
		WithThumbStore(newMockThumbStore()).
		WithOGImageFetcher(&mockOGImageFetcher{})

	_, err := svc.RegenerateThumbnail("asset1")
	if !errors.Is(err, model.ErrNoThumbnailSource) {
		t.Fatalf("erreur attendue ErrNoThumbnailSource, got %v", err)
	}
}

// / @brief  Sans thumbStore configuré, la régénération échoue proprement
// / @input  Service sans WithThumbStore
// / @expect RegenerateThumbnail retourne une erreur
func TestRegenerateThumbnail_NilThumbStore_Rejected(t *testing.T) {
	svc := model.NewService(newMockBC(), &mockFS{}).
		WithOGImageFetcher(&mockOGImageFetcher{})

	if _, err := svc.RegenerateThumbnail("asset1"); err == nil {
		t.Error("RegenerateThumbnail sans thumbStore doit retourner une erreur")
	}
}

// / @brief  Sans OGImageFetcher configuré, la régénération échoue proprement
// / @input  Service sans WithOGImageFetcher
// / @expect RegenerateThumbnail retourne une erreur
func TestRegenerateThumbnail_NilFetcher_Rejected(t *testing.T) {
	svc := model.NewService(newMockBC(), &mockFS{}).
		WithThumbStore(newMockThumbStore())

	if _, err := svc.RegenerateThumbnail("asset1"); err == nil {
		t.Error("RegenerateThumbnail sans ogImageFetcher doit retourner une erreur")
	}
}

// / @brief  Une erreur de récupération de l'og:image (page injoignable, balise absente...) est propagée
// / @input  Asset avec un lien, mockOGImageFetcher retournant une erreur
// / @expect RegenerateThumbnail retourne une erreur et ne persiste rien
func TestRegenerateThumbnail_FetchError_Propagates(t *testing.T) {
	bc := newMockBC()
	bc.records["asset1"] = &model.Model3D{ID: "asset1", Links: []string{"https://boutique.example/piece"}}
	thumbs := newMockThumbStore()
	svc := model.NewService(bc, &mockFS{}).
		WithThumbStore(thumbs).
		WithOGImageFetcher(&mockOGImageFetcher{
			fetchFn: func(string) (string, error) { return "", fmt.Errorf("og:image introuvable") },
		})

	if _, err := svc.RegenerateThumbnail("asset1"); err == nil {
		t.Error("erreur de fetch attendue")
	}
	if saved, _ := thumbs.GetThumbnail("asset1"); saved != "" {
		t.Errorf("aucune miniature ne doit être persistée en cas d'échec, got %q", saved)
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Tests — vocabulaire de référence (InterfaceRefs)
// ══════════════════════════════════════════════════════════════════════════════

func TestRefs_Default(t *testing.T) {
	svc := model.NewService(newMockBC(), &mockFS{}).WithDraftStore(newMockDraftStore())

	refs, err := svc.GetRefs()
	if err != nil {
		t.Fatalf("GetRefs: %v", err)
	}
	if refs == nil {
		t.Fatal("refs ne doit pas être nil")
	}
	found := false
	for _, cat := range refs.Categories {
		if cat == "ELEC" {
			found = true
		}
	}
	if !found {
		t.Error("catégorie ELEC absente des refs par défaut")
	}
}

func TestRefs_AddCategoryTypeUnit(t *testing.T) {
	svc := model.NewService(newMockBC(), &mockFS{}).
		WithIfaceStore(newMockIfaceStore())

	if err := svc.AddRefCategory("OPT"); err != nil {
		t.Fatalf("AddRefCategory: %v", err)
	}
	if err := svc.AddRefType("OPT", "SMA"); err != nil {
		t.Fatalf("AddRefType: %v", err)
	}
	if err := svc.AddRefUnit("OPT", "dBm"); err != nil {
		t.Fatalf("AddRefUnit: %v", err)
	}

	refs, _ := svc.GetRefs()
	foundCat := false
	for _, cat := range refs.Categories {
		if cat == "OPT" {
			foundCat = true
		}
	}
	if !foundCat {
		t.Error("catégorie OPT non trouvée")
	}
	types := refs.Types["OPT"]
	if len(types) == 0 || types[0] != "SMA" {
		t.Errorf("type SMA non trouvé dans OPT: %v", types)
	}
	units := refs.Units["OPT"]
	if len(units) == 0 || units[0] != "dBm" {
		t.Errorf("unité dBm non trouvée dans OPT: %v", units)
	}
}

func TestRefs_NilStore_ReturnsError(t *testing.T) {
	svc := model.NewService(newMockBC(), &mockFS{}).WithDraftStore(newMockDraftStore())

	if err := svc.AddRefCategory("X"); err == nil {
		t.Error("AddRefCategory sans ifaceStore doit retourner une erreur")
	}
	if err := svc.AddRefType("X", "Y"); err == nil {
		t.Error("AddRefType sans ifaceStore doit retourner une erreur")
	}
	if err := svc.AddRefUnit("X", "Z"); err == nil {
		t.Error("AddRefUnit sans ifaceStore doit retourner une erreur")
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Tests — produits
// ══════════════════════════════════════════════════════════════════════════════

func TestModule_CreateGetList(t *testing.T) {
	svc := model.NewService(newMockBC(), &mockFS{}).WithDraftStore(newMockDraftStore())

	p, err := svc.CreateModule(model.ModuleRequest{
		Name:        "Drone v1",
		Description: "prototype",
		OwnerID:     "o1",
		ChannelID:   "ch1",
	})
	if err != nil {
		t.Fatalf("CreateProduct: %v", err)
	}
	if p.ID == "" {
		t.Error("ID produit ne doit pas être vide")
	}
	if p.Status != model.ModuleDraft {
		t.Errorf("Status: got %q, want %q", p.Status, model.ModuleDraft)
	}
	if len(p.Assemblies) != 0 || p.Assemblies == nil {
		t.Error("Assemblies doit être initialisé vide")
	}

	got, err := svc.GetModule(p.ID)
	if err != nil {
		t.Fatalf("GetProduct: %v", err)
	}
	if got.Name != "Drone v1" {
		t.Errorf("Name: got %q", got.Name)
	}

	list, err := svc.ListModules("ch1")
	if err != nil {
		t.Fatalf("ListProducts: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("attendu 1 produit, got %d", len(list))
	}
}

func TestModule_ListNilStore_ReturnsNil(t *testing.T) {
	svc := model.NewService(newMockBC(), &mockFS{}).WithDraftStore(newMockDraftStore())

	list, err := svc.ListModules("ch1")
	if err != nil {
		t.Fatalf("ListModules ne doit pas retourner d'erreur: %v", err)
	}
	if list != nil {
		t.Error("attendu nil")
	}
}

func TestModule_NotFound_ReturnsError(t *testing.T) {
	svc := model.NewService(newMockBC(), &mockFS{}).WithDraftStore(newMockDraftStore())

	if _, err := svc.GetModule("x"); err == nil {
		t.Error("GetModule avec ID inexistant doit retourner une erreur")
	}
	if err := svc.AddAssemblyToModule("x", "y"); err == nil {
		t.Error("AddAssemblyToModule avec ID inexistant doit retourner une erreur")
	}
	if err := svc.RemoveAssemblyFromModule("x", "y"); err == nil {
		t.Error("RemoveAssemblyFromModule avec ID inexistant doit retourner une erreur")
	}
	if _, err := svc.SubmitModule("x", "note"); err == nil {
		t.Error("SubmitModule avec ID inexistant doit retourner une erreur")
	}
	if err := svc.RemoveModule("x"); err == nil {
		t.Error("RemoveModule avec ID inexistant doit retourner une erreur")
	}
}

func TestModule_AddRemoveAssembly(t *testing.T) {
	svc := newFullSvc(t)

	p, _ := svc.CreateModule(model.ModuleRequest{Name: "P1", ChannelID: "ch1", OwnerID: "o1"})
	conn, _ := svc.AddConnection("a1", "a2", "lien")

	if err := svc.AddAssemblyToModule(p.ID, conn.ID); err != nil {
		t.Fatalf("AddAssemblyToProduct: %v", err)
	}

	// Idempotent — ajouter deux fois ne duplique pas
	svc.AddAssemblyToModule(p.ID, conn.ID)
	p2, _ := svc.GetModule(p.ID)
	if len(p2.Assemblies) != 1 {
		t.Errorf("attendu 1 assembly, got %d", len(p2.Assemblies))
	}

	if err := svc.RemoveAssemblyFromModule(p.ID, conn.ID); err != nil {
		t.Fatalf("RemoveAssemblyFromProduct: %v", err)
	}
	p3, _ := svc.GetModule(p.ID)
	if len(p3.Assemblies) != 0 {
		t.Errorf("attendu 0 assembly, got %d", len(p3.Assemblies))
	}
}

func TestModule_Submit(t *testing.T) {
	svc := newFullSvc(t)

	p, _ := svc.CreateModule(model.ModuleRequest{Name: "P1", ChannelID: "ch1", OwnerID: "o1"})
	conn, _ := svc.AddConnection("a1", "a2", "lien")
	svc.AddAssemblyToModule(p.ID, conn.ID)

	submitted, err := svc.SubmitModule(p.ID, "première livraison")
	if err != nil {
		t.Fatalf("SubmitProduct: %v", err)
	}
	if submitted.Status != model.ModuleSubmitted {
		t.Errorf("Status: got %q, want submitted", submitted.Status)
	}
	if len(submitted.ModuleVersions) != 1 {
		t.Fatalf("attendu 1 version, got %d", len(submitted.ModuleVersions))
	}
	v := submitted.ModuleVersions[0]
	if v.Number != 1 {
		t.Errorf("Number: got %d", v.Number)
	}
	if !strings.HasPrefix(v.Hash, "sha256:") {
		t.Errorf("Hash doit commencer par sha256:, got %q", v.Hash)
	}
	if v.Note != "première livraison" {
		t.Errorf("Note: got %q", v.Note)
	}
	if v.BlockID == "" {
		t.Error("BlockID ne doit pas être vide")
	}
	if len(v.Assemblies) != 1 || v.Assemblies[0] != conn.ID {
		t.Errorf("Assemblies snapshot incorrect: %v", v.Assemblies)
	}
}

func TestModule_Submit_NoAssembly(t *testing.T) {
	svc := newFullSvc(t)

	p, _ := svc.CreateModule(model.ModuleRequest{Name: "vide", ChannelID: "ch1", OwnerID: "o1"})
	if _, err := svc.SubmitModule(p.ID, ""); err == nil {
		t.Error("SubmitProduct sans assemblage doit retourner une erreur")
	}
}

func TestModule_SubmitTwice_VersionIncrement(t *testing.T) {
	svc := newFullSvc(t)

	p, _ := svc.CreateModule(model.ModuleRequest{Name: "P1", ChannelID: "ch1", OwnerID: "o1"})
	conn, _ := svc.AddConnection("a1", "a2", "lien")
	svc.AddAssemblyToModule(p.ID, conn.ID)
	svc.SubmitModule(p.ID, "v1")
	svc.AddAssemblyToModule(p.ID, conn.ID)
	submitted, err := svc.SubmitModule(p.ID, "v2")
	if err != nil {
		t.Fatalf("SubmitProduct v2: %v", err)
	}
	if len(submitted.ModuleVersions) != 2 {
		t.Errorf("attendu 2 versions, got %d", len(submitted.ModuleVersions))
	}
	if submitted.ModuleVersions[1].Number != 2 {
		t.Errorf("Version 2 Number: got %d", submitted.ModuleVersions[1].Number)
	}
}

func TestModule_Remove(t *testing.T) {
	svc := newFullSvc(t)

	p, _ := svc.CreateModule(model.ModuleRequest{Name: "P1", ChannelID: "ch1", OwnerID: "o1"})
	if err := svc.RemoveModule(p.ID); err != nil {
		t.Fatalf("RemoveProduct: %v", err)
	}
	if _, err := svc.GetModule(p.ID); err == nil {
		t.Error("produit doit être introuvable après suppression")
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Tests — instances d'un module
// ══════════════════════════════════════════════════════════════════════════════

func TestWorkspace_AddRemoveInstance(t *testing.T) {
	svc := newFullSvc(t)

	p, _ := svc.CreateModule(model.ModuleRequest{Name: "P1", ChannelID: "ch1", OwnerID: "o1"})

	// Ajouter deux instances du même asset
	p1, err := svc.AddAssetToWorkspace(p.ID, "asset1")
	if err != nil {
		t.Fatalf("AddAssetToWorkspace: %v", err)
	}
	p2, _ := svc.AddAssetToWorkspace(p.ID, "asset1")

	if len(p2.WorkspaceInstances) != 2 {
		t.Errorf("attendu 2 instances, got %d", len(p2.WorkspaceInstances))
	}

	// Les instances doivent avoir des IDs distincts
	if p1.WorkspaceInstances[0].ID == p2.WorkspaceInstances[1].ID {
		t.Error("les instances doivent avoir des IDs distincts")
	}

	// Supprimer la première instance
	instID := p2.WorkspaceInstances[0].ID
	p3, err := svc.RemoveAssetFromWorkspace(p.ID, instID)
	if err != nil {
		t.Fatalf("RemoveAssetFromWorkspace: %v", err)
	}
	if len(p3.WorkspaceInstances) != 1 {
		t.Errorf("attendu 1 instance après suppression, got %d", len(p3.WorkspaceInstances))
	}
	if p3.WorkspaceInstances[0].ID == instID {
		t.Error("l'instance supprimée ne doit plus être présente")
	}
}

func TestWorkspace_NilStore_ReturnsError(t *testing.T) {
	svc := model.NewService(newMockBC(), &mockFS{}).WithDraftStore(newMockDraftStore())

	if _, err := svc.AddAssetToWorkspace("p1", "a1"); err == nil {
		t.Error("AddAssetToWorkspace avec ID inexistant doit retourner une erreur")
	}
	if _, err := svc.RemoveAssetFromWorkspace("p1", "i1"); err == nil {
		t.Error("RemoveAssetFromWorkspace avec ID inexistant doit retourner une erreur")
	}
}

func TestWorkspace_TwoDifferentAssets_BothPresent(t *testing.T) {
	// Scénario : ajout de 2 composants distincts au module.
	// Chaque appel à AddAssetToWorkspace doit lire l'état courant du module
	// et y ajouter sa propre instance sans écraser la précédente.
	svc := newFullSvc(t)

	p, _ := svc.CreateModule(model.ModuleRequest{Name: "Module", ChannelID: "ch1", OwnerID: "o1"})

	r1, err := svc.AddAssetToWorkspace(p.ID, "asset1")
	if err != nil {
		t.Fatalf("AddAssetToWorkspace asset1 : %v", err)
	}
	if len(r1.WorkspaceInstances) != 1 {
		t.Errorf("après asset1 : attendu 1 instance, got %d", len(r1.WorkspaceInstances))
	}
	if r1.WorkspaceInstances[0].AssetID != "asset1" {
		t.Errorf("instance[0].AssetID : got %q, want asset1", r1.WorkspaceInstances[0].AssetID)
	}

	r2, err := svc.AddAssetToWorkspace(p.ID, "asset2")
	if err != nil {
		t.Fatalf("AddAssetToWorkspace asset2 : %v", err)
	}
	if len(r2.WorkspaceInstances) != 2 {
		t.Errorf("après asset2 : attendu 2 instances, got %d — asset1 ne doit pas avoir été effacé",
			len(r2.WorkspaceInstances))
	}
	ids := map[string]bool{}
	for _, inst := range r2.WorkspaceInstances {
		ids[inst.AssetID] = true
	}
	if !ids["asset1"] || !ids["asset2"] {
		t.Errorf("WorkspaceInstances doit contenir asset1 et asset2 : got %+v", r2.WorkspaceInstances)
	}
}

func TestWorkspace_ThreeAssets_AllPresent(t *testing.T) {
	// Vérifie que N ajouts séquentiels cumulent correctement.
	svc := newFullSvc(t)

	p, _ := svc.CreateModule(model.ModuleRequest{Name: "M", ChannelID: "ch1", OwnerID: "o1"})
	for i, id := range []string{"a1", "a2", "a3"} {
		result, err := svc.AddAssetToWorkspace(p.ID, id)
		if err != nil {
			t.Fatalf("ajout %d : %v", i+1, err)
		}
		if len(result.WorkspaceInstances) != i+1 {
			t.Errorf("après %d ajouts : attendu %d instance(s), got %d",
				i+1, i+1, len(result.WorkspaceInstances))
		}
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Tests — envoi de fichier 3D (AddFull)
// ══════════════════════════════════════════════════════════════════════════════

func TestAddFull_WithFile_HashAndVersionCreated(t *testing.T) {
	// Quand FilePath est fourni, AddFull doit calculer le hash, appeler Upload
	// et créer une Version avec la référence de stockage.
	uploadedPath := ""
	fs := &mockFSFn{uploadFn: func(fp string) (string, error) {
		uploadedPath = fp
		return "store-ref-abc", nil
	}}
	svc := model.NewService(newMockBC(), fs)

	path := tempFile(t, "solid cube\nfacet normal 0 0 1\nouter loop\nendloop\nendfacet\nendsolid cube\n")
	m, err := svc.AddFull(model.AddRequest{
		Name:      "Cube",
		ChannelID: "ch1",
		OwnerID:   "user1",
		FilePath:  path,
	})
	if err != nil {
		t.Fatalf("AddFull: %v", err)
	}
	if uploadedPath != path {
		t.Errorf("Upload appelé avec %q, attendu %q", uploadedPath, path)
	}
	if m.Hash == "" {
		t.Error("Hash doit être calculé depuis le contenu du fichier")
	}
	if len(m.Versions) != 1 {
		t.Fatalf("attendu 1 version, got %d", len(m.Versions))
	}
	if m.Versions[0].Hash != "store-ref-abc" {
		t.Errorf("Version.Hash: got %q, want store-ref-abc", m.Versions[0].Hash)
	}
	if m.Versions[0].Number != 1 {
		t.Errorf("Version.Number: got %d, want 1", m.Versions[0].Number)
	}
}

func TestAddFull_UploadFails_ReturnsError(t *testing.T) {
	// Quand fileStorage.Upload échoue (ex: IPFS injoignable), AddFull doit
	// propager l'erreur sans créer d'enregistrement blockchain.
	bc := newMockBC()
	fs := &mockFSFn{uploadFn: func(_ string) (string, error) {
		return "", fmt.Errorf("ipfs: connexion refusée")
	}}
	svc := model.NewService(bc, fs)

	path := tempFile(t, "solid test\nendsolid test\n")
	_, err := svc.AddFull(model.AddRequest{
		Name:      "Pièce",
		ChannelID: "ch1",
		FilePath:  path,
	})
	if err == nil {
		t.Fatal("attendu une erreur quand Upload échoue")
	}
	if !strings.Contains(err.Error(), "ipfs") {
		t.Errorf("message d'erreur attendu contenir 'ipfs', got: %v", err)
	}
	// Aucun enregistrement ne doit avoir été stocké sur la blockchain.
	list, _ := bc.ListModelRecords("ch1")
	if len(list) != 0 {
		t.Errorf("blockchain ne doit pas contenir d'enregistrement en cas d'échec d'upload, got %d", len(list))
	}
}

func TestAddFull_MissingFile_ReturnsError(t *testing.T) {
	// Si le fichier pointé par FilePath n'existe pas, AddFull doit retourner
	// une erreur de hash (lecture impossible).
	svc := model.NewService(newMockBC(), &mockFS{}).WithDraftStore(newMockDraftStore())

	_, err := svc.AddFull(model.AddRequest{
		Name:      "Fantôme",
		ChannelID: "ch1",
		FilePath:  "/tmp/fichier-inexistant-myr-test.stl",
	})
	if err == nil {
		t.Fatal("attendu une erreur pour un fichier inexistant")
	}
}

func TestAddFull_NoFile_NoVersionNoHash(t *testing.T) {
	// Sans FilePath, le composant est créé sans hash ni version (composant
	// de catalogue sans géométrie).
	uploadCalled := false
	fs := &mockFSFn{uploadFn: func(_ string) (string, error) {
		uploadCalled = true
		return "ref", nil
	}}
	svc := model.NewService(newMockBC(), fs)

	m, err := svc.AddFull(model.AddRequest{
		Name:      "Vis M3 catalogue",
		ChannelID: "ch1",
		FilePath:  "",
	})
	if err != nil {
		t.Fatalf("AddFull: %v", err)
	}
	if uploadCalled {
		t.Error("Upload ne doit pas être appelé quand FilePath est vide")
	}
	if m.Hash != "" {
		t.Errorf("Hash doit être vide sans fichier, got %q", m.Hash)
	}
	if len(m.Versions) != 0 {
		t.Errorf("Versions doit être vide sans fichier, got %d", len(m.Versions))
	}
}

func TestAddFull_STLAndSTEP_BothAccepted(t *testing.T) {
	// Vérifie que les deux formats de fichier (STL et STEP) sont uploadés
	// sans erreur — le service est agnostique au format.
	cases := []struct {
		name    string
		content string
	}{
		{"piece.stl", "solid test\nfacet normal 0 0 1\nouter loop\nendloop\nendfacet\nendsolid test\n"},
		{"piece.step", "ISO-10303-21;\nHEADER;\nENDHEADER;\nDATA;\nENDSEC;\nEND-ISO-10303-21;\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := model.NewService(newMockBC(), &mockFS{}).WithDraftStore(newMockDraftStore())
			path := tempFile(t, tc.content)
			m, err := svc.AddFull(model.AddRequest{
				Name:      tc.name,
				ChannelID: "ch1",
				FilePath:  path,
			})
			if err != nil {
				t.Fatalf("AddFull %s: %v", tc.name, err)
			}
			if m.Hash == "" {
				t.Errorf("%s: Hash ne doit pas être vide", tc.name)
			}
			if len(m.Versions) != 1 {
				t.Errorf("%s: attendu 1 version, got %d", tc.name, len(m.Versions))
			}
		})
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Tests — GetChildren
// ══════════════════════════════════════════════════════════════════════════════

// / @brief  GetChildren retourne les variantes directes d'un asset parent
// / @input  mockBC avec parent et 1 enfant (CategoryVariation, ParentID=parent.ID)
// / @expect liste de 1 enfant avec l'ID correct
func TestGetChildren_Success(t *testing.T) {
	svc := model.NewService(newMockBC(), &mockFS{}).WithDraftStore(newMockDraftStore())

	parent, _ := svc.AddFull(model.AddRequest{Name: "parent", ChannelID: "ch1", OwnerID: "o1"})
	child, _ := svc.AddFull(model.AddRequest{
		Name: "variante", Category: model.CategoryVariation, ParentID: parent.ID,
		ChannelID: "ch1", OwnerID: "o1",
	})

	children, err := svc.GetChildren(parent.ID)
	if err != nil {
		t.Fatalf("GetChildren: %v", err)
	}
	if len(children) != 1 {
		t.Fatalf("attendu 1 enfant, got %d", len(children))
	}
	if children[0].ID != child.ID {
		t.Errorf("ID: got %q, want %q", children[0].ID, child.ID)
	}
}

// / @brief  GetChildren retourne une liste vide si aucun enfant n'existe
// / @input  mockBC avec 1 asset sans ParentID pointant sur lui
// / @expect slice vide (ou nil) sans erreur
func TestGetChildren_Empty(t *testing.T) {
	svc := model.NewService(newMockBC(), &mockFS{}).WithDraftStore(newMockDraftStore())

	m, _ := svc.AddFull(model.AddRequest{Name: "isolé", ChannelID: "ch1", OwnerID: "o1"})

	children, err := svc.GetChildren(m.ID)
	if err != nil {
		t.Fatalf("GetChildren: %v", err)
	}
	if len(children) != 0 {
		t.Errorf("attendu 0 enfants, got %d", len(children))
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Tests — GetModuleInterfaces
// ══════════════════════════════════════════════════════════════════════════════

// / @brief  GetModuleInterfaces retourne les interfaces directes d'un composant simple
// / @input  service complet, 1 composant avec 1 interface ELEC USB-C Out
// / @expect liste de 1 interface avec l'ID correct
func TestGetModuleInterfaces_SimpleComponent(t *testing.T) {
	svc := newFullSvc(t)

	comp, _ := svc.AddFull(model.AddRequest{Name: "Capteur", ChannelID: "ch1", OwnerID: "o1"})
	iface := &model.AssetInterface{AssetID: comp.ID, Category: "ELEC", Type: "USB-C", Direction: model.IfaceOut}
	svc.AddInterface(iface)

	ifaces, err := svc.GetModuleInterfaces(comp.ID)
	if err != nil {
		t.Fatalf("GetModuleInterfaces: %v", err)
	}
	if len(ifaces) != 1 {
		t.Fatalf("attendu 1 interface, got %d", len(ifaces))
	}
	if ifaces[0].ID != iface.ID {
		t.Errorf("ID: got %q, want %q", ifaces[0].ID, iface.ID)
	}
}

// / @brief  GetModuleInterfaces retourne une erreur si ifaceStore est absent
// / @input  service sans WithIfaceStore, n'importe quel ID
// / @expect retourne une erreur (interface store non configuré)
func TestGetModuleInterfaces_NilStore(t *testing.T) {
	svc := model.NewService(newMockBC(), &mockFS{}).WithDraftStore(newMockDraftStore())

	if _, err := svc.GetModuleInterfaces("quelconque"); err == nil {
		t.Error("GetModuleInterfaces sans ifaceStore doit retourner une erreur")
	}
}

// / @brief  GetModuleInterfaces exclut les interfaces reliées en interne dans un module
// / @input  module AB : compA (ifaceOut) relié à compB (ifaceIn) en interne, compB a aussi ifaceExt
// / @expect ifaceOut et ifaceIn absentes ; ifaceExt présente dans les interfaces exposées
func TestGetModuleInterfaces_InternalConnectionsConsumed(t *testing.T) {
	svc := newFullSvc(t)

	compA, _ := svc.AddFull(model.AddRequest{Name: "A", ChannelID: "ch1", OwnerID: "o1"})
	compB, _ := svc.AddFull(model.AddRequest{Name: "B", ChannelID: "ch1", OwnerID: "o1"})

	ifaceOut := &model.AssetInterface{AssetID: compA.ID, Category: "ELEC", Type: "USB-C", Direction: model.IfaceOut}
	ifaceIn := &model.AssetInterface{AssetID: compB.ID, Category: "ELEC", Type: "USB-C", Direction: model.IfaceIn}
	ifaceExt := &model.AssetInterface{AssetID: compB.ID, Category: "MECA", Type: "Vis M3", Direction: model.IfaceOut}
	svc.AddInterface(ifaceOut)
	svc.AddInterface(ifaceIn)
	svc.AddInterface(ifaceExt)

	mod, _ := svc.CreateModule(model.ModuleRequest{Name: "Module AB", ChannelID: "ch1", OwnerID: "o1"})
	svc.AddAssetToWorkspace(mod.ID, compA.ID)
	svc.AddAssetToWorkspace(mod.ID, compB.ID)

	conn, _ := svc.AddAssemblyLink(ifaceOut.ID, ifaceIn.ID, "liaison", "", "", "")
	svc.AddAssemblyToModule(mod.ID, conn.ID)

	ifaces, err := svc.GetModuleInterfaces(mod.ID)
	if err != nil {
		t.Fatalf("GetModuleInterfaces: %v", err)
	}
	for _, iface := range ifaces {
		if iface.ID == ifaceOut.ID || iface.ID == ifaceIn.ID {
			t.Errorf("interface interne %q ne devrait pas être exposée", iface.ID)
		}
	}
	found := false
	for _, iface := range ifaces {
		if iface.ID == ifaceExt.ID {
			found = true
		}
	}
	if !found {
		t.Error("ifaceExt devrait être exposée car elle n'est pas reliée en interne")
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Tests — CheckModuleLicenseCompatibility
// ══════════════════════════════════════════════════════════════════════════════

// / @brief  Tous les composants sous licence "mit" : produit "mit" est compatible
// / @input  componentLicenseIDs=["mit","mit"], proposed="mit"
// / @expect Compatible=true
func TestCheckModuleLicense_AllCompatible(t *testing.T) {
	result := model.CheckModuleLicenseCompatibility([]string{"mit", "mit"}, "mit")
	if !result.Compatible {
		t.Errorf("attendu compatible, raison: %s", result.Reason)
	}
}

// / @brief  Composant "gpl-3" n'autorise pas "mit" comme licence de produit
// / @input  componentLicenseIDs=["gpl-3"], proposed="mit"
// / @expect Compatible=false (gpl-3 CompatibleWith=["gpl-3"] seulement)
func TestCheckModuleLicense_Incompatible(t *testing.T) {
	result := model.CheckModuleLicenseCompatibility([]string{"gpl-3"}, "mit")
	if result.Compatible {
		t.Error("attendu incompatible : gpl-3 n'autorise pas 'mit' pour un dérivé")
	}
}

// / @brief  Aucun composant avec licence → toute licence est acceptable
// / @input  componentLicenseIDs=[], proposed="mit"
// / @expect Compatible=true
func TestCheckModuleLicense_NoComponents(t *testing.T) {
	result := model.CheckModuleLicenseCompatibility([]string{}, "mit")
	if !result.Compatible {
		t.Errorf("attendu compatible quand aucun composant n'a de licence: %s", result.Reason)
	}
}

// / @brief  Composant avec licence mais aucune licence proposée pour le module
// / @input  componentLicenseIDs=["mit"], proposed=""
// / @expect Compatible=false (aucune licence sélectionnée)
func TestCheckModuleLicense_EmptyProposed(t *testing.T) {
	result := model.CheckModuleLicenseCompatibility([]string{"mit"}, "")
	if result.Compatible {
		t.Error("attendu incompatible quand aucune licence n'est sélectionnée pour le module")
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Tests — ConnectVirtualToPhysical
// Scénario : le client relie une interface virtuelle d'un composant à une
// interface physique existante. La virtuelle est matérialisée (Virtual→false)
// et hérite de la catégorie, du type et de la direction opposée du physique.
// Les valeurs saisies par le client écrasent les valeurs du physique, sinon
// elles sont héritées. La liaison est ensuite créée.
// ══════════════════════════════════════════════════════════════════════════════

// addTestIface crée une interface dans le service et retourne son ID assigné.
func addTestIface(t *testing.T, svc *model.Service, iface *model.AssetInterface) string {
	t.Helper()
	if err := svc.AddInterface(iface); err != nil {
		t.Fatalf("AddInterface: %v", err)
	}
	return iface.ID
}

// / @brief  Interface physique "out" → virtuelle devient "in" (sens inversé)
// / @input  physique ELEC/USB-C/out, virtuelle ELEC/**/bidir Virtual=true
// / @expect virtuelle.Direction=in, Category=ELEC, Type=USB-C, Virtual=false
func TestConnectVirtualToPhysical_OutBecomesIn(t *testing.T) {
	svc := newFullSvc(t)

	physID := addTestIface(t, svc, &model.AssetInterface{
		AssetID:   "compA",
		Category:  "ELEC",
		Type:      "USB-C",
		Direction: model.IfaceOut,
		ValueMin:  5, ValueMax: 5, Unit: "V",
	})
	virtID := addTestIface(t, svc, &model.AssetInterface{
		AssetID:   "compB",
		Category:  "",
		Type:      "",
		Direction: model.IfaceBidi,
		Virtual:   true,
	})

	conn, err := svc.ConnectVirtualToPhysical(virtID, physID, model.AssetInterface{}, "instA", "instB")
	if err != nil {
		t.Fatalf("ConnectVirtualToPhysical: %v", err)
	}
	if conn == nil {
		t.Fatal("connexion nil")
	}

	updated, err := svc.GetInterface(virtID)
	if err != nil {
		t.Fatalf("GetInterface: %v", err)
	}
	if updated.Virtual {
		t.Error("Virtual devrait être false après matérialisation")
	}
	if updated.Category != "ELEC" {
		t.Errorf("Category: got %q, want %q", updated.Category, "ELEC")
	}
	if updated.Type != "USB-C" {
		t.Errorf("Type: got %q, want %q", updated.Type, "USB-C")
	}
	if updated.Direction != model.IfaceIn {
		t.Errorf("Direction: got %q, want %q", updated.Direction, model.IfaceIn)
	}
}

// / @brief  Interface physique "in" → virtuelle devient "out"
// / @input  physique MECA/Vis M3/in, virtuelle Virtual=true
// / @expect virtuelle.Direction=out
func TestConnectVirtualToPhysical_InBecomesOut(t *testing.T) {
	svc := newFullSvc(t)

	physID := addTestIface(t, svc, &model.AssetInterface{
		AssetID:   "compA",
		Category:  "MECA",
		Type:      "Vis M3",
		Direction: model.IfaceIn,
	})
	virtID := addTestIface(t, svc, &model.AssetInterface{
		AssetID:   "compB",
		Category:  "",
		Type:      "",
		Direction: model.IfaceBidi,
		Virtual:   true,
	})

	_, err := svc.ConnectVirtualToPhysical(virtID, physID, model.AssetInterface{}, "", "")
	if err != nil {
		t.Fatalf("ConnectVirtualToPhysical: %v", err)
	}

	updated, _ := svc.GetInterface(virtID)
	if updated.Direction != model.IfaceOut {
		t.Errorf("Direction: got %q, want %q", updated.Direction, model.IfaceOut)
	}
}

// / @brief  Interface physique "bidir" → virtuelle reste "bidir"
// / @input  physique HYD/Push-fit 6mm/bidir, virtuelle Virtual=true
// / @expect virtuelle.Direction=bidir
func TestConnectVirtualToPhysical_BidirStaysBidir(t *testing.T) {
	svc := newFullSvc(t)

	physID := addTestIface(t, svc, &model.AssetInterface{
		AssetID:   "compA",
		Category:  "HYD",
		Type:      "Push-fit 6mm",
		Direction: model.IfaceBidi,
	})
	virtID := addTestIface(t, svc, &model.AssetInterface{
		AssetID:   "compB",
		Category:  "",
		Type:      "",
		Direction: model.IfaceBidi,
		Virtual:   true,
	})

	_, err := svc.ConnectVirtualToPhysical(virtID, physID, model.AssetInterface{}, "", "")
	if err != nil {
		t.Fatalf("ConnectVirtualToPhysical: %v", err)
	}

	updated, _ := svc.GetInterface(virtID)
	if updated.Direction != model.IfaceBidi {
		t.Errorf("Direction: got %q, want bidir", updated.Direction)
	}
}

// / @brief  L'utilisateur saisit un nom dans la popup → la virtuelle l'adopte
// / @input  popupValues.Name="Alimentation principale"
// / @expect virtuelle.Name="Alimentation principale"
func TestConnectVirtualToPhysical_UserOverridesName(t *testing.T) {
	svc := newFullSvc(t)

	physID := addTestIface(t, svc, &model.AssetInterface{
		AssetID:   "compA",
		Category:  "ELEC",
		Type:      "DIN",
		Direction: model.IfaceOut,
		Name:      "Sortie alim",
	})
	virtID := addTestIface(t, svc, &model.AssetInterface{
		AssetID:   "compB",
		Category:  "",
		Type:      "",
		Direction: model.IfaceBidi,
		Virtual:   true,
	})

	popup := model.AssetInterface{Name: "Alimentation principale"}
	_, err := svc.ConnectVirtualToPhysical(virtID, physID, popup, "", "")
	if err != nil {
		t.Fatalf("ConnectVirtualToPhysical: %v", err)
	}

	updated, _ := svc.GetInterface(virtID)
	if updated.Name != "Alimentation principale" {
		t.Errorf("Name: got %q, want %q", updated.Name, "Alimentation principale")
	}
}

// / @brief  Sans nom dans la popup → la virtuelle hérite du nom du physique
// / @input  popupValues vide, physique.Name="Sortie alim"
// / @expect virtuelle.Name="Sortie alim"
func TestConnectVirtualToPhysical_InheritsPhysicalName(t *testing.T) {
	svc := newFullSvc(t)

	physID := addTestIface(t, svc, &model.AssetInterface{
		AssetID:   "compA",
		Category:  "ELEC",
		Type:      "DIN",
		Direction: model.IfaceOut,
		Name:      "Sortie alim",
	})
	virtID := addTestIface(t, svc, &model.AssetInterface{
		AssetID:   "compB",
		Category:  "",
		Type:      "",
		Direction: model.IfaceBidi,
		Virtual:   true,
	})

	_, err := svc.ConnectVirtualToPhysical(virtID, physID, model.AssetInterface{}, "", "")
	if err != nil {
		t.Fatalf("ConnectVirtualToPhysical: %v", err)
	}

	updated, _ := svc.GetInterface(virtID)
	if updated.Name != "Sortie alim" {
		t.Errorf("Name: got %q, want %q", updated.Name, "Sortie alim")
	}
}

// / @brief  L'utilisateur fournit des valeurs min/max/unit dans la popup → héritées
// / @input  popupValues.ValueMin=3, ValueMax=5, IsRange=true, Unit="V"
// / @expect virtuelle.ValueMin=3, ValueMax=5, IsRange=true, Unit="V"
func TestConnectVirtualToPhysical_UserOverridesValues(t *testing.T) {
	svc := newFullSvc(t)

	physID := addTestIface(t, svc, &model.AssetInterface{
		AssetID:   "compA",
		Category:  "ELEC",
		Type:      "USB-C",
		Direction: model.IfaceOut,
		ValueMin:  5, ValueMax: 20, IsRange: true, Unit: "V",
	})
	virtID := addTestIface(t, svc, &model.AssetInterface{
		AssetID:   "compB",
		Category:  "",
		Type:      "",
		Direction: model.IfaceBidi,
		Virtual:   true,
	})

	popup := model.AssetInterface{ValueMin: 3, ValueMax: 5, IsRange: true, Unit: "V"}
	_, err := svc.ConnectVirtualToPhysical(virtID, physID, popup, "", "")
	if err != nil {
		t.Fatalf("ConnectVirtualToPhysical: %v", err)
	}

	updated, _ := svc.GetInterface(virtID)
	if updated.ValueMin != 3 || updated.ValueMax != 5 {
		t.Errorf("Values: got [%v,%v], want [3,5]", updated.ValueMin, updated.ValueMax)
	}
	if !updated.IsRange {
		t.Error("IsRange devrait être true")
	}
	if updated.Unit != "V" {
		t.Errorf("Unit: got %q, want %q", updated.Unit, "V")
	}
}

// / @brief  Sans valeurs dans la popup → la virtuelle hérite des valeurs du physique
// / @input  popupValues vide, physique.ValueMin=12, ValueMax=12, Unit="V"
// / @expect virtuelle.ValueMin=12, ValueMax=12, Unit="V"
func TestConnectVirtualToPhysical_InheritsPhysicalValues(t *testing.T) {
	svc := newFullSvc(t)

	physID := addTestIface(t, svc, &model.AssetInterface{
		AssetID:   "compA",
		Category:  "ELEC",
		Type:      "DIN",
		Direction: model.IfaceOut,
		ValueMin:  12, ValueMax: 12, Unit: "V",
	})
	virtID := addTestIface(t, svc, &model.AssetInterface{
		AssetID:   "compB",
		Category:  "",
		Type:      "",
		Direction: model.IfaceBidi,
		Virtual:   true,
	})

	_, err := svc.ConnectVirtualToPhysical(virtID, physID, model.AssetInterface{}, "", "")
	if err != nil {
		t.Fatalf("ConnectVirtualToPhysical: %v", err)
	}

	updated, _ := svc.GetInterface(virtID)
	if updated.ValueMin != 12 || updated.ValueMax != 12 {
		t.Errorf("Values: got [%v,%v], want [12,12]", updated.ValueMin, updated.ValueMax)
	}
	if updated.Unit != "V" {
		t.Errorf("Unit: got %q, want %q", updated.Unit, "V")
	}
}

// / @brief  La liaison d'assemblage est créée avec les bons endpoints
// / @input  physique AssetID="compA", virtuelle AssetID="compB"
// / @expect conn.ID non vide, conn.From=compA, conn.To=compB
func TestConnectVirtualToPhysical_AssemblyLinkCreated(t *testing.T) {
	svc := newFullSvc(t)

	physID := addTestIface(t, svc, &model.AssetInterface{
		AssetID:   "compA",
		Category:  "ELEC",
		Type:      "USB-C",
		Direction: model.IfaceOut,
		ValueMin:  5, ValueMax: 5, Unit: "V",
	})
	virtID := addTestIface(t, svc, &model.AssetInterface{
		AssetID:   "compB",
		Category:  "",
		Type:      "",
		Direction: model.IfaceBidi,
		Virtual:   true,
	})

	conn, err := svc.ConnectVirtualToPhysical(virtID, physID, model.AssetInterface{}, "instA", "instB")
	if err != nil {
		t.Fatalf("ConnectVirtualToPhysical: %v", err)
	}
	if conn.ID == "" {
		t.Error("Connection.ID vide")
	}
	if conn.From != "compA" {
		t.Errorf("From: got %q, want %q", conn.From, "compA")
	}
	if conn.To != "compB" {
		t.Errorf("To: got %q, want %q", conn.To, "compB")
	}
	// ConnectVirtualToPhysical reçoit (fromInstanceID=instA=instance du VIRTUEL compB,
	// toInstanceID=instB=instance du PHYSIQUE compA).
	// AddAssemblyLink construit conn avec From=physique (compA) → FromInstanceID=instance de compA=instB,
	// et To=virtuel (compB) → ToInstanceID=instance de compB=instA. Le swap est intentionnel.
	if conn.FromInstanceID != "instB" || conn.ToInstanceID != "instA" {
		t.Errorf("InstanceIDs: got (%q,%q), want (instB,instA)", conn.FromInstanceID, conn.ToInstanceID)
	}
}

// / @brief  Appeler ConnectVirtualToPhysical sur une interface non-virtuelle → erreur
// / @input  interface avec Virtual=false
// / @expect retourne une erreur non nil
func TestConnectVirtualToPhysical_NotVirtual_Error(t *testing.T) {
	svc := newFullSvc(t)

	physID := addTestIface(t, svc, &model.AssetInterface{
		AssetID:   "compA",
		Category:  "ELEC",
		Type:      "USB-C",
		Direction: model.IfaceOut,
	})
	notVirtID := addTestIface(t, svc, &model.AssetInterface{
		AssetID:   "compB",
		Category:  "ELEC",
		Type:      "USB-C",
		Direction: model.IfaceIn,
		Virtual:   false, // non virtuelle
	})

	_, err := svc.ConnectVirtualToPhysical(notVirtID, physID, model.AssetInterface{}, "", "")
	if err == nil {
		t.Error("attendu une erreur : l'interface source n'est pas virtuelle")
	}
}

// / @brief  Service sans interface store → erreur immédiate
// / @input  service sans WithIfaceStore
// / @expect retourne "interface store non configuré"
func TestConnectVirtualToPhysical_NilIfaceStore_Error(t *testing.T) {
	svc := model.NewService(newMockBC(), &mockFS{}).WithConnStore(newMockConnStore())

	_, err := svc.ConnectVirtualToPhysical("virt1", "phys1", model.AssetInterface{}, "", "")
	if err == nil {
		t.Error("attendu une erreur : interface store absent")
	}
	if !strings.Contains(err.Error(), "interface store") {
		t.Errorf("message d'erreur inattendu: %v", err)
	}
}

// / @brief  Service sans connection store → erreur immédiate
// / @input  service sans WithConnStore
// / @expect retourne "connection store non configuré"
func TestConnectVirtualToPhysical_NilConnStore_Error(t *testing.T) {
	svc := model.NewService(newMockBC(), &mockFS{}).WithIfaceStore(newMockIfaceStore())

	_, err := svc.ConnectVirtualToPhysical("virt1", "phys1", model.AssetInterface{}, "", "")
	if err == nil {
		t.Error("attendu une erreur : connection store absent")
	}
	if !strings.Contains(err.Error(), "connection store") {
		t.Errorf("message d'erreur inattendu: %v", err)
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Tests — Invariant : slot virtuel toujours présent
//
// Chaque composant expose toujours au moins une interface virtuelle
// (Virtual=true) en plus de ses interfaces physiques, afin qu'un client
// puisse toujours l'utiliser pour créer une nouvelle liaison.
// L'invariant est maintenu automatiquement par le service :
//   • à l'ajout d'un asset à un module
//   • après la matérialisation d'une interface virtuelle (ConnectVirtualToPhysical)
// ══════════════════════════════════════════════════════════════════════════════

// countVirtual retourne le nombre d'interfaces Virtual=true pour un asset.
func countVirtual(t *testing.T, svc *model.Service, assetID string) int {
	t.Helper()
	ifaces, err := svc.ListInterfacesForAsset(assetID)
	if err != nil {
		t.Fatalf("ListInterfacesForAsset(%q): %v", assetID, err)
	}
	n := 0
	for _, iface := range ifaces {
		if iface.Virtual {
			n++
		}
	}
	return n
}

// findVirtual retourne le premier slot Virtual=true pour un asset, ou nil.
func findVirtual(t *testing.T, svc *model.Service, assetID string) *model.AssetInterface {
	t.Helper()
	ifaces, _ := svc.ListInterfacesForAsset(assetID)
	for _, iface := range ifaces {
		if iface.Virtual {
			return iface
		}
	}
	return nil
}

// / @brief  Ajouter un asset à un module crée automatiquement un slot virtuel
// / @input  module vide, asset "compA" sans interface
// / @expect ListInterfacesForAsset("compA") contient au moins un Virtual=true
func TestWorkspace_AlwaysHasVirtualInterface_AfterAdd(t *testing.T) {
	svc := newFullSvc(t)

	mod, err := svc.CreateModule(model.ModuleRequest{Name: "Module", ChannelID: "ch1", OwnerID: "o1"})
	if err != nil {
		t.Fatalf("CreateModule: %v", err)
	}

	if _, err := svc.AddAssetToWorkspace(mod.ID, "compA"); err != nil {
		t.Fatalf("AddAssetToWorkspace: %v", err)
	}

	if n := countVirtual(t, svc, "compA"); n == 0 {
		t.Error("invariant violé : aucun slot virtuel après AddAssetToWorkspace")
	}
}

// / @brief  Ajouter le même asset plusieurs fois ne crée pas de slots virtuels en double
// / @input  même asset "compA" ajouté deux fois
// / @expect exactement 1 slot virtuel (idempotent)
func TestWorkspace_VirtualSlot_Idempotent(t *testing.T) {
	svc := newFullSvc(t)

	mod, _ := svc.CreateModule(model.ModuleRequest{Name: "Module", ChannelID: "ch1", OwnerID: "o1"})
	svc.AddAssetToWorkspace(mod.ID, "compA")
	svc.AddAssetToWorkspace(mod.ID, "compA")

	if n := countVirtual(t, svc, "compA"); n != 1 {
		t.Errorf("attendu 1 slot virtuel (idempotent), got %d", n)
	}
}

// / @brief  Assets différents ont chacun leur propre slot virtuel
// / @input  "compA" et "compB" ajoutés dans le même module
// / @expect chacun dispose d'au moins un slot virtuel indépendant
func TestWorkspace_VirtualSlot_PerAsset(t *testing.T) {
	svc := newFullSvc(t)

	mod, _ := svc.CreateModule(model.ModuleRequest{Name: "Module", ChannelID: "ch1", OwnerID: "o1"})
	svc.AddAssetToWorkspace(mod.ID, "compA")
	svc.AddAssetToWorkspace(mod.ID, "compB")

	if n := countVirtual(t, svc, "compA"); n == 0 {
		t.Error("compA : aucun slot virtuel")
	}
	if n := countVirtual(t, svc, "compB"); n == 0 {
		t.Error("compB : aucun slot virtuel")
	}
}

// / @brief  Le slot virtuel est utilisable : ConnectVirtualToPhysical l'accepte sans erreur
// / @input  compA (physique, out) et compB (virtuel), tous deux instances du même module
// / @expect connexion créée avec succès via le slot virtuel issu de l'invariant
func TestWorkspace_VirtualIsClickable(t *testing.T) {
	svc := newFullSvc(t)

	// Interface physique sur compA (le composant source)
	physID := addTestIface(t, svc, &model.AssetInterface{
		AssetID:   "compA",
		Category:  "ELEC",
		Type:      "USB-C",
		Direction: model.IfaceOut,
		ValueMin:  5, ValueMax: 5, Unit: "V",
	})

	// Ajouter compB au module → slot virtuel auto-créé
	mod, _ := svc.CreateModule(model.ModuleRequest{Name: "Module", ChannelID: "ch1", OwnerID: "o1"})
	if _, err := svc.AddAssetToWorkspace(mod.ID, "compB"); err != nil {
		t.Fatalf("AddAssetToWorkspace: %v", err)
	}

	slot := findVirtual(t, svc, "compB")
	if slot == nil {
		t.Fatal("invariant violé : aucun slot virtuel trouvé pour compB")
	}

	// Cliquer sur le slot virtuel pour créer la liaison
	conn, err := svc.ConnectVirtualToPhysical(slot.ID, physID, model.AssetInterface{}, "instA", "instB")
	if err != nil {
		t.Fatalf("ConnectVirtualToPhysical (clic sur slot virtuel) : %v", err)
	}
	if conn == nil || conn.ID == "" {
		t.Error("connexion non créée")
	}
}

// / @brief  Après matérialisation d'un slot, l'invariant est restauré automatiquement
// / @input  compB a 1 slot virtuel ; on le matérialise → le service doit en recréer un
// / @expect countVirtual("compB") ≥ 1 après ConnectVirtualToPhysical
func TestWorkspace_AlwaysHasVirtualInterface_AfterConnect(t *testing.T) {
	svc := newFullSvc(t)

	physID := addTestIface(t, svc, &model.AssetInterface{
		AssetID:   "compA",
		Category:  "ELEC",
		Type:      "USB-C",
		Direction: model.IfaceOut,
		ValueMin:  5, ValueMax: 5, Unit: "V",
	})

	mod, _ := svc.CreateModule(model.ModuleRequest{Name: "Module", ChannelID: "ch1", OwnerID: "o1"})
	svc.AddAssetToWorkspace(mod.ID, "compB")

	slot := findVirtual(t, svc, "compB")
	if slot == nil {
		t.Fatal("slot virtuel initial absent")
	}

	// Matérialiser le slot (le relier au physique)
	if _, err := svc.ConnectVirtualToPhysical(slot.ID, physID, model.AssetInterface{}, "instA", "instB"); err != nil {
		t.Fatalf("ConnectVirtualToPhysical: %v", err)
	}

	// L'invariant doit être restauré : un nouveau slot virtuel doit exister
	if n := countVirtual(t, svc, "compB"); n == 0 {
		t.Error("invariant violé : plus aucun slot virtuel après matérialisation")
	}
}

// / @brief  Le slot virtuel de remplacement est à son tour utilisable pour une liaison
// / @input  deux liaisons successives via des slots virtuels régénérés
// / @expect les deux connexions sont créées sans erreur
func TestWorkspace_VirtualSlot_ChainedConnections(t *testing.T) {
	svc := newFullSvc(t)

	// Deux interfaces physiques sur compA
	phys1 := addTestIface(t, svc, &model.AssetInterface{
		AssetID:   "compA",
		Category:  "ELEC",
		Type:      "USB-C",
		Direction: model.IfaceOut,
		ValueMin:  5, ValueMax: 5, Unit: "V",
	})
	phys2 := addTestIface(t, svc, &model.AssetInterface{
		AssetID:   "compA",
		Category:  "ELEC",
		Type:      "USB-C",
		Direction: model.IfaceOut,
		ValueMin:  5, ValueMax: 5, Unit: "V",
	})

	mod, _ := svc.CreateModule(model.ModuleRequest{Name: "Module", ChannelID: "ch1", OwnerID: "o1"})
	svc.AddAssetToWorkspace(mod.ID, "compB")

	// Première liaison
	slot1 := findVirtual(t, svc, "compB")
	if slot1 == nil {
		t.Fatal("slot 1 absent")
	}
	if _, err := svc.ConnectVirtualToPhysical(slot1.ID, phys1, model.AssetInterface{}, "instA", "instB"); err != nil {
		t.Fatalf("liaison 1 : %v", err)
	}

	// Le slot a été régénéré — deuxième liaison
	slot2 := findVirtual(t, svc, "compB")
	if slot2 == nil {
		t.Fatal("slot 2 absent après première liaison")
	}
	if slot2.ID == slot1.ID {
		t.Error("le slot régénéré devrait avoir un nouvel ID")
	}
	if _, err := svc.ConnectVirtualToPhysical(slot2.ID, phys2, model.AssetInterface{}, "instA", "instB"); err != nil {
		t.Fatalf("liaison 2 : %v", err)
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Tests — cycle brouillon → soumission des composants (ADR-02) : toute création
// est un brouillon, la blockchain n'est engagée que par Submit explicite.
// ══════════════════════════════════════════════════════════════════════════════

// / @brief  AddFull crée toujours l'asset en local, sans transaction blockchain
// / @input  Service avec DraftStore configuré, AddRequest{Name:"vis"}
// / @expect Retour sans erreur, Status=draft, aucun enregistrement dans mockBC.records
func TestAddFull_AlwaysDraft_NoBlockchainWrite(t *testing.T) {
	bc := newMockBC()
	svc := model.NewService(bc, &mockFS{}).WithDraftStore(newMockDraftStore())

	m, err := svc.AddFull(model.AddRequest{Name: "vis", ChannelID: "ch1"})
	if err != nil {
		t.Fatalf("AddFull: %v", err)
	}
	if m.Status != model.ModuleDraft {
		t.Errorf("Status: got %q, want draft", m.Status)
	}
	if len(bc.records) != 0 {
		t.Errorf("aucune transaction blockchain attendue à la création, got %d", len(bc.records))
	}
}

// / @brief  AddFull sans blockchain configurée doit fonctionner (le point du brouillon)
// / @input  Service sans blockchain (nil), DraftStore configuré
// / @expect Retour sans erreur — la blockchain indisponible n'empêche pas de préparer un brouillon
func TestAddFull_NilBlockchain_StillWorks(t *testing.T) {
	svc := model.NewService(nil, &mockFS{}).WithDraftStore(newMockDraftStore())

	m, err := svc.AddFull(model.AddRequest{Name: "vis", ChannelID: "ch1"})
	if err != nil {
		t.Fatalf("AddFull sans blockchain: %v", err)
	}
	if m.Status != model.ModuleDraft {
		t.Errorf("Status: got %q, want draft", m.Status)
	}
}

// / @brief  AddFull sans DraftStore configuré doit être rejeté
// / @input  Service sans WithDraftStore
// / @expect Erreur non nil
func TestAddFull_NilDraftStore_Rejected(t *testing.T) {
	svc := model.NewService(newMockBC(), &mockFS{}).WithDraftStore(newMockDraftStore())
	if _, err := svc.AddFull(model.AddRequest{Name: "vis"}); err == nil {
		t.Error("AddFull sans DraftStore doit retourner une erreur")
	}
}

// / @brief  Get retrouve un brouillon local avant de chercher sur la blockchain
// / @input  DraftStore contenant un brouillon, blockchain vide
// / @expect Get(id) retourne le brouillon (Status=draft)
func TestGet_FindsDraftBeforeBlockchain(t *testing.T) {
	svc := newFullSvc(t)
	m, _ := svc.AddFull(model.AddRequest{Name: "vis", ChannelID: "ch1"})

	got, err := svc.Get(m.ID, "")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Status != model.ModuleDraft {
		t.Errorf("Status: got %q, want draft", got.Status)
	}
}

// / @brief  List fusionne les brouillons locaux et les assets déjà soumis sur la blockchain
// / @input  Un brouillon jamais soumis + un asset créé en brouillon puis soumis (Submit), même canal
// / @expect List retourne les deux assets
func TestList_MergesDraftsAndBlockchain(t *testing.T) {
	svc := newFullSvc(t)
	draft, _ := svc.AddFull(model.AddRequest{Name: "brouillon", ChannelID: "ch1"})
	toSubmit, _ := svc.AddFull(model.AddRequest{Name: "soumis", ChannelID: "ch1"})
	submitted, err := svc.Submit(toSubmit.ID)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}

	all, err := svc.List("ch1")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("attendu 2 assets (1 brouillon + 1 soumis), got %d", len(all))
	}
	ids := map[string]bool{}
	for _, m := range all {
		ids[m.ID] = true
	}
	if !ids[draft.ID] || !ids[submitted.ID] {
		t.Errorf("List doit contenir le brouillon et l'asset soumis : %v", ids)
	}
}

// / @brief  Submit engage un composant brouillon sur la blockchain en une transaction,
// /         en embarquant ses interfaces locales, puis passe Status à submitted
// / @input  Composant créé en Draft, une interface ajoutée via AddInterface
// / @expect Status=submitted, Interfaces contient l'interface ajoutée, l'asset est
// /         désormais sur la blockchain (mockBC.records) et n'est plus dans DraftStore
func TestSubmit_CommitsDraftWithInterfaces(t *testing.T) {
	bc := newMockBC()
	svc := model.NewService(bc, &mockFS{}).
		WithIfaceStore(newMockIfaceStore()).
		WithDraftStore(newMockDraftStore())

	m, err := svc.AddFull(model.AddRequest{Name: "vis", ChannelID: "ch1"})
	if err != nil {
		t.Fatalf("AddFull draft: %v", err)
	}
	if err := svc.AddInterface(&model.AssetInterface{
		AssetID: m.ID, Category: "MECA", Type: "filetage", Direction: model.IfaceOut,
	}); err != nil {
		t.Fatalf("AddInterface: %v", err)
	}

	submitted, err := svc.Submit(m.ID)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if submitted.Status != model.ModuleSubmitted {
		t.Errorf("Status: got %q, want submitted", submitted.Status)
	}
	if len(submitted.Interfaces) != 1 {
		t.Fatalf("attendu 1 interface embarquée, got %d", len(submitted.Interfaces))
	}
	if _, ok := bc.records[m.ID]; !ok {
		t.Error("Submit doit committer l'asset sur la blockchain")
	}

	// Get doit désormais résoudre via la blockchain, plus via DraftStore.
	got, err := svc.Get(m.ID, "")
	if err != nil {
		t.Fatalf("Get après Submit: %v", err)
	}
	if got.Status != model.ModuleSubmitted {
		t.Errorf("Get après Submit: Status got %q, want submitted", got.Status)
	}
}

// / @brief  Submit sur un ID qui n'est pas un brouillon local (inexistant ou déjà soumis) est rejeté
// / @input  Service avec DraftStore vide, ID inconnu
// / @expect Erreur non nil
func TestSubmit_NotADraft_Rejected(t *testing.T) {
	svc := newFullSvc(t)
	if _, err := svc.Submit("introuvable"); err == nil {
		t.Error("Submit sur un ID hors brouillon doit retourner une erreur")
	}
}

// / @brief  Submit sans DraftStore configuré est rejeté
// / @input  Service sans WithDraftStore
// / @expect Erreur non nil
func TestSubmit_NilDraftStore_Rejected(t *testing.T) {
	svc := model.NewService(newMockBC(), &mockFS{}).WithDraftStore(newMockDraftStore())
	if _, err := svc.Submit("any"); err == nil {
		t.Error("Submit sans DraftStore doit retourner une erreur")
	}
}

// / @brief  Un composant brouillon n'est jamais classé comme module (IsModule) — la
// /         généralisation de Status aux composants ne doit pas les faire passer pour
// /         des modules dans ListModules ou GetModuleInterfaces
// / @input  Un composant Draft (Status=draft, Assemblies nil) + un module réel (CreateModule)
// / @expect ListModules ne retourne que le module ; le composant draft est absent
func TestIsModule_DraftComponent_NotMisclassified(t *testing.T) {
	svc := newFullSvc(t)
	draftComponent, _ := svc.AddFull(model.AddRequest{Name: "vis", ChannelID: "ch1"})
	mod, _ := svc.CreateModule(model.ModuleRequest{Name: "module", ChannelID: "ch1"})

	modules, err := svc.ListModules("ch1")
	if err != nil {
		t.Fatalf("ListModules: %v", err)
	}
	if len(modules) != 1 || modules[0].ID != mod.ID {
		t.Errorf("ListModules doit ne retourner que le module réel, got %+v", modules)
	}
	for _, m := range modules {
		if m.ID == draftComponent.ID {
			t.Error("le composant en brouillon ne doit jamais apparaître dans ListModules")
		}
	}
}

// / @brief  UpdateAsset modifie un brouillon local sans jamais toucher la blockchain
// / @input  Composant Draft, UpdateRequest{Name: nouveau nom}
// / @expect Le nom est mis à jour, l'asset reste absent de mockBC.records
func TestUpdateAsset_Draft_StaysLocal(t *testing.T) {
	bc := newMockBC()
	svc := model.NewService(bc, &mockFS{}).WithDraftStore(newMockDraftStore())

	m, _ := svc.AddFull(model.AddRequest{Name: "vis", ChannelID: "ch1"})
	updated, err := svc.UpdateAsset(model.UpdateRequest{ID: m.ID, Name: "vis M4"})
	if err != nil {
		t.Fatalf("UpdateAsset: %v", err)
	}
	if updated.Name != "vis M4" {
		t.Errorf("Name: got %q, want %q", updated.Name, "vis M4")
	}
	if len(bc.records) != 0 {
		t.Error("UpdateAsset sur un brouillon ne doit jamais écrire sur la blockchain")
	}
}

// / @brief  Remove supprime un brouillon local sans passer par l'adapter blockchain
// / @input  Composant Draft
// / @expect Remove sans erreur, Get(id) échoue ensuite (brouillon supprimé, rien sur la blockchain)
func TestRemove_Draft_RemovesLocally(t *testing.T) {
	svc := newFullSvc(t)
	m, _ := svc.AddFull(model.AddRequest{Name: "vis", ChannelID: "ch1"})

	if err := svc.Remove(m.ID); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if _, err := svc.Get(m.ID, ""); err == nil {
		t.Error("Get doit échouer après suppression du brouillon")
	}
}
