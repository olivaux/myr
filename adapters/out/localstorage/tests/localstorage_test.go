package localstorage_test

import (
	"myr-core/adapters/out/localstorage"
	"os"
	"path/filepath"
	"testing"

	"myr-core/domain/model"
)

// ── helpers ──────────────────────────────────────────────────────────────────

func tmpDB(t *testing.T) (*localstorage.JSONBlockchain, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")
	return localstorage.NewJSONBlockchain(path), path
}

func tmpStorage(t *testing.T) (*localstorage.LocalStorage, string) {
	t.Helper()
	dir := t.TempDir()
	return localstorage.NewLocalStorage(dir), dir
}

// ── GetRefs ───────────────────────────────────────────────────────────────────

// / @brief  Vérifie que GetRefs retourne les valeurs par défaut quand le fichier JSON est absent
// / @input  JSONBlockchain pointant vers un fichier inexistant (répertoire temporaire vide)
// / @expect Aucune erreur retournée, liste de catégories non vide et conforme aux défauts du domaine
func TestGetRefs_EmptyFile_ReturnsDefaults(t *testing.T) {
	db, _ := tmpDB(t)
	refs, err := db.GetRefs()
	if err != nil {
		t.Fatalf("GetRefs: %v", err)
	}
	if len(refs.Categories) == 0 {
		t.Fatal("expected default categories, got empty")
	}
	defaults := model.DefaultInterfaceRefs()
	if len(refs.Categories) != len(defaults.Categories) {
		t.Fatalf("categories len: got %d, want %d", len(refs.Categories), len(defaults.Categories))
	}
}

// / @brief  Vérifie que GetRefs remplace une liste de catégories vide par les valeurs par défaut
// / @details Simule un fichier JSON valide mais avec categories:[] (bug pré-correction)
// / @input  Fichier JSON contenant refs.categories vide mais les clés types et units présentes
// / @expect Aucune erreur, categories remplies avec les défauts du domaine
func TestGetRefs_EmptyCategories_ReturnsDefaults(t *testing.T) {
	// Simulate a JSON file with refs present but Categories empty (pre-fix bug).
	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")
	if err := os.WriteFile(path, []byte(`{"thumbnails":{},"connections":[],"interfaces":[],"refs":{"categories":[],"types":{},"units":{}}}`), 0644); err != nil {
		t.Fatal(err)
	}
	db := localstorage.NewJSONBlockchain(path)
	refs, err := db.GetRefs()
	if err != nil {
		t.Fatalf("GetRefs: %v", err)
	}
	if len(refs.Categories) == 0 {
		t.Fatal("expected defaults when categories are empty, got none")
	}
}

// ── AddRefCategory ────────────────────────────────────────────────────────────

// / @brief  Vérifie qu'une nouvelle catégorie est bien ajoutée et récupérable via GetRefs
// / @input  Base vide, ajout de la catégorie "CUSTOM"
// / @expect GetRefs retourne une liste contenant "CUSTOM", sans erreur
func TestAddRefCategory(t *testing.T) {
	db, _ := tmpDB(t)
	if err := db.AddRefCategory("CUSTOM"); err != nil {
		t.Fatalf("AddRefCategory: %v", err)
	}
	refs, _ := db.GetRefs()
	found := false
	for _, c := range refs.Categories {
		if c == "CUSTOM" {
			found = true
		}
	}
	if !found {
		t.Fatal("category CUSTOM not found after add")
	}
}

// / @brief  Vérifie que l'ajout en double d'une catégorie ne crée pas de doublon
// / @input  Base vide, ajout deux fois de "CUSTOM"
// / @expect GetRefs retourne exactement un seul élément "CUSTOM"
func TestAddRefCategory_Dedup(t *testing.T) {
	db, _ := tmpDB(t)
	_ = db.AddRefCategory("CUSTOM")
	_ = db.AddRefCategory("CUSTOM")
	refs, _ := db.GetRefs()
	count := 0
	for _, c := range refs.Categories {
		if c == "CUSTOM" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected 1 CUSTOM, got %d", count)
	}
}

// / @brief  Vérifie qu'AddRefCategory initialise les maps Types et Units pour la nouvelle catégorie
// / @input  Base vide, ajout de la catégorie "PNEUM"
// / @expect refs.Types["PNEUM"] et refs.Units["PNEUM"] sont initialisés (non nil)
func TestAddRefCategory_InitializesTypesAndUnits(t *testing.T) {
	db, _ := tmpDB(t)
	_ = db.AddRefCategory("PNEUM")
	refs, _ := db.GetRefs()
	if refs.Types["PNEUM"] == nil {
		t.Fatal("expected Types[PNEUM] initialized")
	}
	if refs.Units["PNEUM"] == nil {
		t.Fatal("expected Units[PNEUM] initialized")
	}
}

// ── AddRefType ────────────────────────────────────────────────────────────────

// / @brief  Vérifie qu'un type peut être ajouté sous une catégorie existante
// / @input  Catégorie "ELEC" créée, ajout du type "SMA"
// / @expect refs.Types["ELEC"] contient "SMA", sans erreur
func TestAddRefType(t *testing.T) {
	db, _ := tmpDB(t)
	_ = db.AddRefCategory("ELEC")
	if err := db.AddRefType("ELEC", "SMA"); err != nil {
		t.Fatalf("AddRefType: %v", err)
	}
	refs, _ := db.GetRefs()
	found := false
	for _, tp := range refs.Types["ELEC"] {
		if tp == "SMA" {
			found = true
		}
	}
	if !found {
		t.Fatal("type SMA not found under ELEC")
	}
}

// / @brief  Vérifie que l'ajout en double d'un type sous une catégorie ne crée pas de doublon
// / @input  Ajout deux fois du type "SMA" sous "ELEC"
// / @expect refs.Types["ELEC"] contient exactement un seul "SMA"
func TestAddRefType_Dedup(t *testing.T) {
	db, _ := tmpDB(t)
	_ = db.AddRefType("ELEC", "SMA")
	_ = db.AddRefType("ELEC", "SMA")
	refs, _ := db.GetRefs()
	count := 0
	for _, tp := range refs.Types["ELEC"] {
		if tp == "SMA" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected 1 SMA, got %d", count)
	}
}

// ── AddRefUnit ────────────────────────────────────────────────────────────────

// / @brief  Vérifie qu'une unité peut être ajoutée sous une catégorie existante
// / @input  Catégorie "HYD" créée, ajout de l'unité "psi"
// / @expect refs.Units["HYD"] contient "psi", sans erreur
func TestAddRefUnit(t *testing.T) {
	db, _ := tmpDB(t)
	_ = db.AddRefCategory("HYD")
	if err := db.AddRefUnit("HYD", "psi"); err != nil {
		t.Fatalf("AddRefUnit: %v", err)
	}
	refs, _ := db.GetRefs()
	found := false
	for _, u := range refs.Units["HYD"] {
		if u == "psi" {
			found = true
		}
	}
	if !found {
		t.Fatal("unit psi not found under HYD")
	}
}

// / @brief  Vérifie que l'ajout en double d'une unité sous une catégorie ne crée pas de doublon
// / @input  Ajout deux fois de l'unité "psi" sous "HYD"
// / @expect refs.Units["HYD"] contient exactement un seul "psi"
func TestAddRefUnit_Dedup(t *testing.T) {
	db, _ := tmpDB(t)
	_ = db.AddRefUnit("HYD", "psi")
	_ = db.AddRefUnit("HYD", "psi")
	refs, _ := db.GetRefs()
	count := 0
	for _, u := range refs.Units["HYD"] {
		if u == "psi" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected 1 psi, got %d", count)
	}
}

// ── Refs persistence ──────────────────────────────────────────────────────────

// / @brief  Vérifie que catégories, types et unités sont bien persistés sur disque et rechargés
// / @input  Ajout de la catégorie "PNEUM", du type "Tube 4mm" et de l'unité "bar" ; rechargement via un nouveau JSONBlockchain sur le même chemin
// / @expect Le rechargement retrouve "PNEUM" dans les catégories, "Tube 4mm" dans Types["PNEUM"] et "bar" dans Units["PNEUM"]
func TestRefs_Persistence(t *testing.T) {
	db, path := tmpDB(t)
	_ = db.AddRefCategory("PNEUM")
	_ = db.AddRefType("PNEUM", "Tube 4mm")
	_ = db.AddRefUnit("PNEUM", "bar")

	// Reload from disk.
	db2 := localstorage.NewJSONBlockchain(path)
	refs, _ := db2.GetRefs()

	found := false
	for _, c := range refs.Categories {
		if c == "PNEUM" {
			found = true
		}
	}
	if !found {
		t.Fatal("PNEUM category not persisted")
	}
	foundType := false
	for _, tp := range refs.Types["PNEUM"] {
		if tp == "Tube 4mm" {
			foundType = true
		}
	}
	if !foundType {
		t.Fatal("type Tube 4mm not persisted")
	}
	foundUnit := false
	for _, u := range refs.Units["PNEUM"] {
		if u == "bar" {
			foundUnit = true
		}
	}
	if !foundUnit {
		t.Fatal("unit bar not persisted")
	}
}

// ── SaveInterface / GetInterface / ListInterfacesForAsset / RemoveInterface ───

func newIface(id, assetID string) *model.AssetInterface {
	return &model.AssetInterface{
		ID:        id,
		AssetID:   assetID,
		Name:      "test",
		Category:  "ELEC",
		Type:      "DIN",
		Direction: model.IfaceOut,
		ValueMin:  5.0,
		Unit:      "V",
	}
}

// / @brief  Vérifie que SaveInterface persiste une interface et que GetInterface la retourne correctement
// / @input  Interface avec ID "iface-1" et AssetID "asset-A" sauvegardée dans une base vide
// / @expect GetInterface("iface-1") retourne l'interface avec les mêmes ID et AssetID, sans erreur
func TestSaveInterface_And_GetInterface(t *testing.T) {
	db, _ := tmpDB(t)
	iface := newIface("iface-1", "asset-A")
	if err := db.SaveInterface(iface); err != nil {
		t.Fatalf("SaveInterface: %v", err)
	}
	got, err := db.GetInterface("iface-1")
	if err != nil {
		t.Fatalf("GetInterface: %v", err)
	}
	if got.ID != "iface-1" || got.AssetID != "asset-A" {
		t.Fatalf("unexpected interface: %+v", got)
	}
}

// / @brief  Vérifie que SaveInterface met à jour une interface existante (upsert)
// / @input  Interface "iface-1" sauvegardée, puis re-sauvegardée avec Name="updated"
// / @expect GetInterface retourne l'interface avec Name="updated"
func TestSaveInterface_Update(t *testing.T) {
	db, _ := tmpDB(t)
	iface := newIface("iface-1", "asset-A")
	_ = db.SaveInterface(iface)
	iface.Name = "updated"
	_ = db.SaveInterface(iface)
	got, _ := db.GetInterface("iface-1")
	if got.Name != "updated" {
		t.Fatalf("expected updated name, got %q", got.Name)
	}
}

// / @brief  Vérifie que ListInterfacesForAsset filtre les interfaces par AssetID
// / @input  Interfaces i1 et i2 liées à "asset-A", i3 liée à "asset-B"
// / @expect ListInterfacesForAsset("asset-A") retourne exactement 2 interfaces, sans erreur
func TestListInterfacesForAsset(t *testing.T) {
	db, _ := tmpDB(t)
	_ = db.SaveInterface(newIface("i1", "asset-A"))
	_ = db.SaveInterface(newIface("i2", "asset-A"))
	_ = db.SaveInterface(newIface("i3", "asset-B"))

	list, err := db.ListInterfacesForAsset("asset-A")
	if err != nil {
		t.Fatalf("ListInterfacesForAsset: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 interfaces for asset-A, got %d", len(list))
	}
}

// / @brief  Vérifie que RemoveInterface supprime l'interface demandée et la rend inaccessible
// / @input  Interfaces i1 et i2 liées à "asset-A", suppression de i1
// / @expect ListInterfacesForAsset retourne 1 interface ; GetInterface("i1") retourne une erreur
func TestRemoveInterface(t *testing.T) {
	db, _ := tmpDB(t)
	_ = db.SaveInterface(newIface("i1", "asset-A"))
	_ = db.SaveInterface(newIface("i2", "asset-A"))
	if err := db.RemoveInterface("i1"); err != nil {
		t.Fatalf("RemoveInterface: %v", err)
	}
	list, _ := db.ListInterfacesForAsset("asset-A")
	if len(list) != 1 {
		t.Fatalf("expected 1 interface after remove, got %d", len(list))
	}
	if _, err := db.GetInterface("i1"); err == nil {
		t.Fatal("expected error for removed interface")
	}
}

// / @brief  Vérifie que les interfaces sauvegardées sont relues correctement après rechargement du fichier
// / @input  Interface i1 liée à "asset-A" sauvegardée, rechargement via un nouveau JSONBlockchain
// / @expect GetInterface("i1") retourne l'interface avec AssetID="asset-A", sans erreur
func TestInterface_Persistence(t *testing.T) {
	db, path := tmpDB(t)
	_ = db.SaveInterface(newIface("i1", "asset-A"))

	db2 := localstorage.NewJSONBlockchain(path)
	got, err := db2.GetInterface("i1")
	if err != nil {
		t.Fatalf("interface not persisted: %v", err)
	}
	if got.AssetID != "asset-A" {
		t.Fatalf("wrong asset ID after reload: %q", got.AssetID)
	}
}

// ── SaveDraft / GetDraft / RemoveDraft / ListDrafts ───────────────────────────

func newDraft(id, channelID string) *model.Model3D {
	return &model.Model3D{ID: id, Name: "brouillon " + id, ChannelID: channelID, Status: model.ModuleDraft}
}

// / @brief  Vérifie que SaveDraft persiste un brouillon et que GetDraft le retrouve
// / @input  Brouillon "d1" sauvegardé dans une base vide
// / @expect GetDraft("d1") retourne le brouillon, sans erreur
func TestSaveDraft_And_GetDraft(t *testing.T) {
	db, _ := tmpDB(t)
	if err := db.SaveDraft(newDraft("d1", "ch1")); err != nil {
		t.Fatalf("SaveDraft: %v", err)
	}
	got, err := db.GetDraft("d1")
	if err != nil {
		t.Fatalf("GetDraft: %v", err)
	}
	if got.ID != "d1" || got.ChannelID != "ch1" {
		t.Fatalf("unexpected draft: %+v", got)
	}
}

// / @brief  Vérifie que SaveDraft met à jour un brouillon existant (upsert)
// / @input  Brouillon "d1" sauvegardé, puis re-sauvegardé avec Name modifié
// / @expect GetDraft retourne le brouillon avec le nom mis à jour
func TestSaveDraft_Update(t *testing.T) {
	db, _ := tmpDB(t)
	d := newDraft("d1", "ch1")
	_ = db.SaveDraft(d)
	d.Name = "vis M4"
	_ = db.SaveDraft(d)
	got, _ := db.GetDraft("d1")
	if got.Name != "vis M4" {
		t.Fatalf("expected updated name, got %q", got.Name)
	}
}

// / @brief  Vérifie que GetDraft retourne une erreur pour un brouillon inexistant
// / @input  Base vide
// / @expect GetDraft("absent") retourne une erreur
func TestGetDraft_NotFound(t *testing.T) {
	db, _ := tmpDB(t)
	if _, err := db.GetDraft("absent"); err == nil {
		t.Fatal("expected error for missing draft")
	}
}

// / @brief  Vérifie que ListDrafts filtre par channelID, et retourne tout si channelID=""
// / @input  Brouillons d1/d2 sur "ch1", d3 sur "ch2"
// / @expect ListDrafts("ch1") retourne 2 éléments ; ListDrafts("") retourne 3 éléments
func TestListDrafts_FiltersByChannel(t *testing.T) {
	db, _ := tmpDB(t)
	_ = db.SaveDraft(newDraft("d1", "ch1"))
	_ = db.SaveDraft(newDraft("d2", "ch1"))
	_ = db.SaveDraft(newDraft("d3", "ch2"))

	ch1, err := db.ListDrafts("ch1")
	if err != nil {
		t.Fatalf("ListDrafts(ch1): %v", err)
	}
	if len(ch1) != 2 {
		t.Fatalf("expected 2 drafts for ch1, got %d", len(ch1))
	}
	all, err := db.ListDrafts("")
	if err != nil {
		t.Fatalf("ListDrafts(\"\"): %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3 drafts total, got %d", len(all))
	}
}

// / @brief  Vérifie que RemoveDraft supprime le brouillon et le rend inaccessible
// / @input  Brouillons d1 et d2, suppression de d1
// / @expect GetDraft("d1") retourne une erreur ; ListDrafts("") retourne 1 élément
func TestRemoveDraft(t *testing.T) {
	db, _ := tmpDB(t)
	_ = db.SaveDraft(newDraft("d1", "ch1"))
	_ = db.SaveDraft(newDraft("d2", "ch1"))
	if err := db.RemoveDraft("d1"); err != nil {
		t.Fatalf("RemoveDraft: %v", err)
	}
	if _, err := db.GetDraft("d1"); err == nil {
		t.Fatal("expected error for removed draft")
	}
	all, _ := db.ListDrafts("")
	if len(all) != 1 {
		t.Fatalf("expected 1 draft after remove, got %d", len(all))
	}
}

// / @brief  Vérifie que les brouillons sauvegardés sont relus correctement après rechargement du fichier
// / @input  Brouillon d1 sauvegardé, rechargement via un nouveau JSONBlockchain
// / @expect GetDraft("d1") retourne le brouillon avec ChannelID="ch1", sans erreur
func TestDraft_Persistence(t *testing.T) {
	db, path := tmpDB(t)
	_ = db.SaveDraft(newDraft("d1", "ch1"))

	db2 := localstorage.NewJSONBlockchain(path)
	got, err := db2.GetDraft("d1")
	if err != nil {
		t.Fatalf("draft not persisted: %v", err)
	}
	if got.ChannelID != "ch1" {
		t.Fatalf("wrong channel ID after reload: %q", got.ChannelID)
	}
}

// ── SaveConnection / ListConnections / UpdateConnection / RemoveConnection ────

func newConn(id, from, to string) *model.Connection {
	return &model.Connection{ID: id, From: from, To: to, Label: "test"}
}

// / @brief  Vérifie que SaveConnection persiste une connexion et que ListConnections la retourne
// / @input  Connexion c1 (A→B) sauvegardée dans une base vide
// / @expect ListConnections retourne une liste de 1 élément avec ID="c1", sans erreur
func TestSaveConnection_And_List(t *testing.T) {
	db, _ := tmpDB(t)
	if err := db.SaveConnection(newConn("c1", "A", "B")); err != nil {
		t.Fatalf("SaveConnection: %v", err)
	}
	list, err := db.ListConnections()
	if err != nil {
		t.Fatalf("ListConnections: %v", err)
	}
	if len(list) != 1 || list[0].ID != "c1" {
		t.Fatalf("unexpected list: %+v", list)
	}
}

// / @brief  Vérifie la déduplication des connexions simples ayant le même From/To sans identifiants d'interface
// / @input  Connexions c1 et c2 avec mêmes From="A" et To="B", sans FromIfaceID ni ToIfaceID
// / @expect ListConnections retourne exactement 1 connexion (la première est conservée)
func TestSaveConnection_Dedup_SimpleLink(t *testing.T) {
	db, _ := tmpDB(t)
	_ = db.SaveConnection(newConn("c1", "A", "B"))
	_ = db.SaveConnection(newConn("c2", "A", "B")) // same From/To, no iface IDs
	list, _ := db.ListConnections()
	if len(list) != 1 {
		t.Fatalf("expected dedup to 1, got %d", len(list))
	}
}

// / @brief  Vérifie la déduplication des connexions d'interface ayant les mêmes From, To, FromIfaceID, ToIfaceID et InstanceIDs
// / @input  Connexions c1 et c2 avec mêmes endpoints d'interface (if1→if2, inst1→inst2)
// / @expect ListConnections retourne exactement 1 connexion
func TestSaveConnection_Dedup_IfaceLink(t *testing.T) {
	db, _ := tmpDB(t)
	c1 := &model.Connection{ID: "c1", From: "A", To: "B", FromIfaceID: "if1", ToIfaceID: "if2", FromInstanceID: "inst1", ToInstanceID: "inst2"}
	c2 := &model.Connection{ID: "c2", From: "A", To: "B", FromIfaceID: "if1", ToIfaceID: "if2", FromInstanceID: "inst1", ToInstanceID: "inst2"}
	_ = db.SaveConnection(c1)
	_ = db.SaveConnection(c2)
	list, _ := db.ListConnections()
	if len(list) != 1 {
		t.Fatalf("expected dedup to 1 for same iface link, got %d", len(list))
	}
}

// / @brief  Vérifie qu'UpdateConnection modifie le label d'une connexion existante
// / @input  Connexion c1 (A→B, label="test") sauvegardée, puis mise à jour avec label="updated"
// / @expect ListConnections retourne c1 avec Label="updated", sans erreur
func TestUpdateConnection(t *testing.T) {
	db, _ := tmpDB(t)
	_ = db.SaveConnection(newConn("c1", "A", "B"))
	updated := &model.Connection{ID: "c1", From: "A", To: "B", Label: "updated"}
	if err := db.UpdateConnection(updated); err != nil {
		t.Fatalf("UpdateConnection: %v", err)
	}
	list, _ := db.ListConnections()
	if list[0].Label != "updated" {
		t.Fatalf("expected label updated, got %q", list[0].Label)
	}
}

// / @brief  Vérifie qu'UpdateConnection retourne une erreur si la connexion n'existe pas
// / @input  Base vide, tentative de mise à jour de la connexion "missing"
// / @expect Erreur non nil retournée
func TestUpdateConnection_NotFound(t *testing.T) {
	db, _ := tmpDB(t)
	err := db.UpdateConnection(newConn("missing", "A", "B"))
	if err == nil {
		t.Fatal("expected error for missing connection")
	}
}

// / @brief  Vérifie que RemoveConnection supprime la connexion ciblée sans affecter les autres
// / @input  Connexions c1 (A→B) et c2 (B→C) sauvegardées, suppression de c1
// / @expect ListConnections retourne uniquement c2, sans erreur
func TestRemoveConnection(t *testing.T) {
	db, _ := tmpDB(t)
	_ = db.SaveConnection(newConn("c1", "A", "B"))
	_ = db.SaveConnection(newConn("c2", "B", "C"))
	if err := db.RemoveConnection("c1"); err != nil {
		t.Fatalf("RemoveConnection: %v", err)
	}
	list, _ := db.ListConnections()
	if len(list) != 1 || list[0].ID != "c2" {
		t.Fatalf("unexpected list after remove: %+v", list)
	}
}

// / @brief  Vérifie que les connexions sauvegardées sont relues correctement après rechargement du fichier
// / @input  Connexion c1 (A→B) sauvegardée, rechargement via un nouveau JSONBlockchain sur le même chemin
// / @expect ListConnections retourne 1 élément avec ID="c1", sans erreur
func TestConnection_Persistence(t *testing.T) {
	db, path := tmpDB(t)
	_ = db.SaveConnection(newConn("c1", "A", "B"))

	db2 := localstorage.NewJSONBlockchain(path)
	list, err := db2.ListConnections()
	if err != nil {
		t.Fatalf("ListConnections after reload: %v", err)
	}
	if len(list) != 1 || list[0].ID != "c1" {
		t.Fatalf("connection not persisted")
	}
}

// ── SaveThumbnail / GetThumbnail ──────────────────────────────────────────────

// / @brief  Vérifie que SaveThumbnail persiste une miniature et que GetThumbnail la retourne fidèlement
// / @input  Miniature "data:image/png;base64,abc123" associée à "asset-1"
// / @expect GetThumbnail("asset-1") retourne exactement "data:image/png;base64,abc123", sans erreur
func TestSaveThumbnail_And_Get(t *testing.T) {
	db, _ := tmpDB(t)
	if err := db.SaveThumbnail("asset-1", "data:image/png;base64,abc123"); err != nil {
		t.Fatalf("SaveThumbnail: %v", err)
	}
	got, err := db.GetThumbnail("asset-1")
	if err != nil {
		t.Fatalf("GetThumbnail: %v", err)
	}
	if got != "data:image/png;base64,abc123" {
		t.Fatalf("unexpected thumbnail: %q", got)
	}
}

// / @brief  Vérifie que GetThumbnail retourne une chaîne vide sans erreur pour un asset inconnu
// / @input  Base vide, appel de GetThumbnail avec l'ID "nonexistent"
// / @expect Aucune erreur, chaîne vide retournée
func TestGetThumbnail_Missing_ReturnsEmpty(t *testing.T) {
	db, _ := tmpDB(t)
	got, err := db.GetThumbnail("nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

// / @brief  Vérifie que SaveThumbnail écrase la miniature précédente pour le même asset
// / @input  Miniature "first" puis "second" sauvegardées pour "asset-1"
// / @expect GetThumbnail retourne "second"
func TestThumbnail_Overwrite(t *testing.T) {
	db, _ := tmpDB(t)
	_ = db.SaveThumbnail("asset-1", "first")
	_ = db.SaveThumbnail("asset-1", "second")
	got, _ := db.GetThumbnail("asset-1")
	if got != "second" {
		t.Fatalf("expected overwrite to second, got %q", got)
	}
}

// / @brief  Vérifie que les miniatures sauvegardées sont relues correctement après rechargement du fichier
// / @input  Miniature "data:image/png;base64,xyz" pour "asset-1" sauvegardée, rechargement via un nouveau JSONBlockchain
// / @expect GetThumbnail("asset-1") retourne "data:image/png;base64,xyz"
func TestThumbnail_Persistence(t *testing.T) {
	db, path := tmpDB(t)
	_ = db.SaveThumbnail("asset-1", "data:image/png;base64,xyz")

	db2 := localstorage.NewJSONBlockchain(path)
	got, _ := db2.GetThumbnail("asset-1")
	if got != "data:image/png;base64,xyz" {
		t.Fatalf("thumbnail not persisted, got %q", got)
	}
}

// ── LocalStorage ──────────────────────────────────────────────────────────────

// / @brief  Vérifie qu'Upload copie un fichier source dans le répertoire de stockage et retourne le chemin destination
// / @input  Fichier source "model.glb" contenant "fake-glb-data" dans un répertoire temporaire
// / @expect Chemin destination non vide, contenu du fichier copié identique à la source, sans erreur
func TestLocalStorage_Upload(t *testing.T) {
	ls, dir := tmpStorage(t)

	// Create a source file.
	src := filepath.Join(t.TempDir(), "model.glb")
	if err := os.WriteFile(src, []byte("fake-glb-data"), 0644); err != nil {
		t.Fatal(err)
	}

	dest, err := ls.Upload(src)
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if dest == "" {
		t.Fatal("expected non-empty dest path")
	}
	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("reading uploaded file: %v", err)
	}
	if string(data) != "fake-glb-data" {
		t.Fatalf("unexpected content: %q", string(data))
	}
	_ = dir
}

// / @brief  Vérifie qu'Upload retourne une erreur si le fichier source n'existe pas
// / @input  Chemin source inexistant "/nonexistent/path/model.glb"
// / @expect Erreur non nil retournée
func TestLocalStorage_Upload_MissingSource(t *testing.T) {
	ls, _ := tmpStorage(t)
	_, err := ls.Upload("/nonexistent/path/model.glb")
	if err == nil {
		t.Fatal("expected error for missing source file")
	}
}

// / @brief  Vérifie que Download copie un fichier depuis le stockage vers un chemin destination
// / @input  Fichier "file.glb" contenant "content" déposé directement dans le répertoire de stockage
// / @expect Fichier destination créé avec le même contenu "content", sans erreur
func TestLocalStorage_Download(t *testing.T) {
	ls, dir := tmpStorage(t)

	// Put a file in the storage dir directly.
	stored := filepath.Join(dir, "file.glb")
	if err := os.WriteFile(stored, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	dest := filepath.Join(t.TempDir(), "out.glb")
	if err := ls.Download(stored, dest); err != nil {
		t.Fatalf("Download: %v", err)
	}
	data, _ := os.ReadFile(dest)
	if string(data) != "content" {
		t.Fatalf("unexpected content: %q", string(data))
	}
}

// / @brief  Vérifie que Delete supprime un fichier présent dans le répertoire de stockage
// / @input  Fichier "file.glb" déposé directement dans le répertoire de stockage
// / @expect Le fichier n'existe plus après Delete, sans erreur
func TestLocalStorage_Delete(t *testing.T) {
	ls, dir := tmpStorage(t)

	stored := filepath.Join(dir, "file.glb")
	if err := os.WriteFile(stored, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := ls.Delete(stored); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := os.Stat(stored); !os.IsNotExist(err) {
		t.Fatal("file still exists after Delete")
	}
}

// / @brief  Vérifie le cycle complet Upload puis Delete d'un fichier dans le stockage local
// / @input  Fichier source "model.glb" uploadé via Upload, puis chemin destination passé à Delete
// / @expect Le fichier destination n'existe plus après Delete, aucune erreur dans les deux opérations
func TestLocalStorage_Upload_Then_Delete(t *testing.T) {
	ls, _ := tmpStorage(t)

	src := filepath.Join(t.TempDir(), "model.glb")
	_ = os.WriteFile(src, []byte("data"), 0644)

	dest, err := ls.Upload(src)
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if err := ls.Delete(dest); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatal("file still exists after Delete")
	}
}
