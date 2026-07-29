// adapters/in/rest/tests/handlers_test.go — tests externes des handlers REST
package rest_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"myr-core/adapters/in/rest"
	"myr-core/domain/model"
)

// ── Mock ModelService ─────────────────────────────────────────────────────────

type mockSvc struct {
	addFull                  func(req model.AddRequest) (*model.Model3D, error)
	submit                   func(assetID string) (*model.Model3D, error)
	get                      func(id string) (*model.Model3D, error)
	list                     func(channelID string) ([]*model.Model3D, error)
	verify                   func(id string) (bool, error)
	add                      func(filePath, name, channelID, ownerID string, tags []string) (*model.Model3D, error)
	remove                   func(id string) error
	updateAsset              func(req model.UpdateRequest) (*model.Model3D, error)
	addConnection            func(from, to, label string) (*model.Connection, error)
	addAssemblyLink          func(fromIfaceID, toIfaceID, label, fromInstanceID, toInstanceID, fastenerAssetID string) (*model.Connection, error)
	removeConnection         func(id string) error
	listConnections          func() ([]*model.Connection, error)
	getChildren              func(parentID string) ([]*model.Model3D, error)
	saveThumbnail            func(assetID, dataURL string) error
	getThumbnail             func(assetID string) (string, error)
	regenerateThumbnail      func(assetID string) (string, error)
	addInterface             func(iface *model.AssetInterface) error
	updateInterface          func(iface *model.AssetInterface) error
	removeInterface          func(id string) error
	listInterfacesForAsset   func(assetID string) ([]*model.AssetInterface, error)
	getInterface             func(id string) (*model.AssetInterface, error)
	getRefs                  func() (*model.InterfaceRefs, error)
	addRefCategory           func(cat string) error
	addRefType               func(cat, typeName string) error
	addRefUnit               func(cat, unit string) error
	createModule             func(req model.ModuleRequest) (*model.Model3D, error)
	getModule                func(id string) (*model.Model3D, error)
	listModules              func(channelID string) ([]*model.Model3D, error)
	addAssemblyToModule      func(moduleID, connID string) error
	removeAssemblyFromModule func(moduleID, connID string) error
	submitModule             func(moduleID, note string) (*model.Model3D, error)
	removeModule             func(id string) error
	getModuleInterfaces      func(moduleID string) ([]*model.AssetInterface, error)
	addAssetToWorkspace      func(moduleID, assetID string) (*model.Model3D, error)
	removeAssetFromWorkspace func(moduleID, instanceID string) (*model.Model3D, error)
	updateInstancePosition   func(moduleID, instanceID string, x, y float64) (*model.Model3D, error)
	listLicenses             func() []*model.License
	getLicense               func(id string) (*model.License, error)
	checkLicenseCompat        func(parentID, proposedID string) *model.LicenseCheck
	checkModuleLicenseCompat  func(componentIDs []string, proposedID string) *model.LicenseCheck
	connectVirtualToPhysical  func(virtualIfaceID, physicalIfaceID string, popupValues model.AssetInterface, fromInstanceID, toInstanceID string) (*model.Connection, error)
}

func (m *mockSvc) AddFull(req model.AddRequest) (*model.Model3D, error) {
	if m.addFull != nil {
		return m.addFull(req)
	}
	return &model.Model3D{ID: "new", Name: req.Name, CreatedAt: time.Now()}, nil
}
func (m *mockSvc) Submit(assetID string) (*model.Model3D, error) {
	if m.submit != nil {
		return m.submit(assetID)
	}
	return &model.Model3D{ID: assetID, Status: model.ModuleSubmitted}, nil
}
func (m *mockSvc) Get(id, _ string) (*model.Model3D, error) {
	if m.get != nil {
		return m.get(id)
	}
	return &model.Model3D{ID: id, Name: "asset"}, nil
}
func (m *mockSvc) List(channelID string) ([]*model.Model3D, error) {
	if m.list != nil {
		return m.list(channelID)
	}
	return nil, nil
}
func (m *mockSvc) Verify(id, _ string) (bool, error) {
	if m.verify != nil {
		return m.verify(id)
	}
	return true, nil
}
func (m *mockSvc) Add(filePath, name, channelID, ownerID string, tags []string) (*model.Model3D, error) {
	if m.add != nil {
		return m.add(filePath, name, channelID, ownerID, tags)
	}
	return &model.Model3D{ID: "x", Name: name}, nil
}
func (m *mockSvc) Remove(id string) error {
	if m.remove != nil {
		return m.remove(id)
	}
	return nil
}
func (m *mockSvc) UpdateAsset(req model.UpdateRequest) (*model.Model3D, error) {
	if m.updateAsset != nil {
		return m.updateAsset(req)
	}
	return &model.Model3D{ID: req.ID, Name: req.Name}, nil
}
func (m *mockSvc) AddConnection(from, to, label string) (*model.Connection, error) {
	if m.addConnection != nil {
		return m.addConnection(from, to, label)
	}
	return &model.Connection{ID: "c1", From: from, To: to, Label: label}, nil
}
func (m *mockSvc) AddAssemblyLink(fromIfaceID, toIfaceID, label, fromInstanceID, toInstanceID, fastenerAssetID string) (*model.Connection, error) {
	if m.addAssemblyLink != nil {
		return m.addAssemblyLink(fromIfaceID, toIfaceID, label, fromInstanceID, toInstanceID, fastenerAssetID)
	}
	return &model.Connection{ID: "al1", FromIfaceID: fromIfaceID, ToIfaceID: toIfaceID}, nil
}
func (m *mockSvc) RemoveConnection(id string) error {
	if m.removeConnection != nil {
		return m.removeConnection(id)
	}
	return nil
}
func (m *mockSvc) ListConnections() ([]*model.Connection, error) {
	if m.listConnections != nil {
		return m.listConnections()
	}
	return nil, nil
}
func (m *mockSvc) GetChildren(parentID string) ([]*model.Model3D, error) {
	if m.getChildren != nil {
		return m.getChildren(parentID)
	}
	return nil, nil
}
func (m *mockSvc) SaveThumbnail(assetID, dataURL string) error {
	if m.saveThumbnail != nil {
		return m.saveThumbnail(assetID, dataURL)
	}
	return nil
}
func (m *mockSvc) GetThumbnail(assetID string) (string, error) {
	if m.getThumbnail != nil {
		return m.getThumbnail(assetID)
	}
	return "", nil
}
func (m *mockSvc) RegenerateThumbnail(assetID string) (string, error) {
	if m.regenerateThumbnail != nil {
		return m.regenerateThumbnail(assetID)
	}
	return "", nil
}
func (m *mockSvc) AddInterface(iface *model.AssetInterface) error {
	if m.addInterface != nil {
		return m.addInterface(iface)
	}
	iface.ID = "iface1"
	return nil
}
func (m *mockSvc) UpdateInterface(iface *model.AssetInterface) error {
	if m.updateInterface != nil {
		return m.updateInterface(iface)
	}
	return nil
}
func (m *mockSvc) RemoveInterface(id string) error {
	if m.removeInterface != nil {
		return m.removeInterface(id)
	}
	return nil
}
func (m *mockSvc) ListInterfacesForAsset(assetID string) ([]*model.AssetInterface, error) {
	if m.listInterfacesForAsset != nil {
		return m.listInterfacesForAsset(assetID)
	}
	return nil, nil
}
func (m *mockSvc) GetInterface(id string) (*model.AssetInterface, error) {
	if m.getInterface != nil {
		return m.getInterface(id)
	}
	return &model.AssetInterface{ID: id}, nil
}
func (m *mockSvc) GetRefs() (*model.InterfaceRefs, error) {
	if m.getRefs != nil {
		return m.getRefs()
	}
	return model.DefaultInterfaceRefs(), nil
}
func (m *mockSvc) AddRefCategory(cat string) error {
	if m.addRefCategory != nil {
		return m.addRefCategory(cat)
	}
	return nil
}
func (m *mockSvc) AddRefType(cat, typeName string) error {
	if m.addRefType != nil {
		return m.addRefType(cat, typeName)
	}
	return nil
}
func (m *mockSvc) AddRefUnit(cat, unit string) error {
	if m.addRefUnit != nil {
		return m.addRefUnit(cat, unit)
	}
	return nil
}
func (m *mockSvc) CreateModule(req model.ModuleRequest) (*model.Model3D, error) {
	if m.createModule != nil {
		return m.createModule(req)
	}
	return &model.Model3D{ID: "mod1", Name: req.Name, Status: model.ModuleDraft}, nil
}
func (m *mockSvc) GetModule(id string) (*model.Model3D, error) {
	if m.getModule != nil {
		return m.getModule(id)
	}
	return &model.Model3D{ID: id, Name: "module", Status: model.ModuleDraft}, nil
}
func (m *mockSvc) ListModules(channelID string) ([]*model.Model3D, error) {
	if m.listModules != nil {
		return m.listModules(channelID)
	}
	return nil, nil
}
func (m *mockSvc) AddAssemblyToModule(moduleID, connID string) error {
	if m.addAssemblyToModule != nil {
		return m.addAssemblyToModule(moduleID, connID)
	}
	return nil
}
func (m *mockSvc) RemoveAssemblyFromModule(moduleID, connID string) error {
	if m.removeAssemblyFromModule != nil {
		return m.removeAssemblyFromModule(moduleID, connID)
	}
	return nil
}
func (m *mockSvc) SubmitModule(moduleID, note string) (*model.Model3D, error) {
	if m.submitModule != nil {
		return m.submitModule(moduleID, note)
	}
	return &model.Model3D{ID: moduleID, Status: model.ModuleSubmitted}, nil
}
func (m *mockSvc) RemoveModule(id string) error {
	if m.removeModule != nil {
		return m.removeModule(id)
	}
	return nil
}
func (m *mockSvc) GetModuleInterfaces(moduleID string) ([]*model.AssetInterface, error) {
	if m.getModuleInterfaces != nil {
		return m.getModuleInterfaces(moduleID)
	}
	return nil, nil
}
func (m *mockSvc) AddAssetToWorkspace(moduleID, assetID string) (*model.Model3D, error) {
	if m.addAssetToWorkspace != nil {
		return m.addAssetToWorkspace(moduleID, assetID)
	}
	return &model.Model3D{ID: moduleID, Status: model.ModuleDraft}, nil
}
func (m *mockSvc) RemoveAssetFromWorkspace(moduleID, instanceID string) (*model.Model3D, error) {
	if m.removeAssetFromWorkspace != nil {
		return m.removeAssetFromWorkspace(moduleID, instanceID)
	}
	return &model.Model3D{ID: moduleID, Status: model.ModuleDraft}, nil
}
func (m *mockSvc) UpdateInstancePosition(moduleID, instanceID string, x, y float64) (*model.Model3D, error) {
	if m.updateInstancePosition != nil {
		return m.updateInstancePosition(moduleID, instanceID, x, y)
	}
	return &model.Model3D{ID: moduleID, Status: model.ModuleDraft}, nil
}
func (m *mockSvc) ListLicenses() []*model.License {
	if m.listLicenses != nil {
		return m.listLicenses()
	}
	return []*model.License{{ID: "mit", Name: "MIT"}}
}
func (m *mockSvc) GetLicense(id string) (*model.License, error) {
	if m.getLicense != nil {
		return m.getLicense(id)
	}
	return &model.License{ID: id, Name: id}, nil
}
func (m *mockSvc) CheckLicenseCompatibility(parentID, proposedID string) *model.LicenseCheck {
	if m.checkLicenseCompat != nil {
		return m.checkLicenseCompat(parentID, proposedID)
	}
	return &model.LicenseCheck{Compatible: true}
}
func (m *mockSvc) CheckModuleLicenseCompatibility(componentIDs []string, proposedID string) *model.LicenseCheck {
	if m.checkModuleLicenseCompat != nil {
		return m.checkModuleLicenseCompat(componentIDs, proposedID)
	}
	return &model.LicenseCheck{Compatible: true}
}
func (m *mockSvc) ConnectVirtualToPhysical(virtualIfaceID, physicalIfaceID string, popupValues model.AssetInterface, fromInstanceID, toInstanceID string) (*model.Connection, error) {
	if m.connectVirtualToPhysical != nil {
		return m.connectVirtualToPhysical(virtualIfaceID, physicalIfaceID, popupValues, fromInstanceID, toInstanceID)
	}
	return &model.Connection{ID: "conn1", From: physicalIfaceID, To: virtualIfaceID}, nil
}

func (m *mockSvc) EnsureVirtualSlot(assetID string) {}

// ── Mock ThumbnailStore / InterfaceStore ──────────────────────────────────────

type mockStore struct{}

func (s *mockStore) SaveThumbnail(assetID, dataURL string) error { return nil }
func (s *mockStore) GetThumbnail(assetID string) (string, error) { return "", nil }
func (s *mockStore) SaveInterface(iface *model.AssetInterface) error {
	return nil
}
func (s *mockStore) RemoveInterface(id string) error { return nil }
func (s *mockStore) ListInterfacesForAsset(assetID string) ([]*model.AssetInterface, error) {
	return nil, nil
}
func (s *mockStore) GetInterface(id string) (*model.AssetInterface, error) {
	return &model.AssetInterface{ID: id}, nil
}
func (s *mockStore) GetRefs() (*model.InterfaceRefs, error) {
	return model.DefaultInterfaceRefs(), nil
}
func (s *mockStore) AddRefCategory(cat string) error       { return nil }
func (s *mockStore) AddRefType(cat, typeName string) error { return nil }
func (s *mockStore) AddRefUnit(cat, unit string) error     { return nil }

// ── helpers ───────────────────────────────────────────────────────────────────

// newTestMux builds a full REST mux with an identity service configured so
// tests can obtain a valid session token via loginToken — requireAuth always
// requires a token, there is no auth bypass.
func newTestMux(t *testing.T, svc model.ModelService) http.Handler {
	t.Helper()
	store := &mockStore{}
	h := rest.NewHandler(svc, store, store)
	h.WithIdentityService(&mockIdentitySvc{})
	srv := rest.NewServer(h, "localhost:0")
	mux, err := srv.Handler()
	if err != nil {
		t.Fatalf("Handler(): %v", err)
	}
	return mux
}

// newBlockchainMux builds a mux with network info set (multi-réseau, canal actif).
func newBlockchainMux(t *testing.T, svc model.ModelService) http.Handler {
	t.Helper()
	store := &mockStore{}
	h := rest.NewHandler(svc, store, store)
	h.WithIdentityService(&mockIdentitySvc{})
	h.WithNetworkInfo(rest.NetworkInfo{Network: "peer0.example.com:7051", Channel: "mychannel"})
	srv := rest.NewServer(h, "localhost:0")
	mux, err := srv.Handler()
	if err != nil {
		t.Fatalf("Handler(): %v", err)
	}
	return mux
}

// loginToken effectue une vraie connexion via /api/identity/session (l'identité
// mock accepte tout secret) et retourne un token de session valide pour mux.
func loginToken(t *testing.T, mux http.Handler) string {
	t.Helper()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/identity/session",
		strings.NewReader(`{"name":"test","secret":"test","org_id":"TestMSP"}`))
	r.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(w, r)
	var resp struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil || resp.Token == "" {
		t.Fatalf("loginToken: échec (%d): %s", w.Code, w.Body.String())
	}
	return resp.Token
}

func newRequest(method, path, body string) *http.Request {
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	return r
}

func do(t *testing.T, mux http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	r := newRequest(method, path, body)
	r.Header.Set("X-Myr-Token", loginToken(t, mux))
	mux.ServeHTTP(w, r)
	return w
}

func decodeJSON(t *testing.T, w *httptest.ResponseRecorder, v any) {
	t.Helper()
	if err := json.NewDecoder(w.Body).Decode(v); err != nil {
		t.Fatalf("decode JSON: %v (body: %s)", err, w.Body.String())
	}
}

func buildMultipartForm(t *testing.T, fields map[string]string, fileField, filename string, content []byte) (*bytes.Buffer, string) {
	t.Helper()
	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	for k, v := range fields {
		if err := w.WriteField(k, v); err != nil {
			t.Fatalf("WriteField %q: %v", k, err)
		}
	}
	if fileField != "" {
		part, err := w.CreateFormFile(fileField, filename)
		if err != nil {
			t.Fatalf("CreateFormFile: %v", err)
		}
		part.Write(content)
	}
	w.Close()
	return body, w.FormDataContentType()
}

// ── /api/status ───────────────────────────────────────────────────────────────

/// @brief  Vérifie que GET /api/status retourne le mode et le nombre d'assets
/// @input  GET /api/status, aucun réseau Fabric configuré, mockSvc.list retourne 2 assets
/// @expect HTTP 200, mode="blockchain", assets=2
func TestStatus_ReturnsMode(t *testing.T) {
	svc := &mockSvc{list: func(string) ([]*model.Model3D, error) {
		return []*model.Model3D{{ID: "a"}, {ID: "b"}}, nil
	}}
	w := do(t, newTestMux(t, svc), http.MethodGet, "/api/status", "")
	if w.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", w.Code)
	}
	var resp struct {
		Mode   string `json:"mode"`
		Assets int    `json:"assets"`
	}
	decodeJSON(t, w, &resp)
	if resp.Mode != "blockchain" {
		t.Errorf("mode: got %q, want %q", resp.Mode, "blockchain")
	}
	if resp.Assets != 2 {
		t.Errorf("assets: got %d, want 2", resp.Assets)
	}
}

// ── /api/components ───────────────────────────────────────────────────────────

/// @brief  Vérifie que GET /api/components retourne une liste vide quand aucun asset n'existe
/// @input  GET /api/components, mockSvc.list retourne nil
/// @expect HTTP 200, total=0
func TestComponents_GET_Empty(t *testing.T) {
	w := do(t, newTestMux(t, &mockSvc{}), http.MethodGet, "/api/components", "")
	if w.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", w.Code)
	}
	var resp struct{ Total int `json:"total"` }
	decodeJSON(t, w, &resp)
	if resp.Total != 0 {
		t.Errorf("total: got %d, want 0", resp.Total)
	}
}

/// @brief  Vérifie que GET /api/components exclut les modules de la liste des composants
/// @input  GET /api/components, mockSvc.list retourne 2 assets dont 1 module (status=ModuleDraft)
/// @expect HTTP 200, total=1 (le module est exclu)
func TestComponents_GET_ExcludesModules(t *testing.T) {
	svc := &mockSvc{list: func(string) ([]*model.Model3D, error) {
		return []*model.Model3D{
			{ID: "asset1", Name: "Asset"},
			{ID: "mod1", Name: "Module", Status: model.ModuleDraft, Assemblies: []string{}},
		}, nil
	}}
	w := do(t, newTestMux(t, svc), http.MethodGet, "/api/components", "")
	var resp struct{ Total int `json:"total"` }
	decodeJSON(t, w, &resp)
	if resp.Total != 1 {
		t.Errorf("total: got %d, want 1 (modules exclus)", resp.Total)
	}
}

/// @brief  Vérifie que GET /api/components?q= filtre les composants par nom
/// @input  GET /api/components?q=vis, mockSvc.list retourne "Vis M3" et "Engrenage"
/// @expect HTTP 200, total=1 (seul "Vis M3" correspond)
func TestComponents_GET_FilterByQuery(t *testing.T) {
	svc := &mockSvc{list: func(string) ([]*model.Model3D, error) {
		return []*model.Model3D{
			{ID: "1", Name: "Vis M3"},
			{ID: "2", Name: "Engrenage"},
		}, nil
	}}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/components?q=vis", nil)
	mux := newTestMux(t, svc)
	r.Header.Set("X-Myr-Token", loginToken(t, mux))
	mux.ServeHTTP(w, r)
	var resp struct{ Total int `json:"total"` }
	decodeJSON(t, w, &resp)
	if resp.Total != 1 {
		t.Errorf("filter q=vis: got %d, want 1", resp.Total)
	}
}

/// @brief  Vérifie que GET /api/components?categories= filtre les composants par catégorie
/// @input  GET /api/components?categories=base, mockSvc.list retourne 1 asset "base" et 1 "variation"
/// @expect HTTP 200, total=1 (seule la catégorie "base" correspond)
func TestComponents_GET_FilterByCategory(t *testing.T) {
	svc := &mockSvc{list: func(string) ([]*model.Model3D, error) {
		return []*model.Model3D{
			{ID: "1", Name: "A", Category: model.CategoryBase},
			{ID: "2", Name: "B", Category: model.CategoryVariation},
		}, nil
	}}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/components?categories=base", nil)
	mux := newTestMux(t, svc)
	r.Header.Set("X-Myr-Token", loginToken(t, mux))
	mux.ServeHTTP(w, r)
	var resp struct{ Total int `json:"total"` }
	decodeJSON(t, w, &resp)
	if resp.Total != 1 {
		t.Errorf("filter categories=base: got %d, want 1", resp.Total)
	}
}

/// @brief  Vérifie que GET /api/components?tags= filtre les composants par tag
/// @input  GET /api/components?tags=métal, mockSvc.list retourne 1 asset ["métal","impression"] et 1 ["bois"]
/// @expect HTTP 200, total=1 (seul l'asset ayant le tag "métal" correspond)
func TestComponents_GET_FilterByTag(t *testing.T) {
	svc := &mockSvc{list: func(string) ([]*model.Model3D, error) {
		return []*model.Model3D{
			{ID: "1", Name: "A", Tags: []string{"métal", "impression"}},
			{ID: "2", Name: "B", Tags: []string{"bois"}},
		}, nil
	}}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/components?tags=métal", nil)
	mux := newTestMux(t, svc)
	r.Header.Set("X-Myr-Token", loginToken(t, mux))
	mux.ServeHTTP(w, r)
	var resp struct{ Total int `json:"total"` }
	decodeJSON(t, w, &resp)
	if resp.Total != 1 {
		t.Errorf("filter tags=métal: got %d, want 1", resp.Total)
	}
}

/// @brief  Vérifie que GET /api/components retourne 500 quand le service échoue
/// @input  GET /api/components, mockSvc.list retourne une erreur "db error"
/// @expect HTTP 500
func TestComponents_GET_ServiceError(t *testing.T) {
	svc := &mockSvc{list: func(string) ([]*model.Model3D, error) {
		return nil, errors.New("db error")
	}}
	w := do(t, newTestMux(t, svc), http.MethodGet, "/api/components", "")
	if w.Code != http.StatusInternalServerError {
		t.Errorf("got %d, want 500", w.Code)
	}
}

/// @brief  Vérifie que POST /api/components sans champ "name" retourne 400
/// @input  POST /api/components, multipart avec owner_id uniquement (name absent)
/// @expect HTTP 400
func TestComponents_POST_MissingName(t *testing.T) {
	body := &bytes.Buffer{}
	body.WriteString("--boundary\r\nContent-Disposition: form-data; name=\"owner_id\"\r\n\r\nuser1\r\n--boundary--\r\n")
	req := httptest.NewRequest(http.MethodPost, "/api/components", body)
	req.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")
	w := httptest.NewRecorder()
	mux := newTestMux(t, &mockSvc{})
	req.Header.Set("X-Myr-Token", loginToken(t, mux))
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("got %d, want 400", w.Code)
	}
}

/// @brief  Vérifie que POST /api/components crée toujours un composant en brouillon
///         (status="draft") et retourne 201 — aucune transaction blockchain à la création
/// @input  POST /api/components, multipart avec name="Roulement", mockSvc.addFull retourne l'asset créé
/// @expect HTTP 201, name="Roulement" et status="draft" dans la réponse, AddFull appelé
func TestComponents_POST_Created(t *testing.T) {
	called := false
	svc := &mockSvc{addFull: func(req model.AddRequest) (*model.Model3D, error) {
		called = true
		return &model.Model3D{ID: "new1", Name: req.Name, Status: model.ModuleDraft, CreatedAt: time.Now()}, nil
	}}

	body := &bytes.Buffer{}
	body.WriteString("--boundary\r\n")
	body.WriteString("Content-Disposition: form-data; name=\"name\"\r\n\r\n")
	body.WriteString("Roulement\r\n")
	body.WriteString("--boundary--\r\n")

	req := httptest.NewRequest(http.MethodPost, "/api/components", body)
	req.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")
	w := httptest.NewRecorder()
	mux := newTestMux(t, svc)
	req.Header.Set("X-Myr-Token", loginToken(t, mux))
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("got %d, want 201 (body: %s)", w.Code, w.Body.String())
	}
	if !called {
		t.Error("AddFull not called")
	}
	var dto struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	decodeJSON(t, w, &dto)
	if dto.Name != "Roulement" {
		t.Errorf("name: got %q, want %q", dto.Name, "Roulement")
	}
	if dto.Status != "draft" {
		t.Errorf("status: got %q, want %q", dto.Status, "draft")
	}
}

/// @brief  Vérifie que POST /api/components/{id}/submit engage le composant et
///         renvoie le componentDTO à jour (status="submitted")
/// @input  POST /api/components/comp1/submit, mockSvc.submit retourne l'asset soumis
/// @expect HTTP 200, Submit appelé avec "comp1", status="submitted" dans la réponse
func TestComponents_POST_Submit(t *testing.T) {
	var gotID string
	svc := &mockSvc{submit: func(assetID string) (*model.Model3D, error) {
		gotID = assetID
		return &model.Model3D{ID: assetID, Name: "Vis", Status: model.ModuleSubmitted}, nil
	}}
	w := do(t, newTestMux(t, svc), http.MethodPost, "/api/components/comp1/submit", "")
	if w.Code != http.StatusOK {
		t.Fatalf("got %d, want 200 (body: %s)", w.Code, w.Body.String())
	}
	if gotID != "comp1" {
		t.Errorf("Submit appelé avec %q, want %q", gotID, "comp1")
	}
	var dto struct {
		Status string `json:"status"`
	}
	decodeJSON(t, w, &dto)
	if dto.Status != "submitted" {
		t.Errorf("status: got %q, want %q", dto.Status, "submitted")
	}
}

/// @brief  Vérifie que POST /api/components/{id}/submit sur un brouillon inexistant
///         renvoie une erreur (pas 200)
/// @input  mockSvc.submit retourne une erreur générique
/// @expect HTTP != 200
func TestComponents_POST_Submit_Error(t *testing.T) {
	svc := &mockSvc{submit: func(assetID string) (*model.Model3D, error) {
		return nil, errors.New("brouillon introuvable")
	}}
	w := do(t, newTestMux(t, svc), http.MethodPost, "/api/components/unknown/submit", "")
	if w.Code == http.StatusOK {
		t.Errorf("got 200, want une erreur")
	}
}

/// @brief  Vérifie qu'une erreur ErrBlockchainUnavailable est distinguée d'une erreur
///         générique (503, pas 500) — Problème 2 signalé par le dépôt GUI
/// @input  POST /api/components, mockSvc.addFull retourne model.ErrBlockchainUnavailable
/// @expect HTTP 503 (pas 500, pas 400)
func TestComponents_POST_BlockchainUnavailable_Returns503(t *testing.T) {
	svc := &mockSvc{addFull: func(req model.AddRequest) (*model.Model3D, error) {
		return nil, model.ErrBlockchainUnavailable
	}}
	body := &bytes.Buffer{}
	body.WriteString("--boundary\r\nContent-Disposition: form-data; name=\"name\"\r\n\r\nVis\r\n--boundary--\r\n")
	req := httptest.NewRequest(http.MethodPost, "/api/components", body)
	req.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")
	w := httptest.NewRecorder()
	mux := newTestMux(t, svc)
	req.Header.Set("X-Myr-Token", loginToken(t, mux))
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("got %d, want 503", w.Code)
	}
}

/// @brief  Vérifie qu'une erreur de service générique (ni ErrBlockchainUnavailable, ni
///         une 400 de validation pré-domaine) reste un 500 — la distinction 503 ne doit
///         pas masquer les autres échecs
/// @input  mockSvc.addFull retourne une erreur générique ("hashing file: ...")
/// @expect HTTP 500
func TestComponents_POST_GenericServiceError_Returns500(t *testing.T) {
	svc := &mockSvc{addFull: func(req model.AddRequest) (*model.Model3D, error) {
		return nil, errors.New("hashing file: corrupted")
	}}
	body := &bytes.Buffer{}
	body.WriteString("--boundary\r\nContent-Disposition: form-data; name=\"name\"\r\n\r\nVis\r\n--boundary--\r\n")
	req := httptest.NewRequest(http.MethodPost, "/api/components", body)
	req.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")
	w := httptest.NewRecorder()
	mux := newTestMux(t, svc)
	req.Header.Set("X-Myr-Token", loginToken(t, mux))
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("got %d, want 500", w.Code)
	}
}

/// @brief  Vérifie que PUT /api/components retourne 405 (méthode non autorisée)
/// @input  PUT /api/components
/// @expect HTTP 405
func TestComponents_MethodNotAllowed(t *testing.T) {
	w := do(t, newTestMux(t, &mockSvc{}), http.MethodPut, "/api/components", "")
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("got %d, want 405", w.Code)
	}
}

// ── /api/components POST — upload fichier 3D ─────────────────────────────────

/// @brief  Vérifie que POST /api/components avec un fichier STL renseigne correctement FilePath
/// @input  POST /api/components, multipart avec name="Engrenage", owner_id="user1", fichier STL sous le champ "stl"
/// @expect HTTP 201, AddFull appelé avec FilePath non vide, Name="Engrenage", OwnerID="user1", id="c1" dans la réponse
func TestComponents_POST_WithSTLFile(t *testing.T) {
	var capturedReq model.AddRequest
	svc := &mockSvc{addFull: func(req model.AddRequest) (*model.Model3D, error) {
		capturedReq = req
		return &model.Model3D{ID: "c1", Name: req.Name, CreatedAt: time.Now()}, nil
	}}

	stlContent := []byte("solid test\nfacet normal 0 0 1\nouter loop\nendloop\nendfacet\nendsolid test\n")
	body, ct := buildMultipartForm(t,
		map[string]string{"name": "Engrenage", "owner_id": "user1"},
		"stl", "engrenage.stl", stlContent)

	req := httptest.NewRequest(http.MethodPost, "/api/components", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	mux := newTestMux(t, svc)
	req.Header.Set("X-Myr-Token", loginToken(t, mux))
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("got %d, want 201 (body: %s)", w.Code, w.Body.String())
	}
	if capturedReq.FilePath == "" {
		t.Error("FilePath doit être renseigné quand un fichier STL est fourni")
	}
	if capturedReq.Name != "Engrenage" {
		t.Errorf("Name: got %q, want Engrenage", capturedReq.Name)
	}
	if capturedReq.OwnerID != "user1" {
		t.Errorf("OwnerID: got %q, want user1", capturedReq.OwnerID)
	}
	var dto struct{ ID string `json:"id"` }
	decodeJSON(t, w, &dto)
	if dto.ID != "c1" {
		t.Errorf("ID: got %q, want c1", dto.ID)
	}
}

/// @brief  Vérifie que POST /api/components accepte un fichier sous le champ générique "file"
/// @input  POST /api/components, multipart avec name="Piece STEP", fichier STEP sous le champ "file"
/// @expect HTTP 201, AddFull appelé avec FilePath non vide
func TestComponents_POST_WithFileField(t *testing.T) {
	var capturedReq model.AddRequest
	svc := &mockSvc{addFull: func(req model.AddRequest) (*model.Model3D, error) {
		capturedReq = req
		return &model.Model3D{ID: "c2", Name: req.Name, CreatedAt: time.Now()}, nil
	}}

	stepContent := []byte("ISO-10303-21;\nHEADER;\nENDHEADER;\nDATA;\nENDSEC;\nEND-ISO-10303-21;\n")
	body, ct := buildMultipartForm(t,
		map[string]string{"name": "Piece STEP"},
		"file", "piece.step", stepContent)

	req := httptest.NewRequest(http.MethodPost, "/api/components", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	mux := newTestMux(t, svc)
	req.Header.Set("X-Myr-Token", loginToken(t, mux))
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("got %d, want 201 (body: %s)", w.Code, w.Body.String())
	}
	if capturedReq.FilePath == "" {
		t.Error("FilePath doit être renseigné quand un fichier est fourni sous 'file'")
	}
}

/// @brief  Vérifie que POST /api/components sans fichier 3D laisse FilePath vide et transmet les tags
/// @input  POST /api/components, multipart avec name="Vis M3", description, tags="fixation,métal", sans fichier
/// @expect HTTP 201, FilePath vide, Tags=["fixation","métal"]
func TestComponents_POST_WithoutFile(t *testing.T) {
	var capturedReq model.AddRequest
	svc := &mockSvc{addFull: func(req model.AddRequest) (*model.Model3D, error) {
		capturedReq = req
		return &model.Model3D{ID: "c3", Name: req.Name, CreatedAt: time.Now()}, nil
	}}

	body, ct := buildMultipartForm(t,
		map[string]string{"name": "Vis M3", "description": "vis acier inox", "tags": "fixation,métal"},
		"", "", nil)

	req := httptest.NewRequest(http.MethodPost, "/api/components", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	mux := newTestMux(t, svc)
	req.Header.Set("X-Myr-Token", loginToken(t, mux))
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("got %d, want 201 (body: %s)", w.Code, w.Body.String())
	}
	if capturedReq.FilePath != "" {
		t.Errorf("FilePath doit être vide sans fichier, got %q", capturedReq.FilePath)
	}
	if len(capturedReq.Tags) != 2 {
		t.Errorf("Tags: got %v, want [fixation métal]", capturedReq.Tags)
	}
}

/// @brief  Vérifie que POST /api/components sauvegarde la miniature fournie en data-URL
/// @input  POST /api/components, multipart avec name="Rotor", thumbnail (data-URL PNG), fichier STL
/// @expect HTTP 201, SaveThumbnail appelé avec la data-URL exacte
func TestComponents_POST_WithThumbnail(t *testing.T) {
	thumbSaved := ""
	svc := &mockSvc{
		addFull: func(req model.AddRequest) (*model.Model3D, error) {
			return &model.Model3D{ID: "c4", Name: req.Name, CreatedAt: time.Now()}, nil
		},
		saveThumbnail: func(assetID, dataURL string) error {
			thumbSaved = dataURL
			return nil
		},
	}

	thumb := "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="
	body, ct := buildMultipartForm(t,
		map[string]string{"name": "Rotor", "thumbnail": thumb},
		"stl", "rotor.stl", []byte("solid rotor\nendsolid rotor\n"))

	req := httptest.NewRequest(http.MethodPost, "/api/components", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	mux := newTestMux(t, svc)
	req.Header.Set("X-Myr-Token", loginToken(t, mux))
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("got %d, want 201 (body: %s)", w.Code, w.Body.String())
	}
	if thumbSaved != thumb {
		t.Errorf("thumbnail non sauvegardée: got %q", thumbSaved)
	}
}

/// @brief  Vérifie que POST /api/components/{id}/thumbnail/regenerate délègue au service et renvoie la miniature régénérée
/// @input  POST /api/components/c1/thumbnail/regenerate, RegenerateThumbnail simulé retournant une dataURL fixe
/// @expect HTTP 200, corps {"thumbnail": dataURL}, RegenerateThumbnail appelé avec l'id du composant
func TestComponents_RegenerateThumbnail_OK(t *testing.T) {
	var gotID string
	svc := &mockSvc{regenerateThumbnail: func(assetID string) (string, error) {
		gotID = assetID
		return "data:image/png;base64,regenerated", nil
	}}

	w := do(t, newTestMux(t, svc), http.MethodPost, "/api/components/c1/thumbnail/regenerate", "")

	if w.Code != http.StatusOK {
		t.Fatalf("got %d, want 200 (body: %s)", w.Code, w.Body.String())
	}
	if gotID != "c1" {
		t.Errorf("RegenerateThumbnail appelé avec %q, want c1", gotID)
	}
	var resp map[string]string
	decodeJSON(t, w, &resp)
	if resp["thumbnail"] != "data:image/png;base64,regenerated" {
		t.Errorf("thumbnail: got %q", resp["thumbnail"])
	}
}

/// @brief  Vérifie que POST /api/components/{id}/thumbnail/regenerate propage l'échec (ex: aucun lien externe) en 500
/// @input  POST /api/components/c1/thumbnail/regenerate, RegenerateThumbnail simulé retournant model.ErrNoThumbnailSource
/// @expect HTTP 500
func TestComponents_RegenerateThumbnail_NoSource_Rejected(t *testing.T) {
	svc := &mockSvc{regenerateThumbnail: func(assetID string) (string, error) {
		return "", model.ErrNoThumbnailSource
	}}

	w := do(t, newTestMux(t, svc), http.MethodPost, "/api/components/c1/thumbnail/regenerate", "")

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("got %d, want 500 (body: %s)", w.Code, w.Body.String())
	}
}

/// @brief  Vérifie que POST /api/modules/{id}/thumbnail/regenerate délègue au même service que les composants
/// @input  POST /api/modules/m1/thumbnail/regenerate, RegenerateThumbnail simulé retournant une dataURL fixe
/// @expect HTTP 200, corps {"thumbnail": dataURL}, RegenerateThumbnail appelé avec l'id du module
func TestModules_RegenerateThumbnail_OK(t *testing.T) {
	var gotID string
	svc := &mockSvc{regenerateThumbnail: func(assetID string) (string, error) {
		gotID = assetID
		return "data:image/png;base64,regenerated", nil
	}}

	w := do(t, newTestMux(t, svc), http.MethodPost, "/api/modules/m1/thumbnail/regenerate", "")

	if w.Code != http.StatusOK {
		t.Fatalf("got %d, want 200 (body: %s)", w.Code, w.Body.String())
	}
	if gotID != "m1" {
		t.Errorf("RegenerateThumbnail appelé avec %q, want m1", gotID)
	}
}

/// @brief  Vérifie que GET /api/components/{id}/thumbnail/regenerate est rejeté (action, pas une ressource lisible)
/// @input  GET /api/components/c1/thumbnail/regenerate
/// @expect HTTP 405
func TestComponents_RegenerateThumbnail_WrongMethod_Rejected(t *testing.T) {
	w := do(t, newTestMux(t, &mockSvc{}), http.MethodGet, "/api/components/c1/thumbnail/regenerate", "")

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("got %d, want 405 (body: %s)", w.Code, w.Body.String())
	}
}

/// @brief  Vérifie que POST /api/components avec un nom de plus de 256 caractères retourne 400
/// @input  POST /api/components, multipart avec name=257×"a"
/// @expect HTTP 400
func TestComponents_POST_NameTooLong(t *testing.T) {
	body, ct := buildMultipartForm(t,
		map[string]string{"name": strings.Repeat("a", 257)},
		"", "", nil)
	req := httptest.NewRequest(http.MethodPost, "/api/components", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	mux := newTestMux(t, &mockSvc{})
	req.Header.Set("X-Myr-Token", loginToken(t, mux))
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("got %d, want 400 (name trop long)", w.Code)
	}
}

/// @brief  Vérifie que POST /api/components avec un owner_id contenant des espaces retourne 400
/// @input  POST /api/components, multipart avec name="Vis", owner_id="user with spaces"
/// @expect HTTP 400
func TestComponents_POST_InvalidOwnerID(t *testing.T) {
	body, ct := buildMultipartForm(t,
		map[string]string{"name": "Vis", "owner_id": "user with spaces"},
		"", "", nil)
	req := httptest.NewRequest(http.MethodPost, "/api/components", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	mux := newTestMux(t, &mockSvc{})
	req.Header.Set("X-Myr-Token", loginToken(t, mux))
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("got %d, want 400 (owner_id invalide)", w.Code)
	}
}

/// @brief  Vérifie que POST /api/components retourne 500 quand le service échoue
/// @input  POST /api/components, multipart avec name="Vis", mockSvc.addFull retourne une erreur
/// @expect HTTP 500
func TestComponents_POST_ServiceError(t *testing.T) {
	svc := &mockSvc{addFull: func(req model.AddRequest) (*model.Model3D, error) {
		return nil, errors.New("blockchain indisponible")
	}}
	body, ct := buildMultipartForm(t, map[string]string{"name": "Vis"}, "", "", nil)
	req := httptest.NewRequest(http.MethodPost, "/api/components", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	mux := newTestMux(t, svc)
	req.Header.Set("X-Myr-Token", loginToken(t, mux))
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("got %d, want 500", w.Code)
	}
}

/// @brief  Vérifie que POST /api/components sans catégorie applique CategoryBase par défaut
/// @input  POST /api/components, multipart avec name="Composant" uniquement
/// @expect HTTP 201, AddFull appelé avec Category=CategoryBase
func TestComponents_POST_CategoryDefault(t *testing.T) {
	var capturedReq model.AddRequest
	svc := &mockSvc{addFull: func(req model.AddRequest) (*model.Model3D, error) {
		capturedReq = req
		return &model.Model3D{ID: "c5", Name: req.Name, CreatedAt: time.Now()}, nil
	}}
	body, ct := buildMultipartForm(t, map[string]string{"name": "Composant"}, "", "", nil)
	req := httptest.NewRequest(http.MethodPost, "/api/components", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	mux := newTestMux(t, svc)
	req.Header.Set("X-Myr-Token", loginToken(t, mux))
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("got %d, want 201", w.Code)
	}
	if capturedReq.Category != model.CategoryBase {
		t.Errorf("Category: got %q, want %q", capturedReq.Category, model.CategoryBase)
	}
}

// ── /api/components/:id ───────────────────────────────────────────────────────

/// @brief  Vérifie que GET /api/components/:id retourne le composant correspondant
/// @input  GET /api/components/abc, mockSvc.get retourne un asset avec ID="abc"
/// @expect HTTP 200, id="abc" dans la réponse JSON
func TestComponent_GET_Found(t *testing.T) {
	svc := &mockSvc{get: func(id string) (*model.Model3D, error) {
		return &model.Model3D{ID: id, Name: "Vis"}, nil
	}}
	w := do(t, newTestMux(t, svc), http.MethodGet, "/api/components/abc", "")
	if w.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", w.Code)
	}
	var dto struct{ ID string `json:"id"` }
	decodeJSON(t, w, &dto)
	if dto.ID != "abc" {
		t.Errorf("ID: got %q, want %q", dto.ID, "abc")
	}
}

func TestComponent_GET_NotFound(t *testing.T) {
	svc := &mockSvc{get: func(id string) (*model.Model3D, error) {
		return nil, errors.New("not found")
	}}
	w := do(t, newTestMux(t, svc), http.MethodGet, "/api/components/xyz", "")
	if w.Code != http.StatusNotFound {
		t.Errorf("got %d, want 404", w.Code)
	}
}

func TestComponent_PATCH_OK(t *testing.T) {
	svc := &mockSvc{updateAsset: func(req model.UpdateRequest) (*model.Model3D, error) {
		return &model.Model3D{ID: req.ID, Name: req.Name}, nil
	}}
	w := do(t, newTestMux(t, svc), http.MethodPatch, "/api/components/abc",
		`{"name":"Nouveau nom","tags":["métal"]}`)
	if w.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", w.Code)
	}
}

func TestComponent_PATCH_BadJSON(t *testing.T) {
	w := do(t, newTestMux(t, &mockSvc{}), http.MethodPatch, "/api/components/abc", "{bad")
	if w.Code != http.StatusBadRequest {
		t.Errorf("got %d, want 400", w.Code)
	}
}

func TestComponent_DELETE_NoContent(t *testing.T) {
	w := do(t, newTestMux(t, &mockSvc{}), http.MethodDelete, "/api/components/abc", "")
	if w.Code != http.StatusNoContent {
		t.Errorf("got %d, want 204", w.Code)
	}
}

func TestComponent_DELETE_ServiceError(t *testing.T) {
	svc := &mockSvc{remove: func(id string) error { return errors.New("locked") }}
	w := do(t, newTestMux(t, svc), http.MethodDelete, "/api/components/abc", "")
	if w.Code != http.StatusInternalServerError {
		t.Errorf("got %d, want 500", w.Code)
	}
}

// ── /api/components/:id/interfaces ───────────────────────────────────────────

func TestComponentInterfaces_GET(t *testing.T) {
	svc := &mockSvc{listInterfacesForAsset: func(assetID string) ([]*model.AssetInterface, error) {
		return []*model.AssetInterface{{ID: "i1", AssetID: assetID}}, nil
	}}
	w := do(t, newTestMux(t, svc), http.MethodGet, "/api/components/abc/interfaces", "")
	if w.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", w.Code)
	}
	var ifaces []*model.AssetInterface
	decodeJSON(t, w, &ifaces)
	if len(ifaces) != 1 {
		t.Errorf("got %d interfaces, want 1", len(ifaces))
	}
}

func TestComponentInterfaces_POST_Created(t *testing.T) {
	w := do(t, newTestMux(t, &mockSvc{}), http.MethodPost, "/api/components/abc/interfaces",
		`{"category":"ELEC","type":"USB-C","direction":"out"}`)
	if w.Code != http.StatusCreated {
		t.Errorf("got %d, want 201 (body: %s)", w.Code, w.Body.String())
	}
}

func TestComponentInterfaces_POST_BadJSON(t *testing.T) {
	w := do(t, newTestMux(t, &mockSvc{}), http.MethodPost, "/api/components/abc/interfaces", "{bad")
	if w.Code != http.StatusBadRequest {
		t.Errorf("got %d, want 400", w.Code)
	}
}

// ── /api/interfaces/:id ───────────────────────────────────────────────────────

func TestInterface_DELETE(t *testing.T) {
	w := do(t, newTestMux(t, &mockSvc{}), http.MethodDelete, "/api/interfaces/i1", "")
	if w.Code != http.StatusNoContent {
		t.Errorf("got %d, want 204", w.Code)
	}
}

func TestInterface_PATCH_OK(t *testing.T) {
	w := do(t, newTestMux(t, &mockSvc{}), http.MethodPatch, "/api/interfaces/i1",
		`{"category":"MECA","type":"Vis M3","direction":"in"}`)
	if w.Code != http.StatusOK {
		t.Errorf("got %d, want 200 (body: %s)", w.Code, w.Body.String())
	}
}

func TestInterface_PATCH_NotFound(t *testing.T) {
	svc := &mockSvc{getInterface: func(id string) (*model.AssetInterface, error) {
		return nil, errors.New("not found")
	}}
	w := do(t, newTestMux(t, svc), http.MethodPatch, "/api/interfaces/i99", `{"category":"ELEC"}`)
	if w.Code != http.StatusNotFound {
		t.Errorf("got %d, want 404", w.Code)
	}
}

// ── /api/connections ─────────────────────────────────────────────────────────

func TestConnections_POST_Created(t *testing.T) {
	w := do(t, newTestMux(t, &mockSvc{}), http.MethodPost, "/api/connections",
		`{"from":"a1","to":"a2","label":"assemblage"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("got %d, want 201 (body: %s)", w.Code, w.Body.String())
	}
	var dto struct {
		From string `json:"from"`
		To   string `json:"to"`
	}
	decodeJSON(t, w, &dto)
	if dto.From != "a1" || dto.To != "a2" {
		t.Errorf("from/to: got %q/%q, want a1/a2", dto.From, dto.To)
	}
}

func TestConnections_POST_MissingTo(t *testing.T) {
	w := do(t, newTestMux(t, &mockSvc{}), http.MethodPost, "/api/connections", `{"from":"a1"}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("got %d, want 400", w.Code)
	}
}

func TestConnections_MethodNotAllowed(t *testing.T) {
	w := do(t, newTestMux(t, &mockSvc{}), http.MethodGet, "/api/connections", "")
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("got %d, want 405", w.Code)
	}
}

// ── /api/connections/:id ─────────────────────────────────────────────────────

func TestConnection_DELETE(t *testing.T) {
	w := do(t, newTestMux(t, &mockSvc{}), http.MethodDelete, "/api/connections/c1", "")
	if w.Code != http.StatusNoContent {
		t.Errorf("got %d, want 204", w.Code)
	}
}

func TestConnection_DELETE_ServiceError(t *testing.T) {
	svc := &mockSvc{removeConnection: func(id string) error { return errors.New("locked") }}
	w := do(t, newTestMux(t, svc), http.MethodDelete, "/api/connections/c1", "")
	if w.Code != http.StatusInternalServerError {
		t.Errorf("got %d, want 500", w.Code)
	}
}

func TestConnection_MethodNotAllowed(t *testing.T) {
	w := do(t, newTestMux(t, &mockSvc{}), http.MethodGet, "/api/connections/c1", "")
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("got %d, want 405", w.Code)
	}
}

// ── /api/assembly-links ───────────────────────────────────────────────────────

func TestAssemblyLinks_POST_Created(t *testing.T) {
	w := do(t, newTestMux(t, &mockSvc{}), http.MethodPost, "/api/assembly-links",
		`{"from_iface_id":"i1","to_iface_id":"i2","label":"vis"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("got %d, want 201 (body: %s)", w.Code, w.Body.String())
	}
	var dto struct {
		FromIfaceID string `json:"from_iface_id"`
		ToIfaceID   string `json:"to_iface_id"`
	}
	decodeJSON(t, w, &dto)
	if dto.FromIfaceID != "i1" || dto.ToIfaceID != "i2" {
		t.Errorf("iface IDs: got %q/%q, want i1/i2", dto.FromIfaceID, dto.ToIfaceID)
	}
}

func TestAssemblyLinks_POST_MissingToIface(t *testing.T) {
	w := do(t, newTestMux(t, &mockSvc{}), http.MethodPost, "/api/assembly-links",
		`{"from_iface_id":"i1"}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("got %d, want 400", w.Code)
	}
}

func TestAssemblyLinks_MethodNotAllowed(t *testing.T) {
	w := do(t, newTestMux(t, &mockSvc{}), http.MethodGet, "/api/assembly-links", "")
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("got %d, want 405", w.Code)
	}
}

// ── /api/modules ──────────────────────────────────────────────────────────────

func TestModules_GET_Empty(t *testing.T) {
	w := do(t, newTestMux(t, &mockSvc{}), http.MethodGet, "/api/modules", "")
	if w.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", w.Code)
	}
	var resp struct {
		Items []any `json:"items"`
		Total int   `json:"total"`
	}
	decodeJSON(t, w, &resp)
	if resp.Total != 0 {
		t.Errorf("total: got %d, want 0", resp.Total)
	}
}

func TestModules_GET_WithItems(t *testing.T) {
	svc := &mockSvc{listModules: func(string) ([]*model.Model3D, error) {
		return []*model.Model3D{
			{ID: "m1", Name: "Module A", Status: model.ModuleDraft},
			{ID: "m2", Name: "Module B", Status: model.ModuleSubmitted},
		}, nil
	}}
	w := do(t, newTestMux(t, svc), http.MethodGet, "/api/modules", "")
	var resp struct{ Total int `json:"total"` }
	decodeJSON(t, w, &resp)
	if resp.Total != 2 {
		t.Errorf("total: got %d, want 2", resp.Total)
	}
}

func TestModules_POST_Created(t *testing.T) {
	w := do(t, newTestMux(t, &mockSvc{}), http.MethodPost, "/api/modules",
		`{"name":"Assemblage A","owner_id":"user1"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("got %d, want 201 (body: %s)", w.Code, w.Body.String())
	}
}

func TestModules_POST_MissingName(t *testing.T) {
	w := do(t, newTestMux(t, &mockSvc{}), http.MethodPost, "/api/modules", `{"owner_id":"user1"}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("got %d, want 400", w.Code)
	}
}

func TestModules_MethodNotAllowed(t *testing.T) {
	w := do(t, newTestMux(t, &mockSvc{}), http.MethodPut, "/api/modules", "")
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("got %d, want 405", w.Code)
	}
}

// ── /api/modules/:id ─────────────────────────────────────────────────────────

func TestModule_GET_Found(t *testing.T) {
	w := do(t, newTestMux(t, &mockSvc{}), http.MethodGet, "/api/modules/m1", "")
	if w.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", w.Code)
	}
	var dto struct{ ID string `json:"id"` }
	decodeJSON(t, w, &dto)
	if dto.ID != "m1" {
		t.Errorf("ID: got %q, want %q", dto.ID, "m1")
	}
}

func TestModule_GET_NotFound(t *testing.T) {
	svc := &mockSvc{getModule: func(id string) (*model.Model3D, error) {
		return nil, errors.New("not found")
	}}
	w := do(t, newTestMux(t, svc), http.MethodGet, "/api/modules/xyz", "")
	if w.Code != http.StatusNotFound {
		t.Errorf("got %d, want 404", w.Code)
	}
}

func TestModule_DELETE(t *testing.T) {
	w := do(t, newTestMux(t, &mockSvc{}), http.MethodDelete, "/api/modules/m1", "")
	if w.Code != http.StatusNoContent {
		t.Errorf("got %d, want 204", w.Code)
	}
}

func TestModule_Submit(t *testing.T) {
	w := do(t, newTestMux(t, &mockSvc{}), http.MethodPost, "/api/modules/m1/submit",
		`{"note":"v1.0 release"}`)
	if w.Code != http.StatusOK {
		t.Errorf("got %d, want 200 (body: %s)", w.Code, w.Body.String())
	}
}

func TestModule_AddAssembly(t *testing.T) {
	w := do(t, newTestMux(t, &mockSvc{}), http.MethodPost, "/api/modules/m1/assemblies",
		`{"connection_id":"c1"}`)
	if w.Code != http.StatusOK {
		t.Errorf("got %d, want 200 (body: %s)", w.Code, w.Body.String())
	}
}

func TestModule_AddAssembly_MissingConnectionID(t *testing.T) {
	w := do(t, newTestMux(t, &mockSvc{}), http.MethodPost, "/api/modules/m1/assemblies", `{}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("got %d, want 400", w.Code)
	}
}

func TestModule_AddAssetToWorkspace(t *testing.T) {
	w := do(t, newTestMux(t, &mockSvc{}), http.MethodPost, "/api/modules/m1/instances",
		`{"asset_id":"a1"}`)
	if w.Code != http.StatusOK {
		t.Errorf("got %d, want 200 (body: %s)", w.Code, w.Body.String())
	}
}

func TestModule_ListInstances(t *testing.T) {
	svc := &mockSvc{
		getModule: func(id string) (*model.Model3D, error) {
			return &model.Model3D{ID: id, Status: model.ModuleDraft, WorkspaceInstances: []model.WorkspaceInstance{
				{ID: "inst1", AssetID: "a1"},
			}}, nil
		},
	}
	w := do(t, newTestMux(t, svc), http.MethodGet, "/api/modules/m1/instances", "")
	if w.Code != http.StatusOK {
		t.Errorf("got %d, want 200 (body: %s)", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"instances"`) {
		t.Errorf("réponse doit contenir le champ 'instances' : %s", w.Body.String())
	}
}

func TestModule_AddTwoAssetsSequentially(t *testing.T) {
	var calls []string
	svc := &mockSvc{
		addAssetToWorkspace: func(moduleID, assetID string) (*model.Model3D, error) {
			calls = append(calls, assetID)
			insts := make([]model.WorkspaceInstance, len(calls))
			for i, id := range calls {
				insts[i] = model.WorkspaceInstance{ID: "inst" + id, AssetID: id}
			}
			return &model.Model3D{ID: moduleID, Status: model.ModuleDraft, WorkspaceInstances: insts}, nil
		},
	}
	mux := newTestMux(t, svc)

	for _, assetID := range []string{"asset1", "asset2"} {
		w := do(t, mux, http.MethodPost, "/api/modules/m1/instances", `{"asset_id":"`+assetID+`"}`)
		if w.Code != http.StatusOK {
			t.Fatalf("ajout %s : got %d, want 200 (body: %s)", assetID, w.Code, w.Body.String())
		}
	}

	if len(calls) != 2 {
		t.Fatalf("AddAssetToWorkspace doit être appelé 2 fois, appelé %d fois", len(calls))
	}
	if calls[0] != "asset1" || calls[1] != "asset2" {
		t.Errorf("ordre des appels incorrect : got %v", calls)
	}
}

func TestModule_AddAssetToWorkspace_MissingAssetID(t *testing.T) {
	w := do(t, newTestMux(t, &mockSvc{}), http.MethodPost, "/api/modules/m1/instances", `{}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("got %d, want 400 (asset_id manquant)", w.Code)
	}
}

func TestModule_RemoveAssetFromWorkspace(t *testing.T) {
	w := do(t, newTestMux(t, &mockSvc{}), http.MethodDelete, "/api/modules/m1/instances/inst1", "")
	if w.Code != http.StatusOK {
		t.Errorf("got %d, want 200 (body: %s)", w.Code, w.Body.String())
	}
}

func TestModule_UpdateInstancePosition(t *testing.T) {
	w := do(t, newTestMux(t, &mockSvc{}), http.MethodPatch, "/api/modules/m1/instances/inst1",
		`{"x":10.5,"y":20.0}`)
	if w.Code != http.StatusOK {
		t.Errorf("got %d, want 200 (body: %s)", w.Code, w.Body.String())
	}
}

// ── /api/refs ─────────────────────────────────────────────────────────────────

func TestRefs_GET(t *testing.T) {
	w := do(t, newTestMux(t, &mockSvc{}), http.MethodGet, "/api/refs", "")
	if w.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", w.Code)
	}
	var refs model.InterfaceRefs
	decodeJSON(t, w, &refs)
	if len(refs.Categories) == 0 {
		t.Error("expected non-empty categories")
	}
}

func TestRefs_MethodNotAllowed(t *testing.T) {
	w := do(t, newTestMux(t, &mockSvc{}), http.MethodPost, "/api/refs", "")
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("got %d, want 405", w.Code)
	}
}

func TestRefCategories_POST_Created(t *testing.T) {
	w := do(t, newTestMux(t, &mockSvc{}), http.MethodPost, "/api/refs/categories", `{"name":"PNEUM"}`)
	if w.Code != http.StatusCreated {
		t.Errorf("got %d, want 201", w.Code)
	}
}

func TestRefCategories_POST_MissingName(t *testing.T) {
	w := do(t, newTestMux(t, &mockSvc{}), http.MethodPost, "/api/refs/categories", `{}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("got %d, want 400", w.Code)
	}
}

func TestRefTypes_POST_Created(t *testing.T) {
	w := do(t, newTestMux(t, &mockSvc{}), http.MethodPost, "/api/refs/types",
		`{"category":"ELEC","name":"SMA"}`)
	if w.Code != http.StatusCreated {
		t.Errorf("got %d, want 201", w.Code)
	}
}

func TestRefTypes_POST_MissingCategory(t *testing.T) {
	w := do(t, newTestMux(t, &mockSvc{}), http.MethodPost, "/api/refs/types", `{"name":"SMA"}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("got %d, want 400", w.Code)
	}
}

func TestRefUnits_POST_Created(t *testing.T) {
	w := do(t, newTestMux(t, &mockSvc{}), http.MethodPost, "/api/refs/units",
		`{"category":"HYD","name":"L/s"}`)
	if w.Code != http.StatusCreated {
		t.Errorf("got %d, want 201", w.Code)
	}
}

func TestRefUnits_POST_MissingFields(t *testing.T) {
	w := do(t, newTestMux(t, &mockSvc{}), http.MethodPost, "/api/refs/units", `{"category":"HYD"}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("got %d, want 400", w.Code)
	}
}

// ── Absence de token : toujours refusée, quel que soit le réseau actif ────────

/// @brief  Vérifie que POST /api/refs/types sans en-tête X-Myr-Token est refusé
/// @input  POST /api/refs/types, aucun token
/// @expect HTTP 401
func TestRefTypes_NoToken_Unauthorized(t *testing.T) {
	mux := newTestMux(t, &mockSvc{})
	r := newRequest(http.MethodPost, "/api/refs/types", `{"category":"ELEC","name":"SMA"}`)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("got %d, want 401 (token requis)", w.Code)
	}
}

func TestRefTypes_ResponseContainsNewType(t *testing.T) {
	var calledWith string
	svc := &mockSvc{
		addRefType: func(cat, typeName string) error {
			calledWith = cat + ":" + typeName
			return nil
		},
		getRefs: func() (*model.InterfaceRefs, error) {
			refs := model.DefaultInterfaceRefs()
			refs.Types["ELEC"] = append(refs.Types["ELEC"], "SMA")
			return refs, nil
		},
	}
	w := do(t, newTestMux(t, svc), http.MethodPost, "/api/refs/types",
		`{"category":"ELEC","name":"SMA"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("got %d, want 201", w.Code)
	}
	if calledWith != "ELEC:SMA" {
		t.Errorf("expected AddRefType(ELEC, SMA), got %q", calledWith)
	}
	var refs model.InterfaceRefs
	if err := json.NewDecoder(w.Body).Decode(&refs); err != nil {
		t.Fatalf("decode refs body: %v", err)
	}
	found := false
	for _, tp := range refs.Types["ELEC"] {
		if tp == "SMA" {
			found = true
		}
	}
	if !found {
		t.Error("response refs should contain the newly added type SMA under ELEC")
	}
}

// ── /api/licenses ─────────────────────────────────────────────────────────────

func TestLicenses_GET_List(t *testing.T) {
	w := do(t, newTestMux(t, &mockSvc{}), http.MethodGet, "/api/licenses", "")
	if w.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", w.Code)
	}
	var licenses []*model.License
	decodeJSON(t, w, &licenses)
	if len(licenses) == 0 {
		t.Error("expected at least one license")
	}
}

func TestLicenses_MethodNotAllowed(t *testing.T) {
	w := do(t, newTestMux(t, &mockSvc{}), http.MethodPost, "/api/licenses", "")
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("got %d, want 405", w.Code)
	}
}

func TestLicense_GET_ByID(t *testing.T) {
	w := do(t, newTestMux(t, &mockSvc{}), http.MethodGet, "/api/licenses/mit", "")
	if w.Code != http.StatusOK {
		t.Fatalf("got %d, want 200 (body: %s)", w.Code, w.Body.String())
	}
	var l model.License
	decodeJSON(t, w, &l)
	if l.ID != "mit" {
		t.Errorf("ID: got %q, want %q", l.ID, "mit")
	}
}

func TestLicense_GET_NotFound(t *testing.T) {
	svc := &mockSvc{getLicense: func(id string) (*model.License, error) {
		return nil, errors.New("not found")
	}}
	w := do(t, newTestMux(t, svc), http.MethodGet, "/api/licenses/unknown", "")
	if w.Code != http.StatusNotFound {
		t.Errorf("got %d, want 404", w.Code)
	}
}

func TestLicense_Check_Compatible(t *testing.T) {
	w := do(t, newTestMux(t, &mockSvc{}), http.MethodPost, "/api/licenses/check",
		`{"parent_license_id":"mit","proposed_license_id":"apache"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("got %d, want 200 (body: %s)", w.Code, w.Body.String())
	}
	var result model.LicenseCheck
	decodeJSON(t, w, &result)
	if !result.Compatible {
		t.Error("expected compatible=true")
	}
}

func TestLicense_Check_BadJSON(t *testing.T) {
	w := do(t, newTestMux(t, &mockSvc{}), http.MethodPost, "/api/licenses/check", "{bad")
	if w.Code != http.StatusBadRequest {
		t.Errorf("got %d, want 400", w.Code)
	}
}

func TestLicense_CheckProduct(t *testing.T) {
	w := do(t, newTestMux(t, &mockSvc{}), http.MethodPost, "/api/licenses/check-product",
		`{"component_license_ids":["mit","apache"],"proposed_module_license_id":"mit"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("got %d, want 200 (body: %s)", w.Code, w.Body.String())
	}
}

// ── /api/ping ─────────────────────────────────────────────────────────────────

/// @brief  GET /api/ping retourne 200 avec ok=true
/// @input  GET /api/ping
/// @expect HTTP 200, {"ok":true}
func TestPing_OK(t *testing.T) {
	w := do(t, newTestMux(t, &mockSvc{}), http.MethodGet, "/api/ping", "")
	if w.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", w.Code)
	}
	var resp struct{ Ok bool `json:"ok"` }
	decodeJSON(t, w, &resp)
	if !resp.Ok {
		t.Error("attendu ok=true")
	}
}

// ── /api/components?parent_id= ───────────────────────────────────────────────

/// @brief  GET /api/components?parent_id= filtre les composants par ParentID
/// @input  GET /api/components?parent_id=parent1, liste de 4 assets dont 2 enfants de parent1
/// @expect HTTP 200, total=2
func TestListGraph_FilterByParentID(t *testing.T) {
	svc := &mockSvc{list: func(string) ([]*model.Model3D, error) {
		return []*model.Model3D{
			{ID: "parent1", Name: "Parent"},
			{ID: "child1", Name: "Variante A", Category: model.CategoryVariation, ParentID: "parent1"},
			{ID: "child2", Name: "Variante B", Category: model.CategoryVariation, ParentID: "parent1"},
			{ID: "other", Name: "Autre", ParentID: "parent2"},
		}, nil
	}}
	w := do(t, newTestMux(t, svc), http.MethodGet, "/api/components?parent_id=parent1", "")
	if w.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", w.Code)
	}
	var resp struct{ Total int `json:"total"` }
	decodeJSON(t, w, &resp)
	if resp.Total != 2 {
		t.Errorf("filter parent_id=parent1: got %d, want 2", resp.Total)
	}
}

// ── /api/components/:id/tree ──────────────────────────────────────────────────

/// @brief  GET /api/components/:id/tree retourne les nœuds de l'arbre d'évolution
/// @input  GET /api/components/root/tree, liste avec root + 1 enfant
/// @expect HTTP 200, clé "components" présente, total≥1
func TestComponentTree_GET(t *testing.T) {
	svc := &mockSvc{list: func(string) ([]*model.Model3D, error) {
		return []*model.Model3D{
			{ID: "root", Name: "Root"},
			{ID: "child1", Name: "Variante", Category: model.CategoryVariation, ParentID: "root"},
		}, nil
	}}
	w := do(t, newTestMux(t, svc), http.MethodGet, "/api/components/root/tree", "")
	if w.Code != http.StatusOK {
		t.Fatalf("got %d, want 200 (body: %s)", w.Code, w.Body.String())
	}
	var resp struct {
		Components []any `json:"components"`
		Total      int   `json:"total"`
	}
	decodeJSON(t, w, &resp)
	if resp.Total < 1 {
		t.Errorf("total: got %d, want ≥1", resp.Total)
	}
}

/// @brief  POST /api/components/:id/tree retourne 405
/// @input  POST /api/components/abc/tree
/// @expect HTTP 405
func TestComponentTree_MethodNotAllowed(t *testing.T) {
	w := do(t, newTestMux(t, &mockSvc{}), http.MethodPost, "/api/components/abc/tree", "")
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("got %d, want 405", w.Code)
	}
}

// ── /api/channels GET ─────────────────────────────────────────────────────────

/// @brief  GET /api/channels retourne la liste des canaux et le canal actif
/// @input  GET /api/channels, aucun canal configuré
/// @expect HTTP 200, clés "channels" et "active" présentes
func TestChannels_GET(t *testing.T) {
	w := do(t, newTestMux(t, &mockSvc{}), http.MethodGet, "/api/channels", "")
	if w.Code != http.StatusOK {
		t.Fatalf("got %d, want 200 (body: %s)", w.Code, w.Body.String())
	}
	var resp struct {
		Channels []string `json:"channels"`
		Active   string   `json:"active"`
	}
	decodeJSON(t, w, &resp)
	// Sans canal configuré, la réponse doit quand même être valide JSON.
	_ = resp
}
