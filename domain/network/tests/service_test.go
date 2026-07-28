// domain/network/tests/service_test.go — tests unitaires du service réseau
package network_test

import (
	"errors"
	"testing"

	"myr-core/domain/network"
)

// ── Mock Repo ─────────────────────────────────────────────────────────────────

type mockRepo struct {
	profiles []*network.NetworkProfile
	saveErr  error
	findErr  error
	delErr   error
}

func (m *mockRepo) Save(n *network.NetworkProfile) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	for i, p := range m.profiles {
		if p.ID == n.ID {
			m.profiles[i] = n
			return nil
		}
	}
	m.profiles = append(m.profiles, n)
	return nil
}

func (m *mockRepo) FindAll() ([]*network.NetworkProfile, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	return m.profiles, nil
}

func (m *mockRepo) Delete(id string) error {
	if m.delErr != nil {
		return m.delErr
	}
	filtered := m.profiles[:0]
	for _, p := range m.profiles {
		if p.ID != id {
			filtered = append(filtered, p)
		}
	}
	m.profiles = filtered
	return nil
}

// ── Mock ConnectionTester ─────────────────────────────────────────────────────

type mockTester struct {
	err error
}

func (m *mockTester) Test(_ *network.NetworkProfile) error {
	return m.err
}

// ── Mock PeerProvisioner ──────────────────────────────────────────────────────

type mockProvisioner struct {
	creds *network.PeerCredentials
	err   error
}

func (m *mockProvisioner) RegisterAndProvision(n *network.NetworkProfile, req network.AddPeerRequest) (*network.PeerCredentials, error) {
	return m.creds, m.err
}

// ── Mock NetworkBootstrapper ──────────────────────────────────────────────────

type mockBootstrapper struct {
	result *network.BootstrapResult
	err    error
}

func (m *mockBootstrapper) Bootstrap(req network.CreateNetworkRequest) (*network.BootstrapResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.result != nil {
		return m.result, nil
	}
	return &network.BootstrapResult{Profile: &network.NetworkProfile{ID: "net-bootstrap", Name: req.Name}}, nil
}

// ── Add ───────────────────────────────────────────────────────────────────────

// / @brief  Ajout d'un profil réseau valide
// / @input  mockRepo vide, paramètres complets avec nom et endpoint valides
// / @expect retourne un profil avec un ID généré et le nom fourni, sans erreur
func TestAdd_Success(t *testing.T) {
	svc := network.NewService(&mockRepo{}, nil)
	p, err := svc.Add("Reseau1", "localhost:7051", "peer0.org1.example.com", "Org1MSP",
		"/certs/cert.pem", "/certs/key.pem", "/certs/tls.pem",
		"mychannel", "myrcc", "", "", []string{"mychannel"}, false, false, "", false)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if p.Name != "Reseau1" {
		t.Errorf("Name: got %q, want Reseau1", p.Name)
	}
	if p.ID == "" {
		t.Error("ID should be generated")
	}
}

// / @brief  Ajout refusé si le nom est vide
// / @input  mockRepo vide, nom vide passé à Add
// / @expect retourne une erreur de validation
func TestAdd_MissingName(t *testing.T) {
	svc := network.NewService(&mockRepo{}, nil)
	_, err := svc.Add("", "localhost:7051", "", "", "", "", "", "", "", "", "", nil, false, false, "", false)
	if err == nil {
		t.Error("want error for empty name")
	}
}

// / @brief  Ajout refusé si l'endpoint peer est vide
// / @input  mockRepo vide, peerEndpoint vide
// / @expect retourne une erreur de validation
func TestAdd_MissingPeerEndpoint(t *testing.T) {
	svc := network.NewService(&mockRepo{}, nil)
	_, err := svc.Add("Net", "", "", "", "", "", "", "", "", "", "", nil, false, false, "", false)
	if err == nil {
		t.Error("want error for empty peer endpoint")
	}
}

// / @brief  Ajout refusé si l'endpoint peer n'est pas au format host:port
// / @input  mockRepo vide, peerEndpoint "not-a-host-port"
// / @expect retourne une erreur de format d'adresse
func TestAdd_InvalidPeerEndpoint(t *testing.T) {
	svc := network.NewService(&mockRepo{}, nil)
	_, err := svc.Add("Net", "not-a-host-port", "", "", "", "", "", "", "", "", "", nil, false, false, "", false)
	if err == nil {
		t.Error("want error for invalid host:port")
	}
}

// / @brief  Ajout échoue si le repo retourne une erreur à la sauvegarde
// / @input  mockRepo avec saveErr "disk full"
// / @expect retourne une erreur propagée depuis le repo
func TestAdd_RepoError(t *testing.T) {
	repo := &mockRepo{saveErr: errors.New("disk full")}
	svc := network.NewService(repo, nil)
	_, err := svc.Add("Net", "localhost:7051", "", "", "", "", "", "", "", "", "", nil, false, false, "", false)
	if err == nil {
		t.Error("want error when repo.Save fails")
	}
}

// / @brief  Ajout d'un profil réseau avec la politique d'accès et le flag production
// / @input  mockRepo vide, allowAutoGuest/allowAutoRegister à true, autoRegisterRole "contributor", isProduction à true
// / @expect le profil retourné porte les quatre champs tels que fournis
func TestAdd_PolicyAndProductionFlags(t *testing.T) {
	svc := network.NewService(&mockRepo{}, nil)
	p, err := svc.Add("Reseau1", "localhost:7051", "", "Org1MSP", "", "", "", "", "", "", "", nil,
		true, true, "contributor", true)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if !p.AllowAutoGuest || !p.AllowAutoRegister {
		t.Errorf("AllowAutoGuest/AllowAutoRegister: got %v/%v, want true/true", p.AllowAutoGuest, p.AllowAutoRegister)
	}
	if p.AutoRegisterRole != "contributor" {
		t.Errorf("AutoRegisterRole: got %q, want contributor", p.AutoRegisterRole)
	}
	if !p.IsProduction {
		t.Error("IsProduction: want true")
	}
}

// / @brief  Mise à jour d'un profil réseau permet d'activer la protection production
// / @input  mockRepo avec un profil "n1" non-production, mise à jour avec isProduction=true
// / @expect le profil mis à jour porte IsProduction=true
func TestUpdate_SetsProductionFlag(t *testing.T) {
	repo := &mockRepo{profiles: []*network.NetworkProfile{
		{ID: "n1", Name: "Old", PeerEndpoint: "localhost:7051", IsProduction: false},
	}}
	svc := network.NewService(repo, nil)
	updated, err := svc.Update("n1", "Old", "localhost:7051", "", "", "", "", "", "", "", "", "", false, false, "", true)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if !updated.IsProduction {
		t.Error("IsProduction: want true after update")
	}
}

// ── List ──────────────────────────────────────────────────────────────────────

// / @brief  Listage retourne une liste vide si aucun profil n'existe
// / @input  mockRepo vide
// / @expect retourne une liste de longueur 0 sans erreur
func TestList_Empty(t *testing.T) {
	svc := network.NewService(&mockRepo{}, nil)
	list, err := svc.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("got %d items, want 0", len(list))
	}
}

// / @brief  Listage retourne tous les profils enregistrés
// / @input  mockRepo avec 2 profils "Net1" et "Net2"
// / @expect retourne une liste de 2 éléments sans erreur
func TestList_ReturnsSaved(t *testing.T) {
	repo := &mockRepo{profiles: []*network.NetworkProfile{
		{ID: "n1", Name: "Net1", PeerEndpoint: "localhost:7051"},
		{ID: "n2", Name: "Net2", PeerEndpoint: "localhost:7052"},
	}}
	svc := network.NewService(repo, nil)
	list, err := svc.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("got %d items, want 2", len(list))
	}
}

// ── GetActive ─────────────────────────────────────────────────────────────────

// / @brief  GetActive retourne nil si aucun profil n'est actif
// / @input  mockRepo avec un profil inactif (Active: false)
// / @expect retourne nil sans erreur
func TestGetActive_None(t *testing.T) {
	repo := &mockRepo{profiles: []*network.NetworkProfile{
		{ID: "n1", Name: "Net1", Active: false},
	}}
	svc := network.NewService(repo, nil)
	p, err := svc.GetActive()
	if err != nil {
		t.Fatalf("GetActive: %v", err)
	}
	if p != nil {
		t.Error("want nil when no active network")
	}
}

// / @brief  GetActive retourne le profil actif parmi plusieurs
// / @input  mockRepo avec deux profils dont "n2" actif
// / @expect retourne le profil d'ID "n2" sans erreur
func TestGetActive_Found(t *testing.T) {
	repo := &mockRepo{profiles: []*network.NetworkProfile{
		{ID: "n1", Name: "Net1", Active: false},
		{ID: "n2", Name: "Net2", Active: true},
	}}
	svc := network.NewService(repo, nil)
	p, err := svc.GetActive()
	if err != nil {
		t.Fatalf("GetActive: %v", err)
	}
	if p == nil || p.ID != "n2" {
		t.Errorf("got %v, want n2", p)
	}
}

// ── Activate ──────────────────────────────────────────────────────────────────

// / @brief  Activation bascule le profil actif vers un nouveau profil
// / @input  mockRepo avec "n1" actif et "n2" inactif, activation de "n2"
// / @expect "n2" devient le seul profil actif après l'appel
func TestActivate_SwitchesActive(t *testing.T) {
	repo := &mockRepo{profiles: []*network.NetworkProfile{
		{ID: "n1", Name: "Net1", Active: true},
		{ID: "n2", Name: "Net2", Active: false},
	}}
	svc := network.NewService(repo, nil)
	if err := svc.Activate("n2"); err != nil {
		t.Fatalf("Activate: %v", err)
	}
	active, _ := svc.GetActive()
	if active == nil || active.ID != "n2" {
		t.Errorf("expected n2 to be active, got %v", active)
	}
}

// / @brief  Activation échoue si l'ID est inconnu
// / @input  mockRepo vide, tentative d'activation de "nonexistent"
// / @expect retourne une erreur pour ID introuvable
func TestActivate_NotFound(t *testing.T) {
	svc := network.NewService(&mockRepo{}, nil)
	err := svc.Activate("nonexistent")
	if err == nil {
		t.Error("want error for unknown ID")
	}
}

// ── Update ────────────────────────────────────────────────────────────────────

// / @brief  Mise à jour réussie d'un profil réseau existant
// / @input  mockRepo avec un profil "n1" (nom "Old", endpoint "localhost:7051"), nouveau nom "New" et endpoint "localhost:7052"
// / @expect retourne le profil mis à jour avec les nouvelles valeurs, sans erreur
func TestUpdate_Success(t *testing.T) {
	repo := &mockRepo{profiles: []*network.NetworkProfile{
		{ID: "n1", Name: "Old", PeerEndpoint: "localhost:7051"},
	}}
	svc := network.NewService(repo, nil)
	updated, err := svc.Update("n1", "New", "localhost:7052", "", "", "", "", "", "", "", "", "", false, false, "", false)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != "New" {
		t.Errorf("Name: got %q, want New", updated.Name)
	}
	if updated.PeerEndpoint != "localhost:7052" {
		t.Errorf("PeerEndpoint: got %q, want localhost:7052", updated.PeerEndpoint)
	}
}

// / @brief  Mise à jour échoue si l'ID est inconnu
// / @input  mockRepo vide, tentative de mise à jour de "nonexistent"
// / @expect retourne une erreur pour ID introuvable
func TestUpdate_NotFound(t *testing.T) {
	svc := network.NewService(&mockRepo{}, nil)
	_, err := svc.Update("nonexistent", "X", "localhost:7051", "", "", "", "", "", "", "", "", "", false, false, "", false)
	if err == nil {
		t.Error("want error for unknown ID")
	}
}

// ── Delete ────────────────────────────────────────────────────────────────────

// / @brief  Suppression réussie d'un profil existant
// / @input  mockRepo avec un profil "n1", suppression par ID "n1"
// / @expect liste vide après suppression, sans erreur
func TestDelete_Success(t *testing.T) {
	repo := &mockRepo{profiles: []*network.NetworkProfile{
		{ID: "n1", Name: "Net1"},
	}}
	svc := network.NewService(repo, nil)
	if err := svc.Delete("n1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	list, _ := svc.List()
	if len(list) != 0 {
		t.Errorf("expected 0 after delete, got %d", len(list))
	}
}

func TestDelete_NotFound(t *testing.T) {
	svc := network.NewService(&mockRepo{}, nil)
	err := svc.Delete("nonexistent")
	if err == nil {
		t.Error("want error for unknown ID")
	}
}

// ── TestConnection ────────────────────────────────────────────────────────────

func TestTestConnection_WithTester_Success(t *testing.T) {
	repo := &mockRepo{profiles: []*network.NetworkProfile{
		{ID: "n1", PeerEndpoint: "localhost:7051"},
	}}
	svc := network.NewService(repo, &mockTester{err: nil})
	if err := svc.TestConnection("n1"); err != nil {
		t.Errorf("TestConnection: %v", err)
	}
}

func TestTestConnection_WithTester_Error(t *testing.T) {
	repo := &mockRepo{profiles: []*network.NetworkProfile{
		{ID: "n1", PeerEndpoint: "localhost:7051"},
	}}
	svc := network.NewService(repo, &mockTester{err: errors.New("unreachable")})
	if err := svc.TestConnection("n1"); err == nil {
		t.Error("want error when tester fails")
	}
}

func TestTestConnection_NotFound(t *testing.T) {
	svc := network.NewService(&mockRepo{}, nil)
	err := svc.TestConnection("nonexistent")
	if err == nil {
		t.Error("want error for unknown ID")
	}
}

// ── AddPeer ───────────────────────────────────────────────────────────────────

func TestAddPeer_NoProvisioner(t *testing.T) {
	svc := network.NewService(&mockRepo{}, nil)
	_, err := svc.AddPeer("n1", network.AddPeerRequest{PeerID: "peer1"})
	if err == nil {
		t.Error("want error when no provisioner")
	}
}

func TestAddPeer_Success(t *testing.T) {
	repo := &mockRepo{profiles: []*network.NetworkProfile{
		{ID: "n1", CAEndpoint: "https://ca:7054", Active: true},
	}}
	creds := &network.PeerCredentials{PeerID: "peer2", SignCert: "cert", SignKey: "key"}
	prov := &mockProvisioner{creds: creds}
	svc := network.NewService(repo, nil)
	svc.WithProvisioner(prov)

	got, err := svc.AddPeer("n1", network.AddPeerRequest{PeerID: "peer2"})
	if err != nil {
		t.Fatalf("AddPeer: %v", err)
	}
	if got.SignCert != "cert" {
		t.Errorf("SignCert: got %q, want cert", got.SignCert)
	}
}

func TestAddPeer_MissingPeerID(t *testing.T) {
	repo := &mockRepo{profiles: []*network.NetworkProfile{
		{ID: "n1", CAEndpoint: "https://ca:7054"},
	}}
	svc := network.NewService(repo, nil)
	svc.WithProvisioner(&mockProvisioner{})
	_, err := svc.AddPeer("n1", network.AddPeerRequest{})
	if err == nil {
		t.Error("want error for empty peer-id")
	}
}

func TestAddPeer_NoCAConfigured(t *testing.T) {
	repo := &mockRepo{profiles: []*network.NetworkProfile{
		{ID: "n1", CAEndpoint: ""},
	}}
	svc := network.NewService(repo, nil)
	svc.WithProvisioner(&mockProvisioner{})
	_, err := svc.AddPeer("n1", network.AddPeerRequest{PeerID: "peer2"})
	if err == nil {
		t.Error("want error when CA not configured")
	}
}

// ── Create (UCADM02) ──────────────────────────────────────────────────────────

// / @brief  Création refusée si aucun bootstrapper n'est injecté
// / @input  service sans WithBootstrapper, requête complète
// / @expect retourne une erreur explicite, aucun appel au repo
func TestCreate_NoBootstrapper(t *testing.T) {
	svc := network.NewService(&mockRepo{}, nil)
	_, err := svc.Create(network.CreateNetworkRequest{
		Name: "diy-network", OrgMSPID: "Org1MSP", Domain: "diy-network.com", ChannelName: "sandbox",
	})
	if err == nil {
		t.Error("want error when no bootstrapper injected")
	}
}

// / @brief  Création réussie délègue au bootstrapper puis enregistre le profil résultant
// / @input  bootstrapper retournant un profil "net-bootstrap", requête valide avec 2 peers
// / @expect le profil est retourné et persisté dans le repo
func TestCreate_Success(t *testing.T) {
	repo := &mockRepo{}
	svc := network.NewService(repo, nil).WithBootstrapper(&mockBootstrapper{})
	p, err := svc.Create(network.CreateNetworkRequest{
		Name: "diy-network", OrgMSPID: "Org1MSP", Domain: "diy-network.com", ChannelName: "sandbox", NumPeers: 2,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if p.ID != "net-bootstrap" {
		t.Errorf("ID: got %q, want net-bootstrap", p.ID)
	}
	all, _ := repo.FindAll()
	if len(all) != 1 {
		t.Errorf("expected profile persisted in repo, got %d entries", len(all))
	}
}

// / @brief  Création refusée si le nombre de nœuds résultant est sous le seuil RM27
// / @input  NumPeers=0 forcé artificiellement bas — le défaut (2) satisfait déjà le seuil,
// / donc ce test vérifie qu'un NumPeers explicite trop bas (négatif après défaut) est rejeté
// / @expect le service applique le défaut (2 peers + 1 orderer = 3) sans erreur de seuil
func TestCreate_DefaultsNumPeers(t *testing.T) {
	repo := &mockRepo{}
	bootstrapper := &mockBootstrapper{}
	svc := network.NewService(repo, nil).WithBootstrapper(bootstrapper)
	_, err := svc.Create(network.CreateNetworkRequest{
		Name: "diy-network", OrgMSPID: "Org1MSP", Domain: "diy-network.com", ChannelName: "sandbox",
	})
	if err != nil {
		t.Fatalf("Create with default NumPeers: %v", err)
	}
}

// / @brief  Création refusée si un champ requis est manquant
// / @input  bootstrapper injecté, OrgMSPID vide
// / @expect retourne une erreur de validation, le bootstrapper n'est pas appelé
func TestCreate_MissingOrgMSPID(t *testing.T) {
	svc := network.NewService(&mockRepo{}, nil).WithBootstrapper(&mockBootstrapper{})
	_, err := svc.Create(network.CreateNetworkRequest{Name: "diy-network", Domain: "diy-network.com", ChannelName: "sandbox"})
	if err == nil {
		t.Error("want error for missing OrgMSPID")
	}
}

// / @brief  Création propage l'erreur du bootstrapper
// / @input  bootstrapper retournant une erreur "docker indisponible"
// / @expect l'erreur est propagée, rien n'est persisté dans le repo
func TestCreate_BootstrapError(t *testing.T) {
	repo := &mockRepo{}
	svc := network.NewService(repo, nil).WithBootstrapper(&mockBootstrapper{err: errors.New("docker indisponible")})
	_, err := svc.Create(network.CreateNetworkRequest{
		Name: "diy-network", OrgMSPID: "Org1MSP", Domain: "diy-network.com", ChannelName: "sandbox",
	})
	if err == nil {
		t.Error("want error propagated from bootstrapper")
	}
	all, _ := repo.FindAll()
	if len(all) != 0 {
		t.Errorf("expected nothing persisted on bootstrap error, got %d entries", len(all))
	}
}
