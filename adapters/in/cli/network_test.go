// adapters/in/cli/network_test.go — tests des handlers CLI réseau
package cli

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"myr/domain/network"
)

// ── Mock NetworkService (Pattern B — fn-func) ─────────────────────────────────

type mockNetworkSvc struct {
	list         func() ([]*network.NetworkProfile, error)
	add          func(name, peer, gateway, msp, cert, key, tlsCert, channel, chaincode, ca, caName string, channels []string, autoGuest, autoRegister bool, autoRole string, production bool) (*network.NetworkProfile, error)
	update       func(id, name, peer, gateway, msp, cert, key, tlsCert, channel, chaincode, ca, caName string, autoGuest, autoRegister bool, autoRole string, production bool) (*network.NetworkProfile, error)
	getActive    func() (*network.NetworkProfile, error)
	activate     func(id string) error
	delete       func(id string) error
	testConn     func(id string) error
	addPeer      func(networkID string, req network.AddPeerRequest) (*network.PeerCredentials, error)
	create       func(req network.CreateNetworkRequest) (*network.NetworkProfile, error)
}

func (m *mockNetworkSvc) List() ([]*network.NetworkProfile, error) {
	if m.list != nil {
		return m.list()
	}
	return nil, nil
}

func (m *mockNetworkSvc) Add(name, peer, gateway, msp, cert, key, tlsCert, channel, chaincode, ca, caName string, channels []string, autoGuest, autoRegister bool, autoRole string, production bool) (*network.NetworkProfile, error) {
	if m.add != nil {
		return m.add(name, peer, gateway, msp, cert, key, tlsCert, channel, chaincode, ca, caName, channels, autoGuest, autoRegister, autoRole, production)
	}
	return &network.NetworkProfile{ID: "net-test", Name: name}, nil
}

func (m *mockNetworkSvc) Update(id, name, peer, gateway, msp, cert, key, tlsCert, channel, chaincode, ca, caName string, autoGuest, autoRegister bool, autoRole string, production bool) (*network.NetworkProfile, error) {
	if m.update != nil {
		return m.update(id, name, peer, gateway, msp, cert, key, tlsCert, channel, chaincode, ca, caName, autoGuest, autoRegister, autoRole, production)
	}
	return &network.NetworkProfile{ID: id, Name: name}, nil
}

func (m *mockNetworkSvc) GetActive() (*network.NetworkProfile, error) {
	if m.getActive != nil {
		return m.getActive()
	}
	return nil, nil
}

func (m *mockNetworkSvc) Activate(id string) error {
	if m.activate != nil {
		return m.activate(id)
	}
	return nil
}

func (m *mockNetworkSvc) Delete(id string) error {
	if m.delete != nil {
		return m.delete(id)
	}
	return nil
}

func (m *mockNetworkSvc) TestConnection(id string) error {
	if m.testConn != nil {
		return m.testConn(id)
	}
	return nil
}

func (m *mockNetworkSvc) AddPeer(networkID string, req network.AddPeerRequest) (*network.PeerCredentials, error) {
	if m.addPeer != nil {
		return m.addPeer(networkID, req)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockNetworkSvc) Create(req network.CreateNetworkRequest) (*network.NetworkProfile, error) {
	if m.create != nil {
		return m.create(req)
	}
	return &network.NetworkProfile{ID: "net-test", Name: req.Name}, nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func sampleProfile(id, name string, active bool) *network.NetworkProfile {
	return &network.NetworkProfile{
		ID:            id,
		Name:          name,
		PeerEndpoint:  "203.0.113.10:7051",
		GatewayPeer:   "peer0.org1.example.com",
		MSPID:         "Org1MSP",
		FabricChannel: "sandbox",
		CAEndpoint:    "https://203.0.113.10:7054",
		CAName:        "ca-org1",
		Active:        active,
		CreatedAt:     time.Now(),
	}
}

// ── runNetworkList ────────────────────────────────────────────────────────────

/// @brief  Listage nominal de plusieurs réseaux avec marqueur actif
/// @input  Deux réseaux dont un actif
/// @expect tableau affiché avec colonnes et marqueur "*" sur le réseau actif
func TestRunNetworkList_NominalCase(t *testing.T) {
	var buf bytes.Buffer
	svc := &mockNetworkSvc{
		list: func() ([]*network.NetworkProfile, error) {
			return []*network.NetworkProfile{
				sampleProfile("net-001", "sample-network", true),
				sampleProfile("net-002", "sandbox-test", false),
			}, nil
		},
	}
	if err := runNetworkList(&buf, svc); err != nil {
		t.Fatalf("runNetworkList: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "net-001") {
		t.Errorf("attendu net-001 dans la sortie, obtenu : %s", out)
	}
	if !strings.Contains(out, "*") {
		t.Errorf("attendu marqueur '*' pour le réseau actif, obtenu : %s", out)
	}
	if !strings.Contains(out, "net-002") {
		t.Errorf("attendu net-002 dans la sortie, obtenu : %s", out)
	}
}

/// @brief  Listage retourne le message approprié si aucun réseau configuré
/// @input  Liste vide
/// @expect message indiquant "Aucun réseau configuré"
func TestRunNetworkList_Empty(t *testing.T) {
	var buf bytes.Buffer
	svc := &mockNetworkSvc{
		list: func() ([]*network.NetworkProfile, error) { return nil, nil },
	}
	if err := runNetworkList(&buf, svc); err != nil {
		t.Fatalf("runNetworkList: %v", err)
	}
	if !strings.Contains(buf.String(), "Aucun réseau configuré") {
		t.Errorf("attendu message vide, obtenu : %s", buf.String())
	}
}

/// @brief  Listage propage l'erreur du service
/// @input  Service retournant une erreur
/// @expect l'erreur remonte
func TestRunNetworkList_ServiceError(t *testing.T) {
	svc := &mockNetworkSvc{
		list: func() ([]*network.NetworkProfile, error) { return nil, errors.New("store error") },
	}
	if err := runNetworkList(&bytes.Buffer{}, svc); err == nil {
		t.Error("attendu erreur, obtenu nil")
	}
}

// ── runNetworkShow ────────────────────────────────────────────────────────────

/// @brief  Affichage nominal des détails d'un profil réseau
/// @input  Réseau "net-001" présent dans le store
/// @expect bloc détaillé avec séparateurs et toutes les clés du profil
func TestRunNetworkShow_NominalCase(t *testing.T) {
	var buf bytes.Buffer
	svc := &mockNetworkSvc{
		list: func() ([]*network.NetworkProfile, error) {
			return []*network.NetworkProfile{sampleProfile("net-001", "sample-network", true)}, nil
		},
	}
	if err := runNetworkShow(&buf, svc, "net-001"); err != nil {
		t.Fatalf("runNetworkShow: %v", err)
	}
	out := buf.String()
	for _, expected := range []string{"sample-network", "net-001", "Org1MSP", "sandbox", "203.0.113.10:7051"} {
		if !strings.Contains(out, expected) {
			t.Errorf("attendu %q dans la sortie", expected)
		}
	}
}

/// @brief  show retourne une erreur si l'ID est introuvable
/// @input  ID "net-999" absent
/// @expect erreur "réseau introuvable"
func TestRunNetworkShow_NotFound(t *testing.T) {
	svc := &mockNetworkSvc{
		list: func() ([]*network.NetworkProfile, error) {
			return []*network.NetworkProfile{sampleProfile("net-001", "sample-network", false)}, nil
		},
	}
	err := runNetworkShow(&bytes.Buffer{}, svc, "net-999")
	if err == nil || !strings.Contains(err.Error(), "réseau introuvable") {
		t.Errorf("attendu erreur 'réseau introuvable', obtenu %v", err)
	}
}

// ── runNetworkAdd ─────────────────────────────────────────────────────────────

/// @brief  Ajout nominal d'un profil réseau avec les paramètres requis
/// @input  Paramètres valides : name, peer host:port, msp
/// @expect sortie avec l'ID et instruction pour activer
func TestRunNetworkAdd_NominalCase(t *testing.T) {
	var buf bytes.Buffer
	svc := &mockNetworkSvc{
		add: func(name, peer, _, _, _, _, _, _, _, _, _ string, _ []string, _, _ bool, _ string, _ bool) (*network.NetworkProfile, error) {
			return &network.NetworkProfile{ID: "net-new", Name: name}, nil
		},
	}
	err := runNetworkAdd(&buf, svc, "mon-réseau", "203.0.113.10:7051", "", "Org1MSP", "", "", "", "", "", "", "", false, false, "reader", false)
	if err != nil {
		t.Fatalf("runNetworkAdd: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "net-new") {
		t.Errorf("attendu ID 'net-new' dans la sortie, obtenu : %s", out)
	}
	if !strings.Contains(out, "activate") {
		t.Errorf("attendu instruction 'activate' dans la sortie, obtenu : %s", out)
	}
}

/// @brief  Ajout avec peer invalide (pas host:port) est rejeté par le service
/// @input  Service retournant erreur de validation adresse
/// @expect erreur propagée
func TestRunNetworkAdd_InvalidPeer_Rejected(t *testing.T) {
	svc := &mockNetworkSvc{
		add: func(_, _, _, _, _, _, _, _, _, _, _ string, _ []string, _, _ bool, _ string, _ bool) (*network.NetworkProfile, error) {
			return nil, fmt.Errorf("adresse du peer invalide")
		},
	}
	if err := runNetworkAdd(&bytes.Buffer{}, svc, "test", "invalid", "", "Org1MSP", "", "", "", "", "", "", "", false, false, "reader", false); err == nil {
		t.Error("attendu erreur pour peer invalide")
	}
}

// ── runNetworkActivate ────────────────────────────────────────────────────────

/// @brief  Activation nominale d'un réseau existant
/// @input  Réseau "net-001" présent, activate retourne nil
/// @expect message confirmant l'activation avec le nom et l'ID
func TestRunNetworkActivate_NominalCase(t *testing.T) {
	var buf bytes.Buffer
	svc := &mockNetworkSvc{
		list: func() ([]*network.NetworkProfile, error) {
			return []*network.NetworkProfile{sampleProfile("net-001", "sample-network", false)}, nil
		},
		activate: func(id string) error { return nil },
	}
	if err := runNetworkActivate(&buf, svc, "net-001"); err != nil {
		t.Fatalf("runNetworkActivate: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "sample-network") || !strings.Contains(out, "net-001") {
		t.Errorf("attendu nom et ID dans la sortie, obtenu : %s", out)
	}
}

/// @brief  Activation d'un ID inexistant retourne une erreur
/// @input  Store vide, ID "net-999"
/// @expect erreur "réseau introuvable"
func TestRunNetworkActivate_NotFound(t *testing.T) {
	svc := &mockNetworkSvc{
		list: func() ([]*network.NetworkProfile, error) { return nil, nil },
	}
	err := runNetworkActivate(&bytes.Buffer{}, svc, "net-999")
	if err == nil || !strings.Contains(err.Error(), "réseau introuvable") {
		t.Errorf("attendu erreur 'réseau introuvable', obtenu %v", err)
	}
}

// ── runNetworkDelete ──────────────────────────────────────────────────────────

/// @brief  Suppression nominale avec --yes (pas de prompt)
/// @input  Réseau "net-001" présent, yes=true
/// @expect message de suppression, delete appelé
func TestRunNetworkDelete_WithYes(t *testing.T) {
	var buf bytes.Buffer
	deleted := false
	svc := &mockNetworkSvc{
		list: func() ([]*network.NetworkProfile, error) {
			return []*network.NetworkProfile{sampleProfile("net-001", "sample-network", false)}, nil
		},
		delete: func(id string) error { deleted = true; return nil },
	}
	if err := runNetworkDelete(&buf, strings.NewReader(""), svc, "net-001", true); err != nil {
		t.Fatalf("runNetworkDelete: %v", err)
	}
	if !deleted {
		t.Error("attendu appel à Delete, non effectué")
	}
	if !strings.Contains(buf.String(), "supprimé") {
		t.Errorf("attendu 'supprimé' dans la sortie, obtenu : %s", buf.String())
	}
}

/// @brief  Suppression avec prompt interactif et réponse "o"
/// @input  Réseau "net-001", yes=false, input="o"
/// @expect delete est appelé après confirmation
func TestRunNetworkDelete_InteractiveConfirm(t *testing.T) {
	var buf bytes.Buffer
	deleted := false
	svc := &mockNetworkSvc{
		list: func() ([]*network.NetworkProfile, error) {
			return []*network.NetworkProfile{sampleProfile("net-001", "sample-network", false)}, nil
		},
		delete: func(id string) error { deleted = true; return nil },
	}
	if err := runNetworkDelete(&buf, strings.NewReader("o\n"), svc, "net-001", false); err != nil {
		t.Fatalf("runNetworkDelete: %v", err)
	}
	if !deleted {
		t.Error("attendu appel à Delete après confirmation 'o'")
	}
}

/// @brief  Suppression annulée si l'utilisateur répond "N"
/// @input  Réseau "net-001", yes=false, input="N"
/// @expect delete n'est pas appelé, message "Annulé"
func TestRunNetworkDelete_InteractiveCancel(t *testing.T) {
	var buf bytes.Buffer
	deleted := false
	svc := &mockNetworkSvc{
		list: func() ([]*network.NetworkProfile, error) {
			return []*network.NetworkProfile{sampleProfile("net-001", "sample-network", false)}, nil
		},
		delete: func(id string) error { deleted = true; return nil },
	}
	if err := runNetworkDelete(&buf, strings.NewReader("N\n"), svc, "net-001", false); err != nil {
		t.Fatalf("runNetworkDelete: %v", err)
	}
	if deleted {
		t.Error("delete ne doit pas être appelé après annulation")
	}
	if !strings.Contains(buf.String(), "Annulé") {
		t.Errorf("attendu 'Annulé' dans la sortie, obtenu : %s", buf.String())
	}
}

/// @brief  Suppression d'un réseau actif affiche un avertissement
/// @input  Réseau actif "net-001", yes=true
/// @expect avertissement "actuellement actif" affiché avant suppression
func TestRunNetworkDelete_ActiveNetworkWarning(t *testing.T) {
	var buf bytes.Buffer
	svc := &mockNetworkSvc{
		list: func() ([]*network.NetworkProfile, error) {
			return []*network.NetworkProfile{sampleProfile("net-001", "sample-network", true)}, nil
		},
		delete: func(id string) error { return nil },
	}
	if err := runNetworkDelete(&buf, strings.NewReader(""), svc, "net-001", true); err != nil {
		t.Fatalf("runNetworkDelete: %v", err)
	}
	if !strings.Contains(buf.String(), "actuellement actif") {
		t.Errorf("attendu avertissement 'actuellement actif', obtenu : %s", buf.String())
	}
}

/// @brief  Suppression d'un ID inexistant retourne une erreur
/// @input  Store vide, ID "net-999"
/// @expect erreur "réseau introuvable"
func TestRunNetworkDelete_NotFound(t *testing.T) {
	svc := &mockNetworkSvc{
		list: func() ([]*network.NetworkProfile, error) { return nil, nil },
	}
	err := runNetworkDelete(&bytes.Buffer{}, strings.NewReader(""), svc, "net-999", true)
	if err == nil || !strings.Contains(err.Error(), "réseau introuvable") {
		t.Errorf("attendu erreur 'réseau introuvable', obtenu %v", err)
	}
}

// ── runNetworkTest ────────────────────────────────────────────────────────────

/// @brief  Test de connectivité nominal vers un réseau spécifié par ID
/// @input  Réseau "net-001" présent, testConn retourne nil
/// @expect messages "Test de connectivité" et "OK — peer joignable"
func TestRunNetworkTest_NominalCase(t *testing.T) {
	var buf bytes.Buffer
	svc := &mockNetworkSvc{
		list: func() ([]*network.NetworkProfile, error) {
			return []*network.NetworkProfile{sampleProfile("net-001", "sample-network", false)}, nil
		},
		testConn: func(id string) error { return nil },
	}
	if err := runNetworkTest(&buf, svc, "net-001"); err != nil {
		t.Fatalf("runNetworkTest: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "OK") {
		t.Errorf("attendu 'OK' dans la sortie, obtenu : %s", out)
	}
}

/// @brief  Test sans ID utilise le réseau actif
/// @input  Réseau "net-001" actif, id=""
/// @expect endpoint du réseau actif mentionné dans la sortie
func TestRunNetworkTest_UsesActiveNetwork(t *testing.T) {
	var buf bytes.Buffer
	svc := &mockNetworkSvc{
		list: func() ([]*network.NetworkProfile, error) {
			return []*network.NetworkProfile{sampleProfile("net-001", "sample-network", true)}, nil
		},
		testConn: func(id string) error { return nil },
	}
	if err := runNetworkTest(&buf, svc, ""); err != nil {
		t.Fatalf("runNetworkTest: %v", err)
	}
	if !strings.Contains(buf.String(), "203.0.113.10:7051") {
		t.Errorf("attendu endpoint du réseau actif dans la sortie, obtenu : %s", buf.String())
	}
}

/// @brief  Test sans ID et sans réseau actif retourne une erreur
/// @input  Store vide, id=""
/// @expect erreur "aucun réseau actif"
func TestRunNetworkTest_NoActiveNetwork(t *testing.T) {
	svc := &mockNetworkSvc{
		list: func() ([]*network.NetworkProfile, error) { return nil, nil },
	}
	err := runNetworkTest(&bytes.Buffer{}, svc, "")
	if err == nil || !strings.Contains(err.Error(), "aucun réseau actif") {
		t.Errorf("attendu erreur 'aucun réseau actif', obtenu %v", err)
	}
}

/// @brief  Échec de connectivité retourne une erreur avec l'endpoint
/// @input  Réseau "net-001" présent, testConn retourne une erreur
/// @expect erreur "impossible de joindre" avec l'endpoint
func TestRunNetworkTest_ConnectionFailed(t *testing.T) {
	svc := &mockNetworkSvc{
		list: func() ([]*network.NetworkProfile, error) {
			return []*network.NetworkProfile{sampleProfile("net-001", "sample-network", false)}, nil
		},
		testConn: func(id string) error { return errors.New("connection refused") },
	}
	err := runNetworkTest(&bytes.Buffer{}, svc, "net-001")
	if err == nil || !strings.Contains(err.Error(), "impossible de joindre") {
		t.Errorf("attendu erreur 'impossible de joindre', obtenu %v", err)
	}
}

// ── runNetworkUpdate ──────────────────────────────────────────────────────────

/// @brief  Update nominal : seuls les flags changés sont fusionnés dans le profil existant (DC-CLI-05)
/// @input  Réseau "net-001" existant, --cert changé
/// @expect Update appelé avec les valeurs fusionnées (cert mis à jour, reste inchangé)
func TestRunNetworkUpdate_NominalCase(t *testing.T) {
	var buf bytes.Buffer
	var capturedCert string
	svc := &mockNetworkSvc{
		list: func() ([]*network.NetworkProfile, error) {
			return []*network.NetworkProfile{sampleProfile("net-001", "sample-network", false)}, nil
		},
		update: func(id, name, peer, gateway, msp, cert, key, tlsCert, channel, chaincode, ca, caName string, _, _ bool, _ string, _ bool) (*network.NetworkProfile, error) {
			capturedCert = cert
			return &network.NetworkProfile{ID: id}, nil
		},
	}
	// Simuler un cobra.Command avec le flag "cert" changé
	cmd := &cobra.Command{}
	cmd.Flags().String("name", "", "")
	cmd.Flags().String("peer", "", "")
	cmd.Flags().String("msp", "", "")
	cmd.Flags().String("gateway", "", "")
	cmd.Flags().String("cert", "", "")
	cmd.Flags().String("key", "", "")
	cmd.Flags().String("tls-cert", "", "")
	cmd.Flags().String("channel", "", "")
	cmd.Flags().String("chaincode", "", "")
	cmd.Flags().String("ca", "", "")
	cmd.Flags().String("ca-name", "", "")
	_ = cmd.Flags().Set("cert", "./new-cert.pem")

	if err := runNetworkUpdate(&buf, cmd, svc, "net-001"); err != nil {
		t.Fatalf("runNetworkUpdate: %v", err)
	}
	if capturedCert != "./new-cert.pem" {
		t.Errorf("attendu cert './new-cert.pem', obtenu %q", capturedCert)
	}
	if !strings.Contains(buf.String(), "mis à jour") {
		t.Errorf("attendu 'mis à jour' dans la sortie, obtenu : %s", buf.String())
	}
}

/// @brief  Update d'un ID inexistant retourne une erreur
/// @input  Store vide, ID "net-999"
/// @expect erreur "réseau introuvable"
func TestRunNetworkUpdate_NotFound(t *testing.T) {
	svc := &mockNetworkSvc{
		list: func() ([]*network.NetworkProfile, error) { return nil, nil },
	}
	cmd := &cobra.Command{}
	cmd.Flags().String("cert", "", "")
	err := runNetworkUpdate(&bytes.Buffer{}, cmd, svc, "net-999")
	if err == nil || !strings.Contains(err.Error(), "réseau introuvable") {
		t.Errorf("attendu erreur 'réseau introuvable', obtenu %v", err)
	}
}

// ── runNetworkImport ──────────────────────────────────────────────────────────

/// @brief  Import nominal d'un profil Fabric gateway-connection.json (DC-CLI-06)
/// @input  Fichier JSON Fabric valide avec organization, peer, CA
/// @expect profil importé, Add appelé avec MSP et endpoint extraits
func TestRunNetworkImport_FabricProfile_NominalCase(t *testing.T) {
	dir := t.TempDir()
	profilePath := filepath.Join(dir, "connection.json")
	content := `{
		"name": "test-network",
		"client": {"organization": "Org1"},
		"organizations": {
			"Org1": {"mspid": "Org1MSP", "peers": ["peer0.org1.example.com"]}
		},
		"peers": {
			"peer0.org1.example.com": {
				"url": "grpcs://203.0.113.10:7051",
				"grpcOptions": {"ssl-target-name-override": "peer0.org1.example.com"},
				"tlsCACerts": {"pem": "-----BEGIN CERTIFICATE-----\nMOCK\n-----END CERTIFICATE-----\n"}
			}
		},
		"certificateAuthorities": {
			"ca.org1.example.com": {"url": "https://203.0.113.10:7054", "caName": "ca-org1"}
		},
		"channels": {"sandbox": {}}
	}`
	if err := os.WriteFile(profilePath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	var addedMSP string
	svc := &mockNetworkSvc{
		add: func(name, peer, _, msp, _, _, _, _, _, _, _ string, _ []string, _, _ bool, _ string, _ bool) (*network.NetworkProfile, error) {
			addedMSP = msp
			return &network.NetworkProfile{ID: "net-import", Name: name}, nil
		},
	}

	var buf bytes.Buffer
	if err := runNetworkImport(&buf, svc, profilePath, "", false); err != nil {
		t.Fatalf("runNetworkImport: %v", err)
	}
	if addedMSP != "Org1MSP" {
		t.Errorf("attendu MSP 'Org1MSP', obtenu %q", addedMSP)
	}
	if !strings.Contains(buf.String(), "net-import") {
		t.Errorf("attendu ID 'net-import' dans la sortie, obtenu : %s", buf.String())
	}
}

/// @brief  Import avec nameOverride surcharge le nom extrait du profil
/// @input  Profil Fabric avec name "test-network", nameOverride "mon-réseau"
/// @expect Add appelé avec name "mon-réseau"
func TestRunNetworkImport_NameOverride(t *testing.T) {
	dir := t.TempDir()
	profilePath := filepath.Join(dir, "connection.json")
	content := `{
		"name": "test-network",
		"client": {"organization": "Org1"},
		"organizations": {"Org1": {"mspid": "Org1MSP", "peers": ["peer0.org1.example.com"]}},
		"peers": {
			"peer0.org1.example.com": {
				"url": "grpc://localhost:7051",
				"grpcOptions": {"ssl-target-name-override": ""},
				"tlsCACerts": {"pem": ""}
			}
		},
		"certificateAuthorities": {},
		"channels": {"sandbox": {}}
	}`
	if err := os.WriteFile(profilePath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	var capturedName string
	svc := &mockNetworkSvc{
		add: func(name, _, _, _, _, _, _, _, _, _, _ string, _ []string, _, _ bool, _ string, _ bool) (*network.NetworkProfile, error) {
			capturedName = name
			return &network.NetworkProfile{ID: "net-x", Name: name}, nil
		},
	}
	if err := runNetworkImport(&bytes.Buffer{}, svc, profilePath, "mon-réseau", false); err != nil {
		t.Fatalf("runNetworkImport: %v", err)
	}
	if capturedName != "mon-réseau" {
		t.Errorf("attendu name 'mon-réseau', obtenu %q", capturedName)
	}
}

/// @brief  Import d'un NetworkProfile JSON direct (DC-CLI-06 format 2)
/// @input  JSON avec structure NetworkProfile complète
/// @expect Add appelé avec les champs du profil
func TestRunNetworkImport_NetworkProfileJSON_NominalCase(t *testing.T) {
	dir := t.TempDir()
	profilePath := filepath.Join(dir, "profile.json")
	content := `{
		"name": "my-network",
		"peer_endpoint": "localhost:7051",
		"msp_id": "MyMSP",
		"fabric_channel": "mychannel",
		"chaincode_name": "myrcc"
	}`
	if err := os.WriteFile(profilePath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	var capturedName string
	svc := &mockNetworkSvc{
		add: func(name, _, _, _, _, _, _, _, _, _, _ string, _ []string, _, _ bool, _ string, _ bool) (*network.NetworkProfile, error) {
			capturedName = name
			return &network.NetworkProfile{ID: "net-np", Name: name}, nil
		},
	}
	if err := runNetworkImport(&bytes.Buffer{}, svc, profilePath, "", false); err != nil {
		t.Fatalf("runNetworkImport: %v", err)
	}
	if capturedName != "my-network" {
		t.Errorf("attendu name 'my-network', obtenu %q", capturedName)
	}
}

/// @brief  Import avec fichier introuvable retourne une erreur
/// @input  Chemin de fichier inexistant
/// @expect erreur "fichier introuvable"
func TestRunNetworkImport_FileNotFound(t *testing.T) {
	svc := &mockNetworkSvc{}
	err := runNetworkImport(&bytes.Buffer{}, svc, "/tmp/does-not-exist-myr-test.json", "", false)
	if err == nil || !strings.Contains(err.Error(), "fichier introuvable") {
		t.Errorf("attendu erreur 'fichier introuvable', obtenu %v", err)
	}
}

/// @brief  Import d'un fichier au format invalide retourne une erreur descriptive
/// @input  Fichier JSON sans les champs Fabric ni NetworkProfile reconnus
/// @expect erreur "format de profil invalide"
func TestRunNetworkImport_InvalidFormat(t *testing.T) {
	dir := t.TempDir()
	profilePath := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(profilePath, []byte(`{"foo": "bar"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	svc := &mockNetworkSvc{}
	err := runNetworkImport(&bytes.Buffer{}, svc, profilePath, "", false)
	if err == nil || !strings.Contains(err.Error(), "format de profil invalide") {
		t.Errorf("attendu erreur 'format de profil invalide', obtenu %v", err)
	}
}

// ── runNetworkDestroy ─────────────────────────────────────────────────────────

/// @brief  destroy sans --confirm affiche le message d'avertissement et n'agit pas
/// @input  Réseau "net-001" présent, confirm=false
/// @expect message ATTENTION affiché, delete n'est pas appelé
func TestRunNetworkDestroy_WithoutConfirm(t *testing.T) {
	var buf bytes.Buffer
	deleted := false
	svc := &mockNetworkSvc{
		list: func() ([]*network.NetworkProfile, error) {
			return []*network.NetworkProfile{sampleProfile("net-001", "sample-network", false)}, nil
		},
		delete: func(id string) error { deleted = true; return nil },
	}
	if err := runNetworkDestroy(&buf, svc, "net-001", false, ""); err != nil {
		t.Fatalf("runNetworkDestroy: %v", err)
	}
	if deleted {
		t.Error("delete ne doit pas être appelé sans --confirm")
	}
	if !strings.Contains(buf.String(), "ATTENTION") {
		t.Errorf("attendu message 'ATTENTION', obtenu : %s", buf.String())
	}
}

/// @brief  destroy refuse le démantèlement d'un réseau de production (RM protège IsProduction)
/// @input  Réseau avec IsProduction=true, confirm=true
/// @expect erreur mentionnant IsProduction=true
func TestRunNetworkDestroy_ProductionRefused(t *testing.T) {
	prof := sampleProfile("net-001", "prod-network", false)
	prof.IsProduction = true
	svc := &mockNetworkSvc{
		list: func() ([]*network.NetworkProfile, error) {
			return []*network.NetworkProfile{prof}, nil
		},
		delete: func(id string) error { return nil },
	}
	err := runNetworkDestroy(&bytes.Buffer{}, svc, "net-001", true, "")
	if err == nil || !strings.Contains(err.Error(), "IsProduction") {
		t.Errorf("attendu erreur IsProduction, obtenu %v", err)
	}
}

/// @brief  destroy d'un ID inexistant retourne une erreur
/// @input  Store vide, ID "net-999"
/// @expect erreur "réseau introuvable"
func TestRunNetworkDestroy_NotFound(t *testing.T) {
	svc := &mockNetworkSvc{
		list: func() ([]*network.NetworkProfile, error) { return nil, nil },
	}
	err := runNetworkDestroy(&bytes.Buffer{}, svc, "net-999", true, "")
	if err == nil || !strings.Contains(err.Error(), "réseau introuvable") {
		t.Errorf("attendu erreur 'réseau introuvable', obtenu %v", err)
	}
}

/// @brief  destroy nominal (UCADM05 flux nominal) : --confirm=true, réseau non-production, Delete appelé
/// @input  Réseau "net-001" (IsProduction=false), confirm=true, dataPath="" (pas de ledger local)
/// @expect Delete appelé, sortie contient "démantelé", pas d'erreur
func TestRunNetworkDestroy_NominalCase(t *testing.T) {
	var buf bytes.Buffer
	deleted := false
	svc := &mockNetworkSvc{
		list: func() ([]*network.NetworkProfile, error) {
			return []*network.NetworkProfile{sampleProfile("net-001", "sample-network", false)}, nil
		},
		delete: func(id string) error { deleted = true; return nil },
	}
	if err := runNetworkDestroy(&buf, svc, "net-001", true, ""); err != nil {
		t.Fatalf("runNetworkDestroy nominal: %v", err)
	}
	if !deleted {
		t.Error("attendu appel à Delete, non effectué")
	}
	if !strings.Contains(buf.String(), "démantelé") {
		t.Errorf("attendu 'démantelé' dans la sortie, obtenu : %s", buf.String())
	}
}

// ── resolveChannel ────────────────────────────────────────────────────────────

/// @brief  resolveChannel retourne le flag channel s'il est fourni
/// @input  channelFlag="sandbox", réseau actif avec canal différent
/// @expect retourne "sandbox" sans consulter le réseau actif
func TestResolveChannel_FlagProvided(t *testing.T) {
	svc := &mockNetworkSvc{
		getActive: func() (*network.NetworkProfile, error) {
			return &network.NetworkProfile{FabricChannel: "other-channel"}, nil
		},
	}
	ch, err := resolveChannel("sandbox", svc)
	if err != nil {
		t.Fatalf("resolveChannel: %v", err)
	}
	if ch != "sandbox" {
		t.Errorf("attendu 'sandbox', obtenu %q", ch)
	}
}

/// @brief  resolveChannel utilise FabricChannel du réseau actif si flag absent
/// @input  channelFlag="", réseau actif avec FabricChannel="sandbox"
/// @expect retourne "sandbox"
func TestResolveChannel_UsesActiveNetwork(t *testing.T) {
	svc := &mockNetworkSvc{
		getActive: func() (*network.NetworkProfile, error) {
			return &network.NetworkProfile{FabricChannel: "sandbox"}, nil
		},
	}
	ch, err := resolveChannel("", svc)
	if err != nil {
		t.Fatalf("resolveChannel: %v", err)
	}
	if ch != "sandbox" {
		t.Errorf("attendu 'sandbox', obtenu %q", ch)
	}
}

/// @brief  resolveChannel retourne une erreur si aucun réseau actif et flag absent
/// @input  channelFlag="", getActive retourne nil
/// @expect erreur "canal non configuré"
func TestResolveChannel_NoActiveNetwork(t *testing.T) {
	svc := &mockNetworkSvc{
		getActive: func() (*network.NetworkProfile, error) { return nil, nil },
	}
	_, err := resolveChannel("", svc)
	if err == nil || !strings.Contains(err.Error(), "canal non configuré") {
		t.Errorf("attendu erreur 'canal non configuré', obtenu %v", err)
	}
}
