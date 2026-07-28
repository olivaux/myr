// domain/identity/tests/service_test.go
package identity_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"myr-core/domain/identity"
)

type stubCA struct {
	registerFn      func(ctx context.Context, req identity.RegisterRequest) (string, error)
	enrollFn        func(ctx context.Context, name, secret string) (string, string, string, error)
	getStatusFn     func(ctx context.Context, name string) (string, error)
	reEnrollFn      func(ctx context.Context, name, certPEM, keyPEM string) (string, error)
	updateAttrsFn   func(ctx context.Context, name string, attrs map[string]string) error
	lastRegisterReq identity.RegisterRequest
	lastEnrollName  string
	lastStatusName  string
	lastUpdateName  string
	lastUpdateAttrs map[string]string
}

func (s *stubCA) Register(ctx context.Context, req identity.RegisterRequest) (string, error) {
	s.lastRegisterReq = req
	if s.registerFn != nil {
		return s.registerFn(ctx, req)
	}
	return "secret123", nil
}
func (s *stubCA) Enroll(ctx context.Context, name, secret string) (string, string, string, error) {
	s.lastEnrollName = name
	if s.enrollFn != nil {
		return s.enrollFn(ctx, name, secret)
	}
	return fakeCertPEM, fakeKeyPEM, fakeCACertPEM, nil
}
func (s *stubCA) GetStatus(ctx context.Context, name string) (string, error) {
	s.lastStatusName = name
	if s.getStatusFn != nil {
		return s.getStatusFn(ctx, name)
	}
	return identity.StatusActive, nil
}
func (s *stubCA) ReEnroll(ctx context.Context, name, certPEM, keyPEM string) (string, error) {
	if s.reEnrollFn != nil {
		return s.reEnrollFn(ctx, name, certPEM, keyPEM)
	}
	return fakeCertPEM, nil
}
func (s *stubCA) UpdateAttributes(ctx context.Context, name string, attrs map[string]string) error {
	s.lastUpdateName = name
	s.lastUpdateAttrs = attrs
	if s.updateAttrsFn != nil {
		return s.updateAttrsFn(ctx, name, attrs)
	}
	return nil
}

var _ identity.CAPort = (*stubCA)(nil)

const fakeCertPEM = "-----BEGIN CERTIFICATE-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA0fake==\n-----END CERTIFICATE-----\n"
const fakeKeyPEM = "-----BEGIN EC PRIVATE KEY-----\nMHQCAQEEIFake/key/data==\n-----END EC PRIVATE KEY-----\n"
const fakeCACertPEM = "-----BEGIN CERTIFICATE-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAcafake==\n-----END CERTIFICATE-----\n"

// / @brief  Enregistrement échoue sans CA configurée
// / @input  service sans CA (nil), requête avec nom "alice@Org1"
// / @expect retourne une erreur indiquant l'absence de CA
func TestService_Register_NoCA(t *testing.T) {
	svc := identity.NewService(t.TempDir(), nil)
	_, err := svc.Register(context.Background(), identity.RegisterRequest{Name: "alice@Org1"})
	if err == nil {
		t.Fatal("erreur attendue sans CA")
	}
}

// / @brief  Enregistrement réussi d'une identité via la CA
// / @input  stubCA retournant "secret123", requête complète avec nom, orgID, email et pays
// / @expect retourne le secret "secret123" et transmet le nom exact à la CA
func TestService_Register_Success(t *testing.T) {
	ca := &stubCA{}
	svc := identity.NewService(t.TempDir(), ca)
	req := identity.RegisterRequest{Name: "alice@Org1", OrgID: "Org1MSP", DisplayName: "Alice", Email: "alice@example.com", Country: "FR"}
	secret, err := svc.Register(context.Background(), req)
	if err != nil {
		t.Fatalf("inattendu : %v", err)
	}
	if secret != "secret123" {
		t.Errorf("secret attendu %q, obtenu %q", "secret123", secret)
	}
	if ca.lastRegisterReq.Name != req.Name {
		t.Errorf("nom %q attendu, obtenu %q", req.Name, ca.lastRegisterReq.Name)
	}
}

// / @brief  Enregistrement propage l'erreur retournée par la CA
// / @input  stubCA dont registerFn retourne "CA: identite deja enregistree"
// / @expect retourne exactement l'erreur de la CA
func TestService_Register_CAError(t *testing.T) {
	expected := errors.New("CA: identite deja enregistree")
	ca := &stubCA{registerFn: func(_ context.Context, _ identity.RegisterRequest) (string, error) { return "", expected }}
	svc := identity.NewService(t.TempDir(), ca)
	_, err := svc.Register(context.Background(), identity.RegisterRequest{Name: "alice@Org1"})
	if !errors.Is(err, expected) {
		t.Errorf("erreur attendue %v, obtenu %v", expected, err)
	}
}

// / @brief  Enrôlement échoue sans CA configurée
// / @input  service sans CA (nil), identité "alice" avec secret "secret123"
// / @expect retourne une erreur indiquant l'absence de CA
func TestService_Enroll_NoCA(t *testing.T) {
	svc := identity.NewService(t.TempDir(), nil)
	_, err := svc.Enroll(context.Background(), "alice", "secret123", "Org1MSP")
	if err == nil {
		t.Fatal("erreur attendue sans CA")
	}
}

// / @brief  Enrôlement réussi avec écriture des fichiers MSP sur disque
// / @input  stubCA retournant des PEM valides, répertoire temporaire, identité "alice" dans "Org1MSP"
// / @expect retourne un WalletEntry avec handle "alice@Org1" et statut pending, fichiers cert/key/ca présents
func TestService_Enroll_Success(t *testing.T) {
	walletDir := t.TempDir()
	svc := identity.NewService(walletDir, &stubCA{})
	entry, err := svc.Enroll(context.Background(), "alice", "secret123", "Org1MSP")
	if err != nil {
		t.Fatalf("inattendu : %v", err)
	}
	if entry.Handle != "alice@Org1" {
		t.Errorf("handle attendu %q, obtenu %q", "alice@Org1", entry.Handle)
	}
	if entry.Status != identity.StatusPending {
		t.Errorf("statut attendu %q, obtenu %q", identity.StatusPending, entry.Status)
	}
	mspDir := filepath.Join(walletDir, "alice@Org1", "msp")
	for _, path := range []string{
		filepath.Join(mspDir, "signcerts", "cert.pem"),
		filepath.Join(mspDir, "keystore", "key.pem"),
		filepath.Join(mspDir, "cacerts", "ca.pem"),
	} {
		if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
			t.Errorf("fichier absent : %s", path)
		}
	}
}

// / @brief  Enrôlement propage l'erreur retournée par la CA
// / @input  stubCA dont enrollFn retourne "CA: secret invalide"
// / @expect retourne exactement l'erreur de la CA
func TestService_Enroll_CAError(t *testing.T) {
	expected := errors.New("CA: secret invalide")
	ca := &stubCA{enrollFn: func(_ context.Context, _, _ string) (string, string, string, error) { return "", "", "", expected }}
	svc := identity.NewService(t.TempDir(), ca)
	_, err := svc.Enroll(context.Background(), "alice", "wrong", "Org1MSP")
	if !errors.Is(err, expected) {
		t.Errorf("erreur attendue %v, obtenu %v", expected, err)
	}
}

// / @brief  Enrôlement authentifie auprès de la CA avec l'identifiant complet pseudo@org, pas le pseudo seul
// / @input  stubCA, appel Enroll(ctx, "alice", "secret123", "Org1MSP")
// / @expect la CA reçoit "alice@Org1" comme nom — l'identifiant sous lequel AutoRegister a créé l'identité
func TestService_Enroll_UsesFullHandleForCAAuth(t *testing.T) {
	ca := &stubCA{}
	svc := identity.NewService(t.TempDir(), ca)
	if _, err := svc.Enroll(context.Background(), "alice", "secret123", "Org1MSP"); err != nil {
		t.Fatalf("inattendu : %v", err)
	}
	if ca.lastEnrollName != "alice@Org1" {
		t.Errorf("nom transmis à la CA : got %q, want %q", ca.lastEnrollName, "alice@Org1")
	}
}

// / @brief  GetStatus échoue sans CA configurée
// / @input  service sans CA (nil), entrée de wallet {Name:"alice"}
// / @expect retourne une erreur indiquant l'absence de CA
func TestService_GetStatus_NoCA(t *testing.T) {
	svc := identity.NewService(t.TempDir(), nil)
	_, err := svc.GetStatus(context.Background(), identity.WalletEntry{Name: "alice"})
	if err == nil {
		t.Fatal("erreur attendue sans CA")
	}
}

// / @brief  GetStatus retourne le statut "active" renvoyé par la CA
// / @input  stubCA dont getStatusFn retourne StatusActive, entrée {Name:"alice"}
// / @expect retourne identity.StatusActive sans erreur
func TestService_GetStatus_ReturnsActiveAfterApproval(t *testing.T) {
	ca := &stubCA{getStatusFn: func(_ context.Context, _ string) (string, error) { return identity.StatusActive, nil }}
	svc := identity.NewService(t.TempDir(), ca)
	status, err := svc.GetStatus(context.Background(), identity.WalletEntry{Name: "alice"})
	if err != nil {
		t.Fatalf("inattendu : %v", err)
	}
	if status != identity.StatusActive {
		t.Errorf("attendu %q, obtenu %q", identity.StatusActive, status)
	}
}

// / @brief  GetStatus retourne le statut "pending" renvoyé par la CA
// / @input  stubCA dont getStatusFn retourne StatusPending, entrée {Name:"bob"}
// / @expect retourne identity.StatusPending sans erreur
func TestService_GetStatus_Pending(t *testing.T) {
	ca := &stubCA{getStatusFn: func(_ context.Context, _ string) (string, error) { return identity.StatusPending, nil }}
	svc := identity.NewService(t.TempDir(), ca)
	status, err := svc.GetStatus(context.Background(), identity.WalletEntry{Name: "bob"})
	if err != nil {
		t.Fatalf("inattendu : %v", err)
	}
	if status != identity.StatusPending {
		t.Errorf("attendu %q, obtenu %q", identity.StatusPending, status)
	}
}

// / @brief  GetStatus propage l'erreur retournée par la CA
// / @input  stubCA dont getStatusFn retourne "CA: identite introuvable", entrée {Name:"ghost"}
// / @expect retourne exactement l'erreur de la CA
func TestService_GetStatus_CAError(t *testing.T) {
	expected := errors.New("CA: identite introuvable")
	ca := &stubCA{getStatusFn: func(_ context.Context, _ string) (string, error) { return "", expected }}
	svc := identity.NewService(t.TempDir(), ca)
	_, err := svc.GetStatus(context.Background(), identity.WalletEntry{Name: "ghost"})
	if !errors.Is(err, expected) {
		t.Errorf("erreur attendue %v, obtenu %v", expected, err)
	}
}

// / @brief  GetStatus interroge la CA avec l'identifiant complet pseudo@org (wallet.Handle), pas wallet.Name seul
// / @input  stubCA, appel GetStatus avec WalletEntry{Handle:"alice@Org1", Name:"alice", OrgID:"Org1MSP"}
// / @expect la CA reçoit "alice@Org1" comme nom
func TestService_GetStatus_UsesHandleNotBareName(t *testing.T) {
	ca := &stubCA{}
	svc := identity.NewService(t.TempDir(), ca)
	_, err := svc.GetStatus(context.Background(), identity.WalletEntry{Handle: "alice@Org1", Name: "alice", OrgID: "Org1MSP"})
	if err != nil {
		t.Fatalf("inattendu : %v", err)
	}
	if ca.lastStatusName != "alice@Org1" {
		t.Errorf("nom transmis à la CA : got %q, want %q", ca.lastStatusName, "alice@Org1")
	}
}

// / @brief  Listage des wallets locaux sur un répertoire inexistant retourne nil
// / @input  service avec répertoire "/no/such/dir" inexistant
// / @expect retourne nil sans erreur
func TestService_ListLocalWallets_NonExistentDir(t *testing.T) {
	svc := identity.NewService("/no/such/dir", nil)
	wallets, err := svc.ListLocalWallets()
	if err != nil {
		t.Fatalf("inattendu : %v", err)
	}
	if wallets != nil {
		t.Errorf("nil attendu, obtenu %v", wallets)
	}
}

// / @brief  Listage des wallets locaux sur un répertoire vide retourne une liste vide
// / @input  service avec répertoire temporaire vide
// / @expect retourne une liste de longueur 0 sans erreur
func TestService_ListLocalWallets_EmptyDir(t *testing.T) {
	svc := identity.NewService(t.TempDir(), nil)
	wallets, err := svc.ListLocalWallets()
	if err != nil {
		t.Fatalf("inattendu : %v", err)
	}
	if len(wallets) != 0 {
		t.Errorf("0 wallet attendu, obtenu %d", len(wallets))
	}
}

// / @brief  Listage des wallets locaux retourne les entrées créées après enrôlement
// / @input  répertoire temporaire, deux enrôlements successifs "alice@Org1" et "bob@Org2"
// / @expect retourne 2 wallets avec les handles attendus
func TestService_ListLocalWallets_AfterEnroll(t *testing.T) {
	walletDir := t.TempDir()
	svc := identity.NewService(walletDir, &stubCA{})
	if _, err := svc.Enroll(context.Background(), "alice", "s1", "Org1MSP"); err != nil {
		t.Fatalf("enroll alice: %v", err)
	}
	if _, err := svc.Enroll(context.Background(), "bob", "s2", "Org2MSP"); err != nil {
		t.Fatalf("enroll bob: %v", err)
	}
	wallets, err := svc.ListLocalWallets()
	if err != nil {
		t.Fatalf("inattendu : %v", err)
	}
	if len(wallets) != 2 {
		t.Fatalf("2 wallets attendus, obtenu %d", len(wallets))
	}
	handles := map[string]bool{}
	for _, w := range wallets {
		handles[w.Handle] = true
	}
	for _, expected := range []string{"alice@Org1", "bob@Org2"} {
		if !handles[expected] {
			t.Errorf("wallet %q absent", expected)
		}
	}
}

// / @brief  Flux complet de demande de compte : enregistrement, enrôlement et suivi du statut
// / @details Simule le cycle de vie d'une identité depuis la demande jusqu'à l'activation
// / @input  stubCA simulant register/enroll/getStatus avec transition pending -> active, identité "carol@Org1"
// / @expect enrôlement réussi, statut pending puis active après changement de la CA
func TestAccountRequest_FullFlow(t *testing.T) {
	walletDir := t.TempDir()
	status := identity.StatusPending
	ca := &stubCA{
		registerFn: func(_ context.Context, _ identity.RegisterRequest) (string, error) { return "enrollment-secret", nil },
		enrollFn: func(_ context.Context, name, secret string) (string, string, string, error) {
			if secret != "enrollment-secret" {
				return "", "", "", errors.New("secret invalide")
			}
			return fakeCertPEM, fakeKeyPEM, fakeCACertPEM, nil
		},
		getStatusFn: func(_ context.Context, _ string) (string, error) { return status, nil },
	}
	svc := identity.NewService(walletDir, ca)
	req := identity.RegisterRequest{Name: "carol@Org1", OrgID: "Org1MSP", DisplayName: "Carol", Email: "carol@example.com", Country: "FR"}
	secret, err := svc.Register(context.Background(), req)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	entry, err := svc.Enroll(context.Background(), "carol", secret, "Org1MSP")
	if err != nil {
		t.Fatalf("Enroll: %v", err)
	}
	st, _ := svc.GetStatus(context.Background(), entry)
	if st != identity.StatusPending {
		t.Errorf("attendu pending, obtenu %q", st)
	}
	status = identity.StatusActive
	st, _ = svc.GetStatus(context.Background(), entry)
	if st != identity.StatusActive {
		t.Errorf("attendu active, obtenu %q", st)
	}
}

// ── Mock RequestStore ─────────────────────────────────────────────────────────

type mockRequestStore struct {
	requests []*identity.AccountRequest
	err      error
}

func (m *mockRequestStore) Save(req *identity.AccountRequest) error {
	if m.err != nil {
		return m.err
	}
	m.requests = append(m.requests, req)
	return nil
}

func (m *mockRequestStore) FindAll() ([]*identity.AccountRequest, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.requests, nil
}

// ── ReEnroll ──────────────────────────────────────────────────────────────────

// / @brief  ReEnroll échoue sans CA configurée
// / @input  service sans CA (nil), wallet {Name:"alice"}
// / @expect retourne une erreur indiquant l'absence de CA
func TestReEnroll_NoCA(t *testing.T) {
	svc := identity.NewService(t.TempDir(), nil)
	_, err := svc.ReEnroll(context.Background(), identity.WalletEntry{Name: "alice"})
	if err == nil {
		t.Fatal("erreur attendue sans CA")
	}
}

// / @brief  ReEnroll réussi renouvelle le certificat et l'écrit sur disque
// / @input  wallet enrôlé préalablement, stubCA retournant un nouveau PEM
// / @expect WalletEntry conserve le même handle, fichier cert.pem mis à jour
func TestReEnroll_Success(t *testing.T) {
	walletDir := t.TempDir()
	newCert := "-----BEGIN CERTIFICATE-----\nMIIBnewcert==\n-----END CERTIFICATE-----\n"
	ca := &stubCA{
		reEnrollFn: func(_ context.Context, _, _, _ string) (string, error) {
			return newCert, nil
		},
	}
	svc := identity.NewService(walletDir, ca)
	entry, err := svc.Enroll(context.Background(), "alice", "secret", "Org1MSP")
	if err != nil {
		t.Fatalf("Enroll préalable : %v", err)
	}

	updated, err := svc.ReEnroll(context.Background(), entry)
	if err != nil {
		t.Fatalf("ReEnroll : %v", err)
	}
	if updated.Handle != entry.Handle {
		t.Errorf("Handle: got %q, want %q", updated.Handle, entry.Handle)
	}
	data, err := os.ReadFile(filepath.Join(entry.MSPDir, "signcerts", "cert.pem"))
	if err != nil {
		t.Fatalf("lecture cert : %v", err)
	}
	if string(data) != newCert {
		t.Error("cert.pem non mis à jour après ReEnroll")
	}
}

// / @brief  ReEnroll propage l'erreur retournée par la CA
// / @input  stubCA dont reEnrollFn retourne "CA: cert invalide"
// / @expect retourne exactement l'erreur de la CA
func TestReEnroll_CAError(t *testing.T) {
	expected := errors.New("CA: cert invalide")
	walletDir := t.TempDir()
	ca := &stubCA{
		reEnrollFn: func(_ context.Context, _, _, _ string) (string, error) { return "", expected },
	}
	svc := identity.NewService(walletDir, ca)
	entry, _ := svc.Enroll(context.Background(), "alice", "secret", "Org1MSP")
	_, err := svc.ReEnroll(context.Background(), entry)
	if !errors.Is(err, expected) {
		t.Errorf("erreur attendue %v, obtenu %v", expected, err)
	}
}

// ── SetRole ───────────────────────────────────────────────────────────────────

// / @brief  SetRole refuse sans CA configuré
// / @input  service construit sans CAPort (nil)
// / @expect retourne une erreur explicite
func TestSetRole_NoCA(t *testing.T) {
	svc := identity.NewService(t.TempDir(), nil)
	err := svc.SetRole(context.Background(), "alice@org1", "contributor")
	if err == nil {
		t.Fatal("erreur attendue sans CA")
	}
}

// / @brief  SetRole transmet le nouvel attribut Myr.role à la CA
// / @input  stubCA, identité "alice@org1", nouveau rôle "contributor"
// / @expect stubCA.UpdateAttributes reçoit le nom et l'attribut Myr.role="contributor"
func TestSetRole_Success(t *testing.T) {
	ca := &stubCA{}
	svc := identity.NewService(t.TempDir(), ca)
	if err := svc.SetRole(context.Background(), "alice@org1", "contributor"); err != nil {
		t.Fatalf("SetRole : %v", err)
	}
	if ca.lastUpdateName != "alice@org1" {
		t.Errorf("nom transmis : got %q, want alice@org1", ca.lastUpdateName)
	}
	if ca.lastUpdateAttrs["Myr.role"] != "contributor" {
		t.Errorf("Myr.role : got %q, want contributor", ca.lastUpdateAttrs["Myr.role"])
	}
}

// / @brief  SetRole propage l'erreur retournée par la CA
// / @input  stubCA dont updateAttrsFn retourne "CA: identité introuvable"
// / @expect retourne exactement l'erreur de la CA
func TestSetRole_CAError(t *testing.T) {
	expected := errors.New("CA: identité introuvable")
	ca := &stubCA{updateAttrsFn: func(_ context.Context, _ string, _ map[string]string) error { return expected }}
	svc := identity.NewService(t.TempDir(), ca)
	err := svc.SetRole(context.Background(), "ghost@org1", "admin")
	if !errors.Is(err, expected) {
		t.Errorf("erreur attendue %v, obtenu %v", expected, err)
	}
}

// / @brief  SetRole refuse un nom d'identité vide
// / @input  stubCA valide, name=""
// / @expect retourne une erreur de validation sans appeler la CA
func TestSetRole_MissingName(t *testing.T) {
	ca := &stubCA{}
	svc := identity.NewService(t.TempDir(), ca)
	err := svc.SetRole(context.Background(), "", "admin")
	if err == nil {
		t.Fatal("erreur attendue pour identité vide")
	}
	if ca.lastUpdateName != "" {
		t.Error("la CA ne devrait pas être appelée")
	}
}

// ── SubmitRequest ─────────────────────────────────────────────────────────────

// / @brief  SubmitRequest crée une demande avec ID et statut pending, la persiste dans le store
// / @input  mockStore vide, demande {Pseudo:"bob", Email:"bob@example.com", OrgID:"Org1MSP"}
// / @expect retourne une demande avec ID non vide et Status="pending", store contient 1 entrée
func TestSubmitRequest_Success(t *testing.T) {
	store := &mockRequestStore{}
	svc := identity.NewService(t.TempDir(), nil).WithRequestStore(store)
	req := identity.AccountRequest{Pseudo: "bob", Email: "bob@example.com", OrgID: "Org1MSP", DisplayName: "Bob"}
	got, err := svc.SubmitRequest(req)
	if err != nil {
		t.Fatalf("SubmitRequest : %v", err)
	}
	if got.ID == "" {
		t.Error("ID doit être généré")
	}
	if got.Status != identity.RequestPending {
		t.Errorf("Status: got %q, want pending", got.Status)
	}
	if len(store.requests) != 1 {
		t.Errorf("store: attendu 1 demande, got %d", len(store.requests))
	}
}

// / @brief  SubmitRequest refusé si pseudo, email ou org_id est vide
// / @input  trois variantes de demande incomplète
// / @expect retourne une erreur de validation pour chaque cas
func TestSubmitRequest_MissingFields(t *testing.T) {
	svc := identity.NewService(t.TempDir(), nil)
	cases := []identity.AccountRequest{
		{Pseudo: "", Email: "x@x.com", OrgID: "Org1MSP"},
		{Pseudo: "alice", Email: "", OrgID: "Org1MSP"},
		{Pseudo: "alice", Email: "x@x.com", OrgID: ""},
	}
	for _, c := range cases {
		if _, err := svc.SubmitRequest(c); err == nil {
			t.Errorf("attendu erreur pour %+v", c)
		}
	}
}

// / @brief  SubmitRequest sans store génère quand même l'ID et retourne sans erreur
// / @input  service sans requestStore, demande valide
// / @expect retourne une demande avec ID généré, sans erreur
func TestSubmitRequest_WithoutStore(t *testing.T) {
	svc := identity.NewService(t.TempDir(), nil)
	req := identity.AccountRequest{Pseudo: "carol", Email: "carol@example.com", OrgID: "Org1MSP"}
	got, err := svc.SubmitRequest(req)
	if err != nil {
		t.Fatalf("SubmitRequest sans store : %v", err)
	}
	if got.ID == "" {
		t.Error("ID doit être généré même sans store")
	}
}

// ── AutoRegister ──────────────────────────────────────────────────────────────

// / @brief  AutoRegister délègue à la CA avec le bon nom pseudo@org
// / @input  stubCA retournant "secret123", demande {Pseudo:"dave", OrgID:"Org1MSP"}
// / @expect retourne "secret123", nom CA == "dave@Org1"
func TestAutoRegister_Success(t *testing.T) {
	ca := &stubCA{}
	svc := identity.NewService(t.TempDir(), ca)
	req := identity.AccountRequest{Pseudo: "dave", OrgID: "Org1MSP", DisplayName: "Dave", Email: "dave@example.com"}
	secret, err := svc.AutoRegister(context.Background(), req, identity.RoleContributor)
	if err != nil {
		t.Fatalf("AutoRegister : %v", err)
	}
	if secret != "secret123" {
		t.Errorf("secret: got %q, want secret123", secret)
	}
	if ca.lastRegisterReq.Name != "dave@Org1" {
		t.Errorf("nom CA attendu %q, obtenu %q", "dave@Org1", ca.lastRegisterReq.Name)
	}
}

// / @brief  AutoRegister échoue sans CA configurée
// / @input  service sans CA, demande valide
// / @expect retourne une erreur indiquant l'absence de CA
func TestAutoRegister_NoCA(t *testing.T) {
	svc := identity.NewService(t.TempDir(), nil)
	_, err := svc.AutoRegister(context.Background(),
		identity.AccountRequest{Pseudo: "dave", OrgID: "Org1MSP"}, "")
	if err == nil {
		t.Fatal("erreur attendue sans CA")
	}
}

// / @brief  AutoRegister transmet le rôle exact au RegisterRequest envoyé à la CA
// / @input  stubCA enregistrant la requête, rôle RoleContributor
// / @expect RegisterRequest.Role == RoleContributor
func TestAutoRegister_RoleForwarded(t *testing.T) {
	ca := &stubCA{}
	svc := identity.NewService(t.TempDir(), ca)
	svc.AutoRegister(context.Background(),
		identity.AccountRequest{Pseudo: "eve", OrgID: "Org1MSP"}, identity.RoleContributor)
	if ca.lastRegisterReq.Role != identity.RoleContributor {
		t.Errorf("rôle: got %q, want %q", ca.lastRegisterReq.Role, identity.RoleContributor)
	}
}

// ── ListRequests ──────────────────────────────────────────────────────────────

// / @brief  ListRequests retourne une liste vide si aucune demande n'est stockée
// / @input  mockStore vide
// / @expect retourne une liste de longueur 0 sans erreur
func TestListRequests_Empty(t *testing.T) {
	svc := identity.NewService(t.TempDir(), nil).WithRequestStore(&mockRequestStore{})
	reqs, err := svc.ListRequests()
	if err != nil {
		t.Fatalf("ListRequests : %v", err)
	}
	if len(reqs) != 0 {
		t.Errorf("attendu 0 demande, got %d", len(reqs))
	}
}

// / @brief  ListRequests retourne toutes les demandes enregistrées
// / @input  mockStore avec 2 demandes pré-chargées
// / @expect retourne une liste de 2 éléments sans erreur
func TestListRequests_WithRequests(t *testing.T) {
	store := &mockRequestStore{
		requests: []*identity.AccountRequest{
			{ID: "req-1", Pseudo: "alice", Status: identity.RequestPending},
			{ID: "req-2", Pseudo: "bob", Status: identity.RequestPending},
		},
	}
	svc := identity.NewService(t.TempDir(), nil).WithRequestStore(store)
	reqs, err := svc.ListRequests()
	if err != nil {
		t.Fatalf("ListRequests : %v", err)
	}
	if len(reqs) != 2 {
		t.Errorf("attendu 2 demandes, got %d", len(reqs))
	}
}

// / @brief  ListRequests retourne nil sans erreur si aucun store n'est configuré
// / @input  service sans requestStore
// / @expect retourne (nil, nil)
func TestListRequests_NoStore(t *testing.T) {
	svc := identity.NewService(t.TempDir(), nil)
	reqs, err := svc.ListRequests()
	if err != nil {
		t.Fatalf("ListRequests sans store : %v", err)
	}
	if reqs != nil {
		t.Error("attendu nil sans store")
	}
}

// ── LoadGuestWallet ───────────────────────────────────────────────────────────

// / @brief  LoadGuestWallet crée le wallet invité et les fichiers MSP sous walletDir
// / @input  fichiers cert.pem et key.pem valides, orgID "Org1MSP"
// / @expect handle "guest@Org1", statut active, fichier cert.pem présent dans MSP
func TestLoadGuestWallet_Success(t *testing.T) {
	certFile := filepath.Join(t.TempDir(), "cert.pem")
	keyFile := filepath.Join(t.TempDir(), "key.pem")
	if err := os.WriteFile(certFile, []byte(fakeCertPEM), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyFile, []byte(fakeKeyPEM), 0600); err != nil {
		t.Fatal(err)
	}

	walletDir := t.TempDir()
	svc := identity.NewService(walletDir, nil)
	entry, err := svc.LoadGuestWallet(certFile, keyFile, "", "Org1MSP")
	if err != nil {
		t.Fatalf("LoadGuestWallet : %v", err)
	}
	if entry.Handle != "guest@Org1" {
		t.Errorf("Handle: got %q, want guest@Org1", entry.Handle)
	}
	if entry.Status != identity.StatusActive {
		t.Errorf("Status: got %q, want active", entry.Status)
	}
	certPath := filepath.Join(walletDir, "guest@Org1", "msp", "signcerts", "cert.pem")
	if _, statErr := os.Stat(certPath); os.IsNotExist(statErr) {
		t.Error("cert.pem absent dans le wallet invité")
	}
}

// / @brief  LoadGuestWallet échoue si le fichier certificat est introuvable
// / @input  chemins inexistants pour cert et key
// / @expect retourne une erreur de lecture de fichier
func TestLoadGuestWallet_MissingCert(t *testing.T) {
	svc := identity.NewService(t.TempDir(), nil)
	_, err := svc.LoadGuestWallet("/no/such/cert.pem", "/no/such/key.pem", "", "Org1MSP")
	if err == nil {
		t.Fatal("erreur attendue pour certificat introuvable")
	}
}
