// adapters/in/rest/tests/handlers_identity_test.go — tests externes des handlers d'identité
package rest_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"myr-core/adapters/in/rest"
	"myr-core/domain/identity"
	"myr-core/domain/network"
)

// ── Mock IdentityService ──────────────────────────────────────────────────────

type mockIdentitySvc struct {
	listWallets     func() ([]identity.WalletEntry, error)
	enroll          func(ctx context.Context, name, secret, orgID string) (identity.WalletEntry, error)
	getStatus       func(ctx context.Context, wallet identity.WalletEntry) (string, error)
	reEnroll        func(ctx context.Context, wallet identity.WalletEntry) (identity.WalletEntry, error)
	register        func(ctx context.Context, req identity.RegisterRequest) (string, error)
	autoRegister    func(ctx context.Context, req identity.AccountRequest, role string) (string, error)
	loadGuestWallet func(certPath, keyPath, caCertPath, orgID string) (identity.WalletEntry, error)
	submitRequest   func(req identity.AccountRequest) (*identity.AccountRequest, error)
	listRequests    func() ([]*identity.AccountRequest, error)
	walletDir       func() string
	setRole         func(ctx context.Context, name, newRole string) error
}

func (m *mockIdentitySvc) ListLocalWallets() ([]identity.WalletEntry, error) {
	if m.listWallets != nil {
		return m.listWallets()
	}
	return nil, nil
}
func (m *mockIdentitySvc) Enroll(ctx context.Context, name, secret, orgID string) (identity.WalletEntry, error) {
	if m.enroll != nil {
		return m.enroll(ctx, name, secret, orgID)
	}
	return identity.WalletEntry{Handle: name + "@" + orgID, Name: name, OrgID: orgID, Status: identity.StatusPending}, nil
}
func (m *mockIdentitySvc) GetStatus(ctx context.Context, wallet identity.WalletEntry) (string, error) {
	if m.getStatus != nil {
		return m.getStatus(ctx, wallet)
	}
	return identity.StatusActive, nil
}
func (m *mockIdentitySvc) ReEnroll(ctx context.Context, wallet identity.WalletEntry) (identity.WalletEntry, error) {
	if m.reEnroll != nil {
		return m.reEnroll(ctx, wallet)
	}
	return wallet, nil
}
func (m *mockIdentitySvc) Register(ctx context.Context, req identity.RegisterRequest) (string, error) {
	if m.register != nil {
		return m.register(ctx, req)
	}
	return "secret", nil
}
func (m *mockIdentitySvc) AutoRegister(ctx context.Context, req identity.AccountRequest, role string) (string, error) {
	if m.autoRegister != nil {
		return m.autoRegister(ctx, req, role)
	}
	return "auto-secret", nil
}
func (m *mockIdentitySvc) SubmitRequest(req identity.AccountRequest) (*identity.AccountRequest, error) {
	if m.submitRequest != nil {
		return m.submitRequest(req)
	}
	req.ID = "req-test"
	req.Status = identity.RequestPending
	return &req, nil
}
func (m *mockIdentitySvc) ListRequests() ([]*identity.AccountRequest, error) {
	if m.listRequests != nil {
		return m.listRequests()
	}
	return nil, nil
}
func (m *mockIdentitySvc) LoadGuestWallet(certPath, keyPath, caCertPath, orgID string) (identity.WalletEntry, error) {
	if m.loadGuestWallet != nil {
		return m.loadGuestWallet(certPath, keyPath, caCertPath, orgID)
	}
	return identity.WalletEntry{Handle: "guest@" + orgID, Name: "guest", OrgID: orgID, Status: identity.StatusActive}, nil
}
func (m *mockIdentitySvc) WalletDir() string {
	if m.walletDir != nil {
		return m.walletDir()
	}
	return "/tmp/wallets"
}
func (m *mockIdentitySvc) SetRole(ctx context.Context, name, newRole string) error {
	if m.setRole != nil {
		return m.setRole(ctx, name, newRole)
	}
	return nil
}

var _ identity.IdentityService = (*mockIdentitySvc)(nil)

// ── Mock NetworkService ───────────────────────────────────────────────────────

type mockNetworkSvc struct {
	getActive func() (*network.NetworkProfile, error)
}

func (m *mockNetworkSvc) GetActive() (*network.NetworkProfile, error) {
	if m.getActive != nil {
		return m.getActive()
	}
	return nil, nil
}
func (m *mockNetworkSvc) Add(name, peerEndpoint, gatewayPeer, mspID, certPath, keyPath, tlsCertPath, fabricChannel, chaincodeName, caEndpoint, caName string, channels []string, allowAutoGuest, allowAutoRegister bool, autoRegisterRole string, isProduction bool) (*network.NetworkProfile, error) {
	return nil, nil
}
func (m *mockNetworkSvc) Update(id, name, peerEndpoint, gatewayPeer, mspID, certPath, keyPath, tlsCertPath, fabricChannel, chaincodeName, caEndpoint, caName string, allowAutoGuest, allowAutoRegister bool, autoRegisterRole string, isProduction bool) (*network.NetworkProfile, error) {
	return nil, nil
}
func (m *mockNetworkSvc) List() ([]*network.NetworkProfile, error) { return nil, nil }
func (m *mockNetworkSvc) Activate(id string) error                  { return nil }
func (m *mockNetworkSvc) Delete(id string) error                    { return nil }
func (m *mockNetworkSvc) TestConnection(id string) error            { return nil }
func (m *mockNetworkSvc) AddPeer(networkID string, req network.AddPeerRequest) (*network.PeerCredentials, error) {
	return nil, nil
}
func (m *mockNetworkSvc) Create(req network.CreateNetworkRequest) (*network.NetworkProfile, error) {
	return nil, nil
}

var _ network.NetworkService = (*mockNetworkSvc)(nil)

// ── helpers ───────────────────────────────────────────────────────────────────

// newIdentityMux builds a REST mux with an identity service — requireAuth
// always requires a token, obtainable via a real POST /api/identity/session.
func newIdentityMux(t *testing.T, idSvc identity.IdentityService) http.Handler {
	t.Helper()
	store := &mockStore{}
	h := rest.NewHandler(&mockSvc{}, store, store)
	h.WithIdentityService(idSvc)
	srv := rest.NewServer(h, "localhost:0")
	mux, err := srv.Handler()
	if err != nil {
		t.Fatalf("Handler(): %v", err)
	}
	return mux
}

// newBothMux builds a REST mux with identity + network services (blockchain mode).
func newBothMux(t *testing.T, idSvc identity.IdentityService, netSvc network.NetworkService) http.Handler {
	t.Helper()
	store := &mockStore{}
	h := rest.NewHandler(&mockSvc{}, store, store)
	h.WithIdentityService(idSvc)
	h.WithNetworkService(netSvc)
	srv := rest.NewServer(h, "localhost:0")
	mux, err := srv.Handler()
	if err != nil {
		t.Fatalf("Handler(): %v", err)
	}
	return mux
}

// ── GET /api/identity/wallets ─────────────────────────────────────────────────

func TestHandleIdentityWallets_Success(t *testing.T) {
	svc := &mockIdentitySvc{
		listWallets: func() ([]identity.WalletEntry, error) {
			return []identity.WalletEntry{
				{Handle: "alice@Org1", Name: "alice", OrgID: "Org1MSP", Status: identity.StatusActive},
				{Handle: "bob@Org2", Name: "bob", OrgID: "Org2MSP", Status: identity.StatusPending},
			}, nil
		},
	}
	w := do(t, newIdentityMux(t, svc), http.MethodGet, "/api/identity/wallets", "")
	if w.Code != http.StatusOK {
		t.Fatalf("code attendu 200, obtenu %d", w.Code)
	}
	var dtos []struct {
		Handle string `json:"handle"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(w.Body).Decode(&dtos); err != nil {
		t.Fatalf("JSON invalide : %v", err)
	}
	if len(dtos) != 2 {
		t.Fatalf("2 wallets attendus, obtenu %d", len(dtos))
	}
	if dtos[0].Handle != "alice@Org1" {
		t.Errorf("handle attendu %q, obtenu %q", "alice@Org1", dtos[0].Handle)
	}
}

func TestHandleIdentityWallets_EmptyList(t *testing.T) {
	w := do(t, newIdentityMux(t, &mockIdentitySvc{}), http.MethodGet, "/api/identity/wallets", "")
	if w.Code != http.StatusOK {
		t.Fatalf("code attendu 200, obtenu %d", w.Code)
	}
	var dtos []struct{ Handle string `json:"handle"` }
	json.NewDecoder(w.Body).Decode(&dtos)
	if len(dtos) != 0 {
		t.Errorf("liste vide attendue, obtenu %d éléments", len(dtos))
	}
}

func TestHandleIdentityWallets_NoService(t *testing.T) {
	// Sans identitySvc, aucune session ne peut être créée : requireAuth bloque
	// la requête (401) avant même d'atteindre le contrôle "identitySvc nil" du
	// handler — ce contrôle ne reste atteignable qu'avec une session déjà
	// valide (créée avant que l'identitySvc devienne indisponible).
	store := &mockStore{}
	h := rest.NewHandler(&mockSvc{}, store, store)
	srv := rest.NewServer(h, "localhost:0")
	mux, _ := srv.Handler()

	r := httptest.NewRequest(http.MethodGet, "/api/identity/wallets", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("code attendu 401, obtenu %d", w.Code)
	}
}

func TestHandleIdentityWallets_ServiceError(t *testing.T) {
	svc := &mockIdentitySvc{
		listWallets: func() ([]identity.WalletEntry, error) {
			return nil, errors.New("disque illisible")
		},
	}
	w := do(t, newIdentityMux(t, svc), http.MethodGet, "/api/identity/wallets", "")
	if w.Code != http.StatusInternalServerError {
		t.Errorf("code attendu 500, obtenu %d", w.Code)
	}
}

func TestHandleIdentityWallets_MethodNotAllowed(t *testing.T) {
	w := do(t, newIdentityMux(t, &mockIdentitySvc{}), http.MethodPost, "/api/identity/wallets", "")
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("code attendu 405, obtenu %d", w.Code)
	}
}

// ── POST /api/identity/enroll ─────────────────────────────────────────────────

func TestHandleIdentityEnroll_Success(t *testing.T) {
	var capturedName, capturedSecret, capturedOrg string
	svc := &mockIdentitySvc{
		enroll: func(_ context.Context, name, secret, orgID string) (identity.WalletEntry, error) {
			capturedName, capturedSecret, capturedOrg = name, secret, orgID
			return identity.WalletEntry{
				Handle: name + "@Org1",
				Name:   name,
				OrgID:  orgID,
				Status: identity.StatusPending,
			}, nil
		},
	}

	body, _ := json.Marshal(map[string]string{"name": "carol", "secret": "s3cr3t", "org_id": "Org1MSP"})
	req := httptest.NewRequest(http.MethodPost, "/api/identity/enroll", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	newIdentityMux(t, svc).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("code attendu 200, obtenu %d — body: %s", w.Code, w.Body.String())
	}
	var dto struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(w.Body).Decode(&dto); err != nil {
		t.Fatalf("JSON invalide : %v", err)
	}
	if dto.Status != identity.StatusPending {
		t.Errorf("statut attendu %q, obtenu %q", identity.StatusPending, dto.Status)
	}
	if capturedName != "carol" || capturedSecret != "s3cr3t" || capturedOrg != "Org1MSP" {
		t.Errorf("arguments transmis incorrects : name=%q secret=%q org=%q", capturedName, capturedSecret, capturedOrg)
	}
}

func TestHandleIdentityEnroll_MissingFields(t *testing.T) {
	mux := newIdentityMux(t, &mockIdentitySvc{})
	cases := []map[string]string{
		{"name": "", "secret": "x", "org_id": "Org1MSP"},
		{"name": "alice", "secret": "", "org_id": "Org1MSP"},
		{"name": "alice", "secret": "x", "org_id": ""},
	}
	for _, bodyMap := range cases {
		b, _ := json.Marshal(bodyMap)
		req := httptest.NewRequest(http.MethodPost, "/api/identity/enroll", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("code 400 attendu pour %v, obtenu %d", bodyMap, w.Code)
		}
	}
}

func TestHandleIdentityEnroll_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/identity/enroll", bytes.NewBufferString("{bad"))
	w := httptest.NewRecorder()
	newIdentityMux(t, &mockIdentitySvc{}).ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("code 400 attendu, obtenu %d", w.Code)
	}
}

func TestHandleIdentityEnroll_ServiceError(t *testing.T) {
	svc := &mockIdentitySvc{
		enroll: func(_ context.Context, _, _, _ string) (identity.WalletEntry, error) {
			return identity.WalletEntry{}, errors.New("CA: secret invalide")
		},
	}

	body, _ := json.Marshal(map[string]string{"name": "alice", "secret": "wrong", "org_id": "Org1MSP"})
	req := httptest.NewRequest(http.MethodPost, "/api/identity/enroll", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	newIdentityMux(t, svc).ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("code 400 attendu, obtenu %d", w.Code)
	}
}

func TestHandleIdentityEnroll_NoService(t *testing.T) {
	store := &mockStore{}
	h := rest.NewHandler(&mockSvc{}, store, store)
	srv := rest.NewServer(h, "localhost:0")
	mux, _ := srv.Handler()

	body, _ := json.Marshal(map[string]string{"name": "alice", "secret": "x", "org_id": "Org1MSP"})
	req := httptest.NewRequest(http.MethodPost, "/api/identity/enroll", bytes.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("code 503 attendu, obtenu %d", w.Code)
	}
}

// ── GET /api/identity/status ──────────────────────────────────────────────────

func TestHandleIdentityStatus_Active(t *testing.T) {
	var capturedHandle string
	svc := &mockIdentitySvc{
		getStatus: func(_ context.Context, we identity.WalletEntry) (string, error) {
			capturedHandle = we.Handle
			return identity.StatusActive, nil
		},
	}
	w := do(t, newIdentityMux(t, svc), http.MethodGet, "/api/identity/status?handle=alice@Org1", "")
	if w.Code != http.StatusOK {
		t.Fatalf("code attendu 200, obtenu %d", w.Code)
	}
	var res map[string]string
	json.NewDecoder(w.Body).Decode(&res)
	if res["status"] != identity.StatusActive {
		t.Errorf("statut attendu %q, obtenu %q", identity.StatusActive, res["status"])
	}
	if capturedHandle != "alice@Org1" {
		t.Errorf("handle attendu %q transmis au service, obtenu %q", "alice@Org1", capturedHandle)
	}
}

func TestHandleIdentityStatus_MissingHandle(t *testing.T) {
	w := do(t, newIdentityMux(t, &mockIdentitySvc{}), http.MethodGet, "/api/identity/status", "")
	if w.Code != http.StatusBadRequest {
		t.Errorf("code 400 attendu, obtenu %d", w.Code)
	}
}

func TestHandleIdentityStatus_InvalidHandle(t *testing.T) {
	w := do(t, newIdentityMux(t, &mockIdentitySvc{}), http.MethodGet, "/api/identity/status?handle=noslash", "")
	if w.Code != http.StatusBadRequest {
		t.Errorf("code 400 attendu, obtenu %d", w.Code)
	}
}

func TestHandleIdentityStatus_ServiceError(t *testing.T) {
	svc := &mockIdentitySvc{
		getStatus: func(_ context.Context, _ identity.WalletEntry) (string, error) {
			return "", errors.New("CA injoignable")
		},
	}
	w := do(t, newIdentityMux(t, svc), http.MethodGet, "/api/identity/status?handle=alice@Org1", "")
	if w.Code != http.StatusInternalServerError {
		t.Errorf("code 500 attendu, obtenu %d", w.Code)
	}
}

func TestHandleIdentityStatus_NoService(t *testing.T) {
	// Sans identitySvc, aucune session ne peut être créée : requireAuth bloque
	// la requête (401) avant même d'atteindre le contrôle "identitySvc nil" du
	// handler — voir TestHandleIdentityWallets_NoService pour le même raisonnement.
	store := &mockStore{}
	h := rest.NewHandler(&mockSvc{}, store, store)
	srv := rest.NewServer(h, "localhost:0")
	mux, _ := srv.Handler()

	r := httptest.NewRequest(http.MethodGet, "/api/identity/status?handle=alice@Org1", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("code 401 attendu, obtenu %d", w.Code)
	}
}

// ── GET /api/identity/policy ──────────────────────────────────────────────────

func TestHandleIdentityPolicy_BlockchainPrivate(t *testing.T) {
	netSvc := &mockNetworkSvc{
		getActive: func() (*network.NetworkProfile, error) {
			return &network.NetworkProfile{Name: "TestNet", MSPID: "Org1MSP", AllowAutoGuest: false}, nil
		},
	}
	w := do(t, newBothMux(t, &mockIdentitySvc{}, netSvc), http.MethodGet, "/api/identity/policy", "")
	var res struct {
		AllowAutoGuest bool   `json:"allow_auto_guest"`
		NetworkName    string `json:"network_name"`
	}
	json.NewDecoder(w.Body).Decode(&res)
	if res.AllowAutoGuest {
		t.Error("allow_auto_guest=false attendu pour un réseau privé")
	}
	if res.NetworkName != "TestNet" {
		t.Errorf("network_name attendu %q, obtenu %q", "TestNet", res.NetworkName)
	}
}

func TestHandleIdentityPolicy_BlockchainPublic(t *testing.T) {
	netSvc := &mockNetworkSvc{
		getActive: func() (*network.NetworkProfile, error) {
			return &network.NetworkProfile{Name: "PublicNet", AllowAutoGuest: true}, nil
		},
	}
	w := do(t, newBothMux(t, &mockIdentitySvc{}, netSvc), http.MethodGet, "/api/identity/policy", "")
	var res struct {
		AllowAutoGuest bool `json:"allow_auto_guest"`
	}
	json.NewDecoder(w.Body).Decode(&res)
	if !res.AllowAutoGuest {
		t.Error("allow_auto_guest=true attendu pour un réseau public")
	}
}

// ── POST /api/identity/guest ──────────────────────────────────────────────────

func TestHandleIdentityGuest_AllowedDeliversToken(t *testing.T) {
	netSvc := &mockNetworkSvc{
		getActive: func() (*network.NetworkProfile, error) {
			return &network.NetworkProfile{AllowAutoGuest: true, MSPID: "Org1MSP"}, nil
		},
	}

	body, _ := json.Marshal(map[string]string{"pseudo": "visiteur"})
	req := httptest.NewRequest(http.MethodPost, "/api/identity/guest", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	newBothMux(t, &mockIdentitySvc{}, netSvc).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("code 200 attendu, obtenu %d — body: %s", w.Code, w.Body.String())
	}
	var sess struct {
		Token string `json:"token"`
		Role  string `json:"role"`
		Guest bool   `json:"guest"`
	}
	json.NewDecoder(w.Body).Decode(&sess)
	if sess.Token == "" {
		t.Error("token attendu dans la réponse")
	}
	if sess.Role != "reader" {
		t.Errorf("role reader attendu, obtenu %q", sess.Role)
	}
	if !sess.Guest {
		t.Error("guest=true attendu")
	}
}

func TestHandleIdentityGuest_DisallowedReturnsForbidden(t *testing.T) {
	netSvc := &mockNetworkSvc{
		getActive: func() (*network.NetworkProfile, error) {
			return &network.NetworkProfile{AllowAutoGuest: false}, nil
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/identity/guest", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	newBothMux(t, &mockIdentitySvc{}, netSvc).ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("code 403 attendu, obtenu %d", w.Code)
	}
}

// ── requireAuth middleware ────────────────────────────────────────────────────

func TestRequireAuth_BlockchainWithoutToken(t *testing.T) {
	// Un accès sans token, quel que soit le réseau configuré, → 401.
	mux := newBlockchainMux(t, &mockSvc{})
	r := httptest.NewRequest(http.MethodGet, "/api/refs", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("code 401 attendu sans token, obtenu %d", w.Code)
	}
}

func TestRequireAuth_BlockchainWithValidToken(t *testing.T) {
	// Obtenir un token via /api/identity/session puis l'utiliser pour /api/refs.
	idSvc := &mockIdentitySvc{} // Enroll par défaut retourne succès
	store := &mockStore{}
	h := rest.NewHandler(&mockSvc{}, store, store)
	h.WithNetworkInfo(rest.NetworkInfo{Network: "peer0.example.com:7051", Channel: "mychannel"})
	h.WithIdentityService(idSvc)
	srv := rest.NewServer(h, "localhost:0")
	mux, err := srv.Handler()
	if err != nil {
		t.Fatal(err)
	}

	// Étape 1 : créer une session via /api/identity/session
	sessionBody, _ := json.Marshal(map[string]string{"name": "alice", "secret": "x", "org_id": "Org1MSP"})
	r := httptest.NewRequest(http.MethodPost, "/api/identity/session", bytes.NewReader(sessionBody))
	r.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	mux.ServeHTTP(w1, r)
	if w1.Code != http.StatusOK {
		t.Fatalf("création session: got %d, want 200 (body: %s)", w1.Code, w1.Body.String())
	}
	var sess struct{ Token string `json:"token"` }
	json.NewDecoder(w1.Body).Decode(&sess)

	// Étape 2 : accès à /api/refs avec token valide
	r2 := httptest.NewRequest(http.MethodGet, "/api/refs", nil)
	r2.Header.Set("X-Myr-Token", sess.Token)
	w2 := httptest.NewRecorder()
	mux.ServeHTTP(w2, r2)
	if w2.Code != http.StatusOK {
		t.Errorf("code 200 attendu avec token valide, obtenu %d", w2.Code)
	}
}

// ── POST /api/identity/request ────────────────────────────────────────────────

func TestHandleIdentityRequest_Success(t *testing.T) {
	var captured identity.AccountRequest
	svc := &mockIdentitySvc{
		submitRequest: func(req identity.AccountRequest) (*identity.AccountRequest, error) {
			captured = req
			req.ID = "req-001"
			req.Status = identity.RequestPending
			return &req, nil
		},
	}

	body, _ := json.Marshal(map[string]string{
		"pseudo": "bob", "email": "bob@exemple.com", "org_id": "Org1MSP", "message": "accès lecture",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/identity/request", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	newIdentityMux(t, svc).ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("code 201 attendu, obtenu %d — body: %s", w.Code, w.Body.String())
	}
	if captured.Pseudo != "bob" || captured.Email != "bob@exemple.com" || captured.OrgID != "Org1MSP" {
		t.Errorf("champs transmis incorrects : %+v", captured)
	}
	var resp identity.AccountRequest
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Status != identity.RequestPending {
		t.Errorf("statut attendu %q, obtenu %q", identity.RequestPending, resp.Status)
	}
}

func TestHandleIdentityRequest_MissingFields(t *testing.T) {
	mux := newIdentityMux(t, &mockIdentitySvc{})
	cases := []map[string]string{
		{"pseudo": "", "email": "a@b.com", "org_id": "Org1MSP"},
		{"pseudo": "bob", "email": "", "org_id": "Org1MSP"},
		{"pseudo": "bob", "email": "a@b.com", "org_id": ""},
	}
	for _, c := range cases {
		b, _ := json.Marshal(c)
		req := httptest.NewRequest(http.MethodPost, "/api/identity/request", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("code 400 attendu pour %v, obtenu %d", c, w.Code)
		}
	}
}

func TestHandleIdentityRequest_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/identity/request", bytes.NewBufferString("{bad"))
	w := httptest.NewRecorder()
	newIdentityMux(t, &mockIdentitySvc{}).ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("code 400 attendu, obtenu %d", w.Code)
	}
}

func TestHandleIdentityRequest_ServiceError(t *testing.T) {
	svc := &mockIdentitySvc{
		submitRequest: func(req identity.AccountRequest) (*identity.AccountRequest, error) {
			return nil, errors.New("disque plein")
		},
	}
	body, _ := json.Marshal(map[string]string{"pseudo": "bob", "email": "b@b.com", "org_id": "Org1MSP"})
	req := httptest.NewRequest(http.MethodPost, "/api/identity/request", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	newIdentityMux(t, svc).ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("code 500 attendu, obtenu %d", w.Code)
	}
}

// ── GET /api/identity/requests ────────────────────────────────────────────────

func TestHandleIdentityRequests_ReturnsList(t *testing.T) {
	svc := &mockIdentitySvc{
		listRequests: func() ([]*identity.AccountRequest, error) {
			return []*identity.AccountRequest{
				{ID: "req-1", Pseudo: "alice", Status: identity.RequestPending},
				{ID: "req-2", Pseudo: "bob", Status: identity.RequestApproved},
			}, nil
		},
	}
	w := do(t, newIdentityMux(t, svc), http.MethodGet, "/api/identity/requests", "")
	if w.Code != http.StatusOK {
		t.Fatalf("code 200 attendu, obtenu %d", w.Code)
	}
	var list []*identity.AccountRequest
	json.NewDecoder(w.Body).Decode(&list)
	if len(list) != 2 {
		t.Errorf("2 demandes attendues, obtenu %d", len(list))
	}
}

func TestHandleIdentityRequests_EmptyList(t *testing.T) {
	w := do(t, newIdentityMux(t, &mockIdentitySvc{}), http.MethodGet, "/api/identity/requests", "")
	if w.Code != http.StatusOK {
		t.Fatalf("code 200 attendu, obtenu %d", w.Code)
	}
	var list []*identity.AccountRequest
	json.NewDecoder(w.Body).Decode(&list)
	if len(list) != 0 {
		t.Errorf("liste vide attendue, obtenu %d", len(list))
	}
}

// ── POST /api/identity/session ────────────────────────────────────────────────

/// @brief  POST /api/identity/session crée une session et retourne un token
/// @input  service Enroll retourne succès, champs name/secret/org_id valides
/// @expect HTTP 200, token non vide, role="contributor"
func TestHandleIdentitySession_Success(t *testing.T) {
	svc := &mockIdentitySvc{}
	mux := newIdentityMux(t, svc)

	body, _ := json.Marshal(map[string]string{"name": "alice", "secret": "s3cr3t", "org_id": "Org1MSP"})
	r := httptest.NewRequest(http.MethodPost, "/api/identity/session", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("code 200 attendu, obtenu %d — body: %s", w.Code, w.Body.String())
	}
	var sess struct {
		Token string `json:"token"`
		Role  string `json:"role"`
	}
	if err := json.NewDecoder(w.Body).Decode(&sess); err != nil {
		t.Fatalf("JSON invalide : %v", err)
	}
	if sess.Token == "" {
		t.Error("token attendu dans la réponse")
	}
	if sess.Role != "contributor" {
		t.Errorf("role attendu contributor, obtenu %q", sess.Role)
	}
}

/// @brief  POST /api/identity/session retourne 400 si des champs obligatoires sont absents
/// @input  cas : name vide, secret vide, org_id vide
/// @expect HTTP 400 dans chaque cas
func TestHandleIdentitySession_MissingFields(t *testing.T) {
	mux := newIdentityMux(t, &mockIdentitySvc{})
	cases := []map[string]string{
		{"name": "", "secret": "x", "org_id": "Org1MSP"},
		{"name": "alice", "secret": "", "org_id": "Org1MSP"},
		{"name": "alice", "secret": "x", "org_id": ""},
	}
	for _, bodyMap := range cases {
		b, _ := json.Marshal(bodyMap)
		r := httptest.NewRequest(http.MethodPost, "/api/identity/session", bytes.NewReader(b))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != http.StatusBadRequest {
			t.Errorf("code 400 attendu pour %v, obtenu %d", bodyMap, w.Code)
		}
	}
}

/// @brief  POST /api/identity/session retourne 400 pour un corps JSON invalide
/// @input  corps "{bad"
/// @expect HTTP 400
func TestHandleIdentitySession_InvalidJSON(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/api/identity/session", bytes.NewBufferString("{bad"))
	w := httptest.NewRecorder()
	newIdentityMux(t, &mockIdentitySvc{}).ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("code 400 attendu, obtenu %d", w.Code)
	}
}

/// @brief  POST /api/identity/session retourne 503 si aucun service identité n'est configuré
/// @input  handler sans WithIdentityService, corps valide
/// @expect HTTP 503
func TestHandleIdentitySession_NoIdentityService(t *testing.T) {
	store := &mockStore{}
	h := rest.NewHandler(&mockSvc{}, store, store)
	srv := rest.NewServer(h, "localhost:0")
	mux, _ := srv.Handler()

	body, _ := json.Marshal(map[string]string{"name": "alice", "secret": "x", "org_id": "Org1MSP"})
	r := httptest.NewRequest(http.MethodPost, "/api/identity/session", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("code 503 attendu, obtenu %d", w.Code)
	}
}

/// @brief  GET /api/identity/session retourne 405 (seul POST est autorisé)
/// @input  GET /api/identity/session
/// @expect HTTP 405
func TestHandleIdentitySession_MethodNotAllowed(t *testing.T) {
	w := do(t, newIdentityMux(t, &mockIdentitySvc{}), http.MethodGet, "/api/identity/session", "")
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("code 405 attendu, obtenu %d", w.Code)
	}
}

/// @brief  POST /api/identity/session retourne 401 si Enroll échoue
/// @input  mockIdentitySvc.Enroll retourne une erreur "secret invalide"
/// @expect HTTP 401
func TestHandleIdentitySession_EnrollError(t *testing.T) {
	svc := &mockIdentitySvc{
		enroll: func(_ context.Context, _, _, _ string) (identity.WalletEntry, error) {
			return identity.WalletEntry{}, errors.New("secret invalide")
		},
	}
	body, _ := json.Marshal(map[string]string{"name": "alice", "secret": "wrong", "org_id": "Org1MSP"})
	r := httptest.NewRequest(http.MethodPost, "/api/identity/session", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	newIdentityMux(t, svc).ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("code 401 attendu, obtenu %d", w.Code)
	}
}

// ── /api/channels PUT ─────────────────────────────────────────────────────────

/// @brief  PUT /api/channels sans token retourne 401
/// @input  PUT /api/channels sans en-tête X-Myr-Token
/// @expect HTTP 401
func TestChannels_PUT_WithoutToken(t *testing.T) {
	r := httptest.NewRequest(http.MethodPut, "/api/channels", strings.NewReader(`{"channel":"sandbox"}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	newIdentityMux(t, &mockIdentitySvc{}).ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("code 401 attendu sans token, obtenu %d", w.Code)
	}
}

/// @brief  PUT /api/channels avec token valide change le canal de la session
/// @input  POST /api/identity/session pour obtenir un token, puis PUT /api/channels
/// @expect HTTP 200, channel="sandbox" dans la réponse
func TestChannels_PUT_SwitchesChannel(t *testing.T) {
	mux := newIdentityMux(t, &mockIdentitySvc{})

	// Étape 1 : créer une session
	sessionBody, _ := json.Marshal(map[string]string{"name": "alice", "secret": "x", "org_id": "Org1MSP"})
	r1 := httptest.NewRequest(http.MethodPost, "/api/identity/session", bytes.NewReader(sessionBody))
	r1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	mux.ServeHTTP(w1, r1)
	if w1.Code != http.StatusOK {
		t.Fatalf("création session: got %d, want 200 (body: %s)", w1.Code, w1.Body.String())
	}
	var sess struct{ Token string `json:"token"` }
	json.NewDecoder(w1.Body).Decode(&sess)
	if sess.Token == "" {
		t.Fatal("token absent dans la réponse de session")
	}

	// Étape 2 : changer le canal avec le token
	r2 := httptest.NewRequest(http.MethodPut, "/api/channels", strings.NewReader(`{"channel":"sandbox"}`))
	r2.Header.Set("Content-Type", "application/json")
	r2.Header.Set("X-Myr-Token", sess.Token)
	w2 := httptest.NewRecorder()
	mux.ServeHTTP(w2, r2)
	if w2.Code != http.StatusOK {
		t.Fatalf("PUT channels: got %d, want 200 (body: %s)", w2.Code, w2.Body.String())
	}
	var resp struct{ Channel string `json:"channel"` }
	json.NewDecoder(w2.Body).Decode(&resp)
	if resp.Channel != "sandbox" {
		t.Errorf("channel: got %q, want sandbox", resp.Channel)
	}
}
