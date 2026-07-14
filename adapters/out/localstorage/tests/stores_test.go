// adapters/out/localstorage/tests/stores_test.go
// Tests d'intégration des adaptateurs localstorage : session, réseau, requêtes, canaux.
// Chaque test utilise un fichier temporaire isolé pour garantir l'indépendance.
package localstorage_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"myr/adapters/out/localstorage"
	"myr/domain/channel"
	"myr/domain/identity"
	"myr/domain/network"
	"myr/domain/session"
)

func tmpFile(t *testing.T, suffix string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "myr-test-*"+suffix)
	if err != nil {
		t.Fatalf("tmpFile: %v", err)
	}
	f.Close()
	os.Remove(f.Name()) // start empty
	return f.Name()
}

// ── JSONSessionStore ──────────────────────────────────────────────────────────

// / @brief  Vérifie que Load retourne nil sans erreur quand le fichier de session est absent
// / @input  JSONSessionStore pointant vers un fichier inexistant dans un répertoire temporaire
// / @expect Aucune erreur, valeur nil retournée
func TestSessionStore_LoadMissingFile_ReturnsNil(t *testing.T) {
	store := localstorage.NewJSONSessionStore(filepath.Join(t.TempDir(), "session.json"))
	sess, err := store.Load()
	if err != nil {
		t.Fatalf("Load on missing file: %v", err)
	}
	if sess != nil {
		t.Errorf("want nil, got %+v", sess)
	}
}

// / @brief  Vérifie que Save persiste une session et que Load la restitue fidèlement
// / @input  Session {Name:"alice", OrgID:"Org1MSP"} sauvegardée dans un fichier temporaire vide
// / @expect Load retourne une session avec les mêmes Name et OrgID, sans erreur
func TestSessionStore_SaveAndLoad(t *testing.T) {
	store := localstorage.NewJSONSessionStore(tmpFile(t, ".json"))
	want := &session.Session{Name: "alice", OrgID: "Org1MSP"}

	if err := store.Save(want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got == nil {
		t.Fatal("Load returned nil after Save")
	}
	if got.Name != want.Name || got.OrgID != want.OrgID {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

// / @brief  Vérifie que Save écrase la session précédente (comportement last-write-wins)
// / @input  Save de "alice/Org1MSP" puis Save de "bob/Org2MSP" sur le même fichier
// / @expect Load retourne la session "bob", sans erreur
func TestSessionStore_SaveOverwrites(t *testing.T) {
	path := tmpFile(t, ".json")
	store := localstorage.NewJSONSessionStore(path)

	store.Save(&session.Session{Name: "alice", OrgID: "Org1MSP"})
	store.Save(&session.Session{Name: "bob", OrgID: "Org2MSP"})

	got, _ := store.Load()
	if got.Name != "bob" {
		t.Errorf("expected overwrite: got %q, want bob", got.Name)
	}
}

// / @brief  Vérifie que Clear supprime la session sauvegardée et que Load retourne nil ensuite
// / @input  Session "alice" sauvegardée, puis Clear appelé
// / @expect Load retourne nil sans erreur après Clear
func TestSessionStore_Clear(t *testing.T) {
	path := tmpFile(t, ".json")
	store := localstorage.NewJSONSessionStore(path)

	store.Save(&session.Session{Name: "alice"})
	if err := store.Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load after Clear: %v", err)
	}
	if got != nil {
		t.Errorf("want nil after Clear, got %+v", got)
	}
}

// / @brief  Vérifie que Clear ne retourne pas d'erreur quand le fichier de session est absent
// / @input  JSONSessionStore pointant vers un fichier inexistant
// / @expect Aucune erreur retournée par Clear
func TestSessionStore_ClearMissingFile_NoError(t *testing.T) {
	store := localstorage.NewJSONSessionStore(filepath.Join(t.TempDir(), "absent.json"))
	if err := store.Clear(); err != nil {
		t.Errorf("Clear on missing file should not error: %v", err)
	}
}

// / @brief  Vérifie que les données de session sont relues correctement après rechargement du store
// / @input  Session {Name:"carol", OrgID:"OrgX"} sauvegardée, rechargement via un nouveau JSONSessionStore sur le même chemin
// / @expect Load retourne la session "carol" sans erreur
func TestSessionStore_Persistence(t *testing.T) {
	path := tmpFile(t, ".json")
	localstorage.NewJSONSessionStore(path).Save(&session.Session{Name: "carol", OrgID: "OrgX"})

	// Reload from same path
	got, err := localstorage.NewJSONSessionStore(path).Load()
	if err != nil || got == nil || got.Name != "carol" {
		t.Errorf("persistence failed: err=%v sess=%v", err, got)
	}
}

// ── JSONNetworkStore ──────────────────────────────────────────────────────────

// / @brief  Vérifie que FindAll retourne une liste vide sans erreur quand le fichier réseau est absent
// / @input  JSONNetworkStore pointant vers un fichier inexistant dans un répertoire temporaire
// / @expect Aucune erreur, slice vide retournée
func TestNetworkStore_FindAll_EmptyFile_ReturnsNil(t *testing.T) {
	store := localstorage.NewJSONNetworkStore(filepath.Join(t.TempDir(), "networks.json"))
	list, err := store.FindAll()
	if err != nil {
		t.Fatalf("FindAll on missing file: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("want 0, got %d", len(list))
	}
}

// / @brief  Vérifie que Save persiste un profil réseau et que FindAll le retourne
// / @input  Profil réseau {ID:"net-1", Name:"TestNet", PeerEndpoint:"localhost:7051"} sauvegardé
// / @expect FindAll retourne 1 élément avec ID="net-1", sans erreur
func TestNetworkStore_SaveAndFindAll(t *testing.T) {
	store := localstorage.NewJSONNetworkStore(tmpFile(t, ".json"))
	n := &network.NetworkProfile{ID: "net-1", Name: "TestNet", PeerEndpoint: "localhost:7051"}

	if err := store.Save(n); err != nil {
		t.Fatalf("Save: %v", err)
	}
	list, err := store.FindAll()
	if err != nil {
		t.Fatalf("FindAll: %v", err)
	}
	if len(list) != 1 || list[0].ID != "net-1" {
		t.Errorf("FindAll: got %v", list)
	}
}

// / @brief  Vérifie que Save met à jour un profil réseau existant portant le même ID (upsert)
// / @input  Profil "net-1" avec Name="Old" sauvegardé, puis re-sauvegardé avec Name="New"
// / @expect FindAll retourne 1 élément unique avec Name="New"
func TestNetworkStore_SaveUpdatesExisting(t *testing.T) {
	store := localstorage.NewJSONNetworkStore(tmpFile(t, ".json"))
	store.Save(&network.NetworkProfile{ID: "net-1", Name: "Old"})
	store.Save(&network.NetworkProfile{ID: "net-1", Name: "New"})

	list, _ := store.FindAll()
	if len(list) != 1 {
		t.Fatalf("expected 1 item after upsert, got %d", len(list))
	}
	if list[0].Name != "New" {
		t.Errorf("Name: got %q, want New", list[0].Name)
	}
}

// / @brief  Vérifie que plusieurs profils réseau distincts coexistent dans le store
// / @input  Profils "net-1" et "net-2" sauvegardés successivement
// / @expect FindAll retourne 2 éléments
func TestNetworkStore_SaveMultiple(t *testing.T) {
	store := localstorage.NewJSONNetworkStore(tmpFile(t, ".json"))
	store.Save(&network.NetworkProfile{ID: "net-1", Name: "Net1"})
	store.Save(&network.NetworkProfile{ID: "net-2", Name: "Net2"})

	list, _ := store.FindAll()
	if len(list) != 2 {
		t.Errorf("expected 2, got %d", len(list))
	}
}

// / @brief  Vérifie que Delete supprime le profil réseau ciblé sans affecter les autres
// / @input  Profils "net-1" et "net-2" sauvegardés, suppression de "net-1"
// / @expect FindAll retourne uniquement "net-2", sans erreur
func TestNetworkStore_Delete(t *testing.T) {
	store := localstorage.NewJSONNetworkStore(tmpFile(t, ".json"))
	store.Save(&network.NetworkProfile{ID: "net-1", Name: "Net1"})
	store.Save(&network.NetworkProfile{ID: "net-2", Name: "Net2"})

	if err := store.Delete("net-1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	list, _ := store.FindAll()
	if len(list) != 1 || list[0].ID != "net-2" {
		t.Errorf("after delete: %v", list)
	}
}

// / @brief  Vérifie que les profils réseau sont relus correctement après rechargement du store
// / @input  Profil {ID:"net-1", Name:"Persist"} sauvegardé, rechargement via un nouveau JSONNetworkStore
// / @expect FindAll retourne 1 élément avec Name="Persist", sans erreur
func TestNetworkStore_Persistence(t *testing.T) {
	path := tmpFile(t, ".json")
	localstorage.NewJSONNetworkStore(path).Save(&network.NetworkProfile{ID: "net-1", Name: "Persist"})

	list, err := localstorage.NewJSONNetworkStore(path).FindAll()
	if err != nil || len(list) != 1 || list[0].Name != "Persist" {
		t.Errorf("persistence failed: err=%v list=%v", err, list)
	}
}

// ── JSONRequestStore ──────────────────────────────────────────────────────────

// / @brief  Vérifie que FindAll retourne une liste vide sans erreur quand le fichier de requêtes est absent
// / @input  JSONRequestStore pointant vers un fichier inexistant dans un répertoire temporaire
// / @expect Aucune erreur, slice vide retournée
func TestRequestStore_FindAll_MissingFile_ReturnsNil(t *testing.T) {
	store := localstorage.NewJSONRequestStore(filepath.Join(t.TempDir(), "requests.json"))
	list, err := store.FindAll()
	if err != nil {
		t.Fatalf("FindAll on missing file: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("want 0, got %d", len(list))
	}
}

// / @brief  Vérifie que Save persiste une requête de compte et que FindAll la retourne
// / @input  AccountRequest {ID:"req-1", Pseudo:"alice", Status:Pending} sauvegardée
// / @expect FindAll retourne 1 élément avec ID="req-1", sans erreur
func TestRequestStore_SaveAndFindAll(t *testing.T) {
	store := localstorage.NewJSONRequestStore(tmpFile(t, ".json"))
	req := &identity.AccountRequest{ID: "req-1", Pseudo: "alice", Status: identity.RequestPending}

	if err := store.Save(req); err != nil {
		t.Fatalf("Save: %v", err)
	}
	list, err := store.FindAll()
	if err != nil {
		t.Fatalf("FindAll: %v", err)
	}
	if len(list) != 1 || list[0].ID != "req-1" {
		t.Errorf("FindAll: got %v", list)
	}
}

// / @brief  Vérifie que Save met à jour une requête existante par son ID (upsert sur le statut)
// / @input  Requête "req-1" avec Status=Pending sauvegardée, puis re-sauvegardée avec Status=Approved
// / @expect FindAll retourne 1 élément unique avec Status=Approved
func TestRequestStore_SaveUpdatesExistingByID(t *testing.T) {
	store := localstorage.NewJSONRequestStore(tmpFile(t, ".json"))
	store.Save(&identity.AccountRequest{ID: "req-1", Status: identity.RequestPending})
	store.Save(&identity.AccountRequest{ID: "req-1", Status: identity.RequestApproved})

	list, _ := store.FindAll()
	if len(list) != 1 {
		t.Fatalf("expected 1 item after upsert, got %d", len(list))
	}
	if list[0].Status != identity.RequestApproved {
		t.Errorf("Status: got %q, want approved", list[0].Status)
	}
}

// / @brief  Vérifie que plusieurs requêtes de compte distinctes coexistent dans le store
// / @input  Requêtes "req-1" (alice) et "req-2" (bob) sauvegardées successivement
// / @expect FindAll retourne 2 éléments
func TestRequestStore_SaveMultiple(t *testing.T) {
	store := localstorage.NewJSONRequestStore(tmpFile(t, ".json"))
	store.Save(&identity.AccountRequest{ID: "req-1", Pseudo: "alice"})
	store.Save(&identity.AccountRequest{ID: "req-2", Pseudo: "bob"})

	list, _ := store.FindAll()
	if len(list) != 2 {
		t.Errorf("expected 2, got %d", len(list))
	}
}

// / @brief  Vérifie que les requêtes de compte sont relues correctement après rechargement du store
// / @input  Requête {ID:"req-1", Pseudo:"carol"} sauvegardée, rechargement via un nouveau JSONRequestStore
// / @expect FindAll retourne 1 élément avec Pseudo="carol", sans erreur
func TestRequestStore_Persistence(t *testing.T) {
	path := tmpFile(t, ".json")
	localstorage.NewJSONRequestStore(path).Save(&identity.AccountRequest{ID: "req-1", Pseudo: "carol"})

	list, err := localstorage.NewJSONRequestStore(path).FindAll()
	if err != nil || len(list) != 1 || list[0].Pseudo != "carol" {
		t.Errorf("persistence failed: err=%v list=%v", err, list)
	}
}

// ── JSONChannelStore ──────────────────────────────────────────────────────────

// / @brief  Vérifie que FindAll retourne une liste vide sans erreur quand le fichier de canaux est absent
// / @input  JSONChannelStore pointant vers un fichier inexistant dans un répertoire temporaire
// / @expect Aucune erreur, slice vide retournée
func TestChannelStore_FindAll_MissingFile_ReturnsEmpty(t *testing.T) {
	store := localstorage.NewJSONChannelStore(filepath.Join(t.TempDir(), "channels.json"))
	list, err := store.FindAll()
	if err != nil {
		t.Fatalf("FindAll on missing file: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("want 0, got %d", len(list))
	}
}

// / @brief  Vérifie que FindAll désérialise correctement un fichier JSON contenant plusieurs canaux
// / @input  Fichier JSON préchargé avec deux canaux ch1 et ch2
// / @expect FindAll retourne 2 éléments, sans erreur
func TestChannelStore_FindAll_ReadsJSON(t *testing.T) {
	path := tmpFile(t, ".json")
	data, _ := json.Marshal(map[string]*channel.Channel{
		"ch1": {ID: "ch1", Name: "Channel 1"},
		"ch2": {ID: "ch2", Name: "Channel 2"},
	})
	os.WriteFile(path, data, 0644)

	store := localstorage.NewJSONChannelStore(path)
	list, err := store.FindAll()
	if err != nil {
		t.Fatalf("FindAll: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("want 2, got %d", len(list))
	}
}

// / @brief  Vérifie que FindByID retourne le canal correspondant à l'ID demandé
// / @input  Fichier JSON contenant le canal {ID:"ch1", Name:"Green"}, recherche par "ch1"
// / @expect Canal retourné avec Name="Green", sans erreur
func TestChannelStore_FindByID_Found(t *testing.T) {
	path := tmpFile(t, ".json")
	data, _ := json.Marshal(map[string]*channel.Channel{
		"ch1": {ID: "ch1", Name: "Green"},
	})
	os.WriteFile(path, data, 0644)

	store := localstorage.NewJSONChannelStore(path)
	ch, err := store.FindByID("ch1")
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if ch == nil || ch.Name != "Green" {
		t.Errorf("got %v, want Green", ch)
	}
}

// / @brief  Vérifie que FindByID retourne nil sans erreur pour un ID inconnu
// / @input  Fichier JSON contenant uniquement ch1, recherche par "nonexistent"
// / @expect nil retourné sans erreur
func TestChannelStore_FindByID_NotFound(t *testing.T) {
	path := tmpFile(t, ".json")
	data, _ := json.Marshal(map[string]*channel.Channel{
		"ch1": {ID: "ch1"},
	})
	os.WriteFile(path, data, 0644)

	store := localstorage.NewJSONChannelStore(path)
	ch, err := store.FindByID("nonexistent")
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if ch != nil {
		t.Errorf("want nil for unknown ID, got %+v", ch)
	}
}

// ── StaticChannelStore ────────────────────────────────────────────────────────

// / @brief  Vérifie que FindAll retourne tous les canaux fournis à la construction du store statique
// / @input  StaticChannelStore initialisé avec 3 canaux (ch1, ch2, ch3)
// / @expect FindAll retourne 3 éléments, sans erreur
func TestStaticChannelStore_FindAll(t *testing.T) {
	channels := []*channel.Channel{
		{ID: "ch1", Name: "Channel 1"},
		{ID: "ch2", Name: "Channel 2"},
		{ID: "ch3", Name: "Channel 3"},
	}
	store := localstorage.NewStaticChannelStore(channels)

	list, err := store.FindAll()
	if err != nil {
		t.Fatalf("FindAll: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("want 3, got %d", len(list))
	}
}

// / @brief  Vérifie que FindByID retourne le canal correspondant dans un store statique
// / @input  StaticChannelStore avec ch1 (Green) et ch2 (Blue), recherche par "ch2"
// / @expect Canal retourné avec Name="Blue", sans erreur
func TestStaticChannelStore_FindByID_Found(t *testing.T) {
	store := localstorage.NewStaticChannelStore([]*channel.Channel{
		{ID: "ch1", Name: "Green"},
		{ID: "ch2", Name: "Blue"},
	})

	ch, err := store.FindByID("ch2")
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if ch == nil || ch.Name != "Blue" {
		t.Errorf("got %v, want Blue", ch)
	}
}

// / @brief  Vérifie que FindByID retourne nil sans erreur pour un ID absent du store statique
// / @input  StaticChannelStore avec ch1 (Green), recherche par "nonexistent"
// / @expect nil retourné sans erreur
func TestStaticChannelStore_FindByID_NotFound(t *testing.T) {
	store := localstorage.NewStaticChannelStore([]*channel.Channel{
		{ID: "ch1", Name: "Green"},
	})

	ch, err := store.FindByID("nonexistent")
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if ch != nil {
		t.Errorf("want nil for unknown ID, got %+v", ch)
	}
}

// / @brief  Vérifie que FindAll retourne une liste vide quand le store statique est initialisé avec nil
// / @input  StaticChannelStore initialisé avec nil
// / @expect FindAll retourne une slice vide (longueur 0)
func TestStaticChannelStore_Empty(t *testing.T) {
	store := localstorage.NewStaticChannelStore(nil)
	list, _ := store.FindAll()
	if len(list) != 0 {
		t.Errorf("want 0, got %d", len(list))
	}
}
