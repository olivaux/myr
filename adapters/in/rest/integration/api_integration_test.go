// adapters/in/rest/integration/api_integration_test.go
//
// Tests d'intégration end-to-end de l'API REST.
// Aucun mock : domaine réel + localstorage réel + serveur HTTP réel (httptest.Server).
//
// Lancer avec :
//   go test -tags integration -v -timeout 60s ./adapters/in/rest/integration/...

//go:build integration

package rest_integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"myr/adapters/in/rest"
	"myr/adapters/out/localstorage"
	"myr/domain/model"
)

// ── In-memory blockchain (remplace Fabric en test d'intégration) ──────────────
// Implémente BlockchainPort + le port de suppression optionnel.

type memBC struct {
	mu      sync.Mutex
	records map[string]*model.Model3D
}

func newMemBC() *memBC { return &memBC{records: make(map[string]*model.Model3D)} }

func (b *memBC) StoreModelRecord(m *model.Model3D) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	cp := *m
	b.records[m.ID] = &cp
	return nil
}

func (b *memBC) GetModelRecord(id, _ string) (*model.Model3D, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if m, ok := b.records[id]; ok {
		cp := *m
		return &cp, nil
	}
	return nil, fmt.Errorf("modèle introuvable : %s", id)
}

func (b *memBC) ListModelRecords(channelID string) ([]*model.Model3D, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	var list []*model.Model3D
	for _, m := range b.records {
		if channelID == "" || m.ChannelID == channelID {
			cp := *m
			list = append(list, &cp)
		}
	}
	return list, nil
}

func (b *memBC) VerifyIntegrity(id, hash, _ string) (bool, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	m, ok := b.records[id]
	if !ok {
		return false, fmt.Errorf("modèle introuvable : %s", id)
	}
	return m.Hash == hash, nil
}

func (b *memBC) RemoveModelRecord(id string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.records[id]; !ok {
		return fmt.Errorf("modèle introuvable : %s", id)
	}
	delete(b.records, id)
	return nil
}

// ── Helpers ────────────────────────────────────────────────────────────────────

// newIntegrationServer câble un serveur HTTP complet avec des adapters réels.
func newIntegrationServer(t *testing.T) *httptest.Server {
	t.Helper()
	dataDir := t.TempDir()
	filesDir := filepath.Join(dataDir, "files")
	os.MkdirAll(filesDir, 0755)

	bc := newMemBC()
	fs := localstorage.NewLocalStorage(filesDir)
	db := localstorage.NewJSONBlockchain(filepath.Join(dataDir, "db.json"))

	svc := model.NewService(bc, fs).
		WithConnStore(db).
		WithThumbStore(db).
		WithIfaceStore(db)

	h := rest.NewHandler(svc, db, db)
	srv := rest.NewServer(h, "localhost:0")
	mux, err := srv.Handler()
	if err != nil {
		t.Fatalf("Handler(): %v", err)
	}
	return httptest.NewServer(mux)
}

// apiDo envoie une requête HTTP et retourne la réponse.
func apiDo(t *testing.T, client *http.Client, method, url, body string) *http.Response {
	t.Helper()
	var req *http.Request
	var err error
	if body != "" {
		req, err = http.NewRequest(method, url, bytes.NewBufferString(body))
		if err != nil {
			t.Fatalf("NewRequest: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest(method, url, nil)
		if err != nil {
			t.Fatalf("NewRequest: %v", err)
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Do %s %s: %v", method, url, err)
	}
	return resp
}

// decodeBody décode le JSON de la réponse dans v.
func decodeBody(t *testing.T, resp *http.Response, v any) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		t.Fatalf("decode body: %v", err)
	}
}

// multipartBody construit un corps multipart minimal pour les tests de composants.
func multipartBody(name string) (string, string) {
	boundary := "testboundary"
	body := fmt.Sprintf("--%s\r\nContent-Disposition: form-data; name=\"name\"\r\n\r\n%s\r\n--%s--\r\n",
		boundary, name, boundary)
	ct := "multipart/form-data; boundary=" + boundary
	return body, ct
}

// ── Tests de base ─────────────────────────────────────────────────────────────

/// @brief  GET /api/ping retourne 200 avec ok=true (serveur réel)
func TestAPIIntegration_Ping(t *testing.T) {
	srv := newIntegrationServer(t)
	defer srv.Close()

	resp, err := srv.Client().Get(srv.URL + "/api/ping")
	if err != nil {
		t.Fatalf("GET /api/ping: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("got %d, want 200", resp.StatusCode)
	}
	var result struct{ Ok bool `json:"ok"` }
	json.NewDecoder(resp.Body).Decode(&result)
	if !result.Ok {
		t.Error("ok doit être true")
	}
}

/// @brief  GET /api/status retourne le mode et le nombre d'assets
func TestAPIIntegration_Status(t *testing.T) {
	srv := newIntegrationServer(t)
	defer srv.Close()

	resp, err := srv.Client().Get(srv.URL + "/api/status")
	if err != nil {
		t.Fatalf("GET /api/status: %v", err)
	}
	var status struct {
		Mode   string `json:"mode"`
		Assets int    `json:"assets"`
	}
	decodeBody(t, resp, &status)
	if status.Mode == "" {
		t.Error("mode ne doit pas être vide")
	}
}

/// @brief  GET /api/licenses retourne le catalogue complet (MIT présent)
func TestAPIIntegration_LicenseCatalog(t *testing.T) {
	srv := newIntegrationServer(t)
	defer srv.Close()

	resp, err := srv.Client().Get(srv.URL + "/api/licenses")
	if err != nil {
		t.Fatalf("GET /api/licenses: %v", err)
	}
	var licenses []struct{ ID string `json:"id"` }
	decodeBody(t, resp, &licenses)
	if len(licenses) == 0 {
		t.Fatal("catalogue de licences vide")
	}
	found := false
	for _, l := range licenses {
		if l.ID == "mit" {
			found = true
		}
	}
	if !found {
		t.Error("licence MIT absente du catalogue")
	}
}

// ── Cycle de vie d'un composant ───────────────────────────────────────────────

/// @brief  Cycle complet : POST → GET → PATCH → DELETE /api/components avec vrais adapters
/// @input  Domaine réel + LocalStorage réel + JSONBlockchain réel
/// @expect Chaque étape retourne le code HTTP attendu ; le nom est bien mis à jour ; 404 après suppression
func TestAPIIntegration_AssetLifecycle(t *testing.T) {
	srv := newIntegrationServer(t)
	defer srv.Close()
	client := srv.Client()

	// Créer
	body, ct := multipartBody("Vis M3")
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/components", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", ct)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("POST /api/components: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		resp.Body.Close()
		t.Fatalf("création: got %d, want 201", resp.StatusCode)
	}
	var created struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	decodeBody(t, resp, &created)
	if created.ID == "" {
		t.Fatal("ID vide dans la réponse de création")
	}
	if created.Name != "Vis M3" {
		t.Errorf("name: got %q, want Vis M3", created.Name)
	}

	// Lire
	resp2 := apiDo(t, client, http.MethodGet, srv.URL+"/api/components/"+created.ID, "")
	if resp2.StatusCode != http.StatusOK {
		resp2.Body.Close()
		t.Fatalf("lecture: got %d, want 200", resp2.StatusCode)
	}
	var got struct{ Name string `json:"name"` }
	decodeBody(t, resp2, &got)
	if got.Name != "Vis M3" {
		t.Errorf("name lu: got %q, want Vis M3", got.Name)
	}

	// Vérifier dans la liste
	resp3 := apiDo(t, client, http.MethodGet, srv.URL+"/api/components", "")
	var list struct{ Total int `json:"total"` }
	decodeBody(t, resp3, &list)
	if list.Total < 1 {
		t.Errorf("total: got %d, want ≥1", list.Total)
	}

	// Mettre à jour
	resp4 := apiDo(t, client, http.MethodPatch, srv.URL+"/api/components/"+created.ID, `{"name":"Vis M4"}`)
	if resp4.StatusCode != http.StatusOK {
		resp4.Body.Close()
		t.Errorf("PATCH: got %d, want 200", resp4.StatusCode)
	} else {
		var updated struct{ Name string `json:"name"` }
		decodeBody(t, resp4, &updated)
		if updated.Name != "Vis M4" {
			t.Errorf("name après PATCH: got %q, want Vis M4", updated.Name)
		}
	}

	// Supprimer
	resp5 := apiDo(t, client, http.MethodDelete, srv.URL+"/api/components/"+created.ID, "")
	resp5.Body.Close()
	if resp5.StatusCode != http.StatusNoContent {
		t.Errorf("DELETE: got %d, want 204", resp5.StatusCode)
	}

	// Vérifier la suppression
	resp6 := apiDo(t, client, http.MethodGet, srv.URL+"/api/components/"+created.ID, "")
	resp6.Body.Close()
	if resp6.StatusCode != http.StatusNotFound {
		t.Errorf("GET après DELETE: got %d, want 404", resp6.StatusCode)
	}
}

// ── Connexions ────────────────────────────────────────────────────────────────

/// @brief  Cycle complet : POST → DELETE /api/connections avec JSONBlockchain réel
/// @input  Connexion from/to persistée en JSON
/// @expect connexion créée avec ID ; supprimée en 204 ; persiste entre les requêtes
func TestAPIIntegration_ConnectionLifecycle(t *testing.T) {
	srv := newIntegrationServer(t)
	defer srv.Close()
	client := srv.Client()

	resp := apiDo(t, client, http.MethodPost, srv.URL+"/api/connections",
		`{"from":"asset-A","to":"asset-B","label":"assemblage"}`)
	if resp.StatusCode != http.StatusCreated {
		resp.Body.Close()
		t.Fatalf("POST connexion: got %d, want 201", resp.StatusCode)
	}
	var conn struct {
		ID    string `json:"id"`
		From  string `json:"from"`
		To    string `json:"to"`
		Label string `json:"label"`
	}
	decodeBody(t, resp, &conn)
	if conn.ID == "" {
		t.Fatal("connexion ID vide")
	}
	if conn.From != "asset-A" || conn.To != "asset-B" {
		t.Errorf("from/to: got %q/%q", conn.From, conn.To)
	}

	resp2 := apiDo(t, client, http.MethodDelete, srv.URL+"/api/connections/"+conn.ID, "")
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusNoContent {
		t.Errorf("DELETE connexion: got %d, want 204", resp2.StatusCode)
	}
}

// ── Modules ────────────────────────────────────────────────────────────────────

/// @brief  Cycle complet : créer module → ajouter assemblage → soumettre
/// @input  Module + connexion réels, soumission blockchain simulée (memBC)
/// @expect statut "draft" à la création, "submitted" après SubmitModule
func TestAPIIntegration_ModuleLifecycle(t *testing.T) {
	srv := newIntegrationServer(t)
	defer srv.Close()
	client := srv.Client()

	// Créer le module
	resp := apiDo(t, client, http.MethodPost, srv.URL+"/api/modules",
		`{"name":"Drone v1","owner_id":"user1"}`)
	if resp.StatusCode != http.StatusCreated {
		resp.Body.Close()
		t.Fatalf("POST module: got %d, want 201", resp.StatusCode)
	}
	var mod struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	decodeBody(t, resp, &mod)
	if mod.Status != string(model.ModuleDraft) {
		t.Errorf("status initial: got %q, want draft", mod.Status)
	}

	// Créer une connexion
	resp2 := apiDo(t, client, http.MethodPost, srv.URL+"/api/connections",
		`{"from":"a1","to":"a2","label":"lien"}`)
	var conn struct{ ID string `json:"id"` }
	decodeBody(t, resp2, &conn)

	// Ajouter la connexion comme assemblage
	resp3 := apiDo(t, client, http.MethodPost, srv.URL+"/api/modules/"+mod.ID+"/assemblies",
		`{"connection_id":"`+conn.ID+`"}`)
	if resp3.StatusCode != http.StatusOK {
		resp3.Body.Close()
		t.Fatalf("POST assemblies: got %d, want 200", resp3.StatusCode)
	}
	resp3.Body.Close()

	// Soumettre
	resp4 := apiDo(t, client, http.MethodPost, srv.URL+"/api/modules/"+mod.ID+"/submit",
		`{"note":"première livraison"}`)
	if resp4.StatusCode != http.StatusOK {
		resp4.Body.Close()
		t.Fatalf("POST submit: got %d, want 200", resp4.StatusCode)
	}
	var submitted struct{ Status string `json:"status"` }
	decodeBody(t, resp4, &submitted)
	if submitted.Status != string(model.ModuleSubmitted) {
		t.Errorf("status après submit: got %q, want submitted", submitted.Status)
	}
}

// ── Interfaces physiques ──────────────────────────────────────────────────────

/// @brief  Cycle complet : ajouter interface → lister → supprimer (JSONBlockchain réel)
/// @input  Composant créé ; interface ELEC USB-C Out ajoutée et supprimée
/// @expect interface présente dans la liste après ajout ; absente après suppression
func TestAPIIntegration_InterfaceLifecycle(t *testing.T) {
	srv := newIntegrationServer(t)
	defer srv.Close()
	client := srv.Client()

	// Créer un composant
	body, ct := multipartBody("Capteur")
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/components", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", ct)
	resp, _ := client.Do(req)
	var asset struct{ ID string `json:"id"` }
	decodeBody(t, resp, &asset)

	// Ajouter une interface
	resp2 := apiDo(t, client, http.MethodPost, srv.URL+"/api/components/"+asset.ID+"/interfaces",
		`{"category":"ELEC","type":"USB-C","direction":"out"}`)
	if resp2.StatusCode != http.StatusCreated {
		resp2.Body.Close()
		t.Fatalf("POST interface: got %d, want 201", resp2.StatusCode)
	}
	var iface struct{ ID string `json:"id"` }
	decodeBody(t, resp2, &iface)
	if iface.ID == "" {
		t.Fatal("interface ID vide")
	}

	// Lister
	resp3 := apiDo(t, client, http.MethodGet, srv.URL+"/api/components/"+asset.ID+"/interfaces", "")
	var ifaces []struct{ ID string `json:"id"` }
	decodeBody(t, resp3, &ifaces)
	if len(ifaces) != 1 {
		t.Errorf("got %d interfaces, want 1", len(ifaces))
	}

	// Supprimer
	resp4 := apiDo(t, client, http.MethodDelete, srv.URL+"/api/interfaces/"+iface.ID, "")
	resp4.Body.Close()
	if resp4.StatusCode != http.StatusNoContent {
		t.Errorf("DELETE interface: got %d, want 204", resp4.StatusCode)
	}

	// Vérifier l'absence
	resp5 := apiDo(t, client, http.MethodGet, srv.URL+"/api/components/"+asset.ID+"/interfaces", "")
	var empty []struct{ ID string `json:"id"` }
	decodeBody(t, resp5, &empty)
	if len(empty) != 0 {
		t.Errorf("interface toujours présente après suppression : got %d", len(empty))
	}
}

// ── Persistance JSON ──────────────────────────────────────────────────────────

/// @brief  Les connexions persistent sur disque entre deux instances du service
/// @input  Connexion créée via la 1re instance ; relue via une 2e instance pointant sur le même fichier
/// @expect connexion retrouvée après rechargement du JSONBlockchain depuis disque
func TestAPIIntegration_ConnectionPersistsOnDisk(t *testing.T) {
	dataDir := t.TempDir()
	dbPath := filepath.Join(dataDir, "db.json")

	// Instance 1 : créer une connexion
	func() {
		db := localstorage.NewJSONBlockchain(dbPath)
		if err := db.SaveConnection(&model.Connection{
			ID: "c-persist", From: "asset-X", To: "asset-Y", Label: "test",
		}); err != nil {
			t.Fatalf("SaveConnection: %v", err)
		}
	}()

	// Instance 2 : recharger depuis disque et vérifier
	db2 := localstorage.NewJSONBlockchain(dbPath)
	conns, err := db2.ListConnections()
	if err != nil {
		t.Fatalf("ListConnections: %v", err)
	}
	if len(conns) != 1 {
		t.Fatalf("got %d connexions, want 1", len(conns))
	}
	if conns[0].ID != "c-persist" {
		t.Errorf("ID: got %q, want c-persist", conns[0].ID)
	}
}

// ── Requêtes concurrentes ─────────────────────────────────────────────────────

/// @brief  N assets créés en parallèle — aucune course dans le handler HTTP
/// @input  10 goroutines POST simultanées sur le serveur httptest
/// @expect 10 codes 201 ; total=10 dans la liste finale
func TestAPIIntegration_ConcurrentAssetCreation(t *testing.T) {
	srv := newIntegrationServer(t)
	defer srv.Close()
	client := srv.Client()

	const n = 10
	results := make(chan int, n)

	for i := 0; i < n; i++ {
		go func(idx int) {
			body, ct := multipartBody(fmt.Sprintf("Asset%02d", idx))
			req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/components", bytes.NewBufferString(body))
			req.Header.Set("Content-Type", ct)
			resp, err := client.Do(req)
			if err != nil {
				results <- 0
				return
			}
			resp.Body.Close()
			results <- resp.StatusCode
		}(i)
	}

	for i := 0; i < n; i++ {
		code := <-results
		if code != http.StatusCreated {
			t.Errorf("goroutine %d: got %d, want 201", i, code)
		}
	}

	resp := apiDo(t, client, http.MethodGet, srv.URL+"/api/components", "")
	var list struct{ Total int `json:"total"` }
	decodeBody(t, resp, &list)
	if list.Total != n {
		t.Errorf("total après %d créations parallèles: got %d, want %d", n, list.Total, n)
	}
}

// ── Vocabulaire de référence ──────────────────────────────────────────────────

/// @brief  Ajout d'une catégorie de référence puis lecture via GET /api/refs
/// @input  POST /api/refs/categories {"name":"PNEUM"} ; GET /api/refs
/// @expect catégorie "PNEUM" présente dans refs.categories
func TestAPIIntegration_RefsCategoryPersists(t *testing.T) {
	srv := newIntegrationServer(t)
	defer srv.Close()
	client := srv.Client()

	resp := apiDo(t, client, http.MethodPost, srv.URL+"/api/refs/categories", `{"name":"PNEUM"}`)
	if resp.StatusCode != http.StatusCreated {
		resp.Body.Close()
		t.Fatalf("POST refs/categories: got %d, want 201", resp.StatusCode)
	}
	resp.Body.Close()

	resp2 := apiDo(t, client, http.MethodGet, srv.URL+"/api/refs", "")
	var refs struct {
		Categories []string `json:"categories"`
	}
	decodeBody(t, resp2, &refs)
	found := false
	for _, cat := range refs.Categories {
		if cat == "PNEUM" {
			found = true
		}
	}
	if !found {
		t.Errorf("catégorie PNEUM absente dans refs : %v", refs.Categories)
	}
}
