// adapters/in/cli/org_test.go — tests des handlers CLI organisation (UCADM01)
package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"myr/domain/channel"
	"myr/domain/network"
)

// ── Mock ChannelService (Pattern B — fn-func) ─────────────────────────────────

type mockChannelSvc struct {
	get              func(id string) (*channel.Channel, error)
	list             func() ([]*channel.Channel, error)
	addOrganisation  func(channelID string, org channel.Organization) error
	addNode          func(channelID string, nodeType channel.NodeType, addr, orgMSP string, certs channel.NodeCerts) error
	removeNode       func(channelID, addr string) error
}

func (m *mockChannelSvc) Get(id string) (*channel.Channel, error) {
	if m.get != nil {
		return m.get(id)
	}
	return nil, nil
}

func (m *mockChannelSvc) List() ([]*channel.Channel, error) {
	if m.list != nil {
		return m.list()
	}
	return nil, nil
}

func (m *mockChannelSvc) AddOrganisation(channelID string, org channel.Organization) error {
	if m.addOrganisation != nil {
		return m.addOrganisation(channelID, org)
	}
	return nil
}

func (m *mockChannelSvc) AddNode(channelID string, nodeType channel.NodeType, addr, orgMSP string, certs channel.NodeCerts) error {
	if m.addNode != nil {
		return m.addNode(channelID, nodeType, addr, orgMSP, certs)
	}
	return nil
}

func (m *mockChannelSvc) RemoveNode(channelID, addr string) error {
	if m.removeNode != nil {
		return m.removeNode(channelID, addr)
	}
	return nil
}

// activeNetSvc retourne un mockNetworkSvc avec un réseau actif sur le canal "sandbox".
func activeNetSvc() network.NetworkService {
	return &mockNetworkSvc{
		getActive: func() (*network.NetworkProfile, error) {
			return &network.NetworkProfile{FabricChannel: "sandbox"}, nil
		},
	}
}

// writeTempCert écrit un fichier PEM temporaire dans t.TempDir() et retourne son chemin.
func writeTempCert(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "cert.pem")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// ── runOrgAdd ─────────────────────────────────────────────────────────────────

// Sauvegarde et restauration des flags globaux pour les tests
func withOrgAddFlags(msp, name, cert, channelFlag, role, tlsCert string, update bool, fn func()) {
	prevMSP, prevName, prevCert, prevChannel, prevRole, prevTLS, prevUpdate :=
		orgAddMSP, orgAddName, orgAddCert, orgAddChannel, orgAddRole, orgAddTLSCert, orgAddUpdate
	orgAddMSP = msp
	orgAddName = name
	orgAddCert = cert
	orgAddChannel = channelFlag
	orgAddRole = role
	orgAddTLSCert = tlsCert
	orgAddUpdate = update
	fn()
	orgAddMSP, orgAddName, orgAddCert, orgAddChannel, orgAddRole, orgAddTLSCert, orgAddUpdate =
		prevMSP, prevName, prevCert, prevChannel, prevRole, prevTLS, prevUpdate
}

/// @brief  Ajout nominal d'une organisation valide au canal Fabric (UCADM01 flux nominal)
/// @input  MSP "Org2MSP", cert valide, réseau actif sur canal "sandbox"
/// @expect message de succès affiché, AddOrganisation appelé avec le bon channelID
func TestRunOrgAdd_NominalCase(t *testing.T) {
	certPath := writeTempCert(t, "-----BEGIN CERTIFICATE-----\nMOCK\n-----END CERTIFICATE-----\n")

	var capturedChannelID string
	var capturedMSP string
	chSvc := &mockChannelSvc{
		addOrganisation: func(channelID string, org channel.Organization) error {
			capturedChannelID = channelID
			capturedMSP = org.MSPID
			return nil
		},
	}
	netSvc := activeNetSvc()

	withOrgAddFlags("Org2MSP", "Org 2", certPath, "", "member", "", false, func() {
		var buf bytes.Buffer
		if err := runOrgAdd(&buf, chSvc, netSvc); err != nil {
			t.Fatalf("runOrgAdd: %v", err)
		}
		out := buf.String()
		if !strings.Contains(out, "Org 2") || !strings.Contains(out, "sandbox") {
			t.Errorf("attendu nom org et canal dans la sortie, obtenu : %s", out)
		}
	})

	if capturedChannelID != "sandbox" {
		t.Errorf("attendu channelID 'sandbox', obtenu %q", capturedChannelID)
	}
	if capturedMSP != "Org2MSP" {
		t.Errorf("attendu MSP 'Org2MSP', obtenu %q", capturedMSP)
	}
}

/// @brief  Ajout avec --channel explicite surcharge le réseau actif (DC-CLI-01)
/// @input  --channel "greenchannel", réseau actif sur "sandbox"
/// @expect AddOrganisation appelé sur "greenchannel"
func TestRunOrgAdd_ExplicitChannel(t *testing.T) {
	certPath := writeTempCert(t, "MOCK-PEM")
	var capturedChannelID string
	chSvc := &mockChannelSvc{
		addOrganisation: func(channelID string, org channel.Organization) error {
			capturedChannelID = channelID
			return nil
		},
	}
	netSvc := activeNetSvc()

	withOrgAddFlags("Org2MSP", "Org 2", certPath, "greenchannel", "member", "", false, func() {
		if err := runOrgAdd(&bytes.Buffer{}, chSvc, netSvc); err != nil {
			t.Fatalf("runOrgAdd: %v", err)
		}
	})

	if capturedChannelID != "greenchannel" {
		t.Errorf("attendu channelID 'greenchannel', obtenu %q", capturedChannelID)
	}
}

/// @brief  Identifiant d'organisation invalide retourne l'erreur ErrInvalidMSPID du service
/// @input  org-id "Org#2 MSP" (caractères interdits), service retourne ErrInvalidMSPID
/// @expect erreur "identifiant d'organisation invalide"
func TestRunOrgAdd_InvalidMSPID_Rejected(t *testing.T) {
	certPath := writeTempCert(t, "MOCK-PEM")
	chSvc := &mockChannelSvc{
		addOrganisation: func(channelID string, org channel.Organization) error {
			return channel.ErrInvalidMSPID
		},
	}

	withOrgAddFlags("Org#2 MSP", "Org 2", certPath, "sandbox", "member", "", false, func() {
		err := runOrgAdd(&bytes.Buffer{}, chSvc, activeNetSvc())
		if err == nil || !strings.Contains(err.Error(), "identifiant d'organisation invalide") {
			t.Errorf("attendu erreur 'identifiant d'organisation invalide', obtenu %v", err)
		}
	})
}

/// @brief  Adaptateur blockchain non configuré retourne l'erreur appropriée
/// @input  Service retourne ErrFabricUnavailable
/// @expect erreur "adaptateur blockchain non configuré"
func TestRunOrgAdd_FabricUnavailable_Rejected(t *testing.T) {
	certPath := writeTempCert(t, "MOCK-PEM")
	chSvc := &mockChannelSvc{
		addOrganisation: func(channelID string, org channel.Organization) error {
			return channel.ErrFabricUnavailable
		},
	}

	withOrgAddFlags("Org2MSP", "Org 2", certPath, "sandbox", "member", "", false, func() {
		err := runOrgAdd(&bytes.Buffer{}, chSvc, activeNetSvc())
		if err == nil || !strings.Contains(err.Error(), "adaptateur blockchain non configuré") {
			t.Errorf("attendu erreur 'adaptateur blockchain non configuré', obtenu %v", err)
		}
	})
}

/// @brief  Politique d'endorsement non satisfaite retourne l'erreur appropriée
/// @input  Service retourne ErrEndorsementPolicy
/// @expect erreur "politique d'endorsement non satisfaite"
func TestRunOrgAdd_EndorsementPolicy_Rejected(t *testing.T) {
	certPath := writeTempCert(t, "MOCK-PEM")
	chSvc := &mockChannelSvc{
		addOrganisation: func(channelID string, org channel.Organization) error {
			return channel.ErrEndorsementPolicy
		},
	}

	withOrgAddFlags("Org2MSP", "Org 2", certPath, "sandbox", "member", "", false, func() {
		err := runOrgAdd(&bytes.Buffer{}, chSvc, activeNetSvc())
		if err == nil || !strings.Contains(err.Error(), "politique d'endorsement") {
			t.Errorf("attendu erreur 'politique d'endorsement', obtenu %v", err)
		}
	})
}

/// @brief  Organisation déjà membre sans --update retourne une erreur suggérant --update
/// @input  Service retourne ErrAlreadyMember, update=false
/// @expect erreur "utilisez --update"
func TestRunOrgAdd_AlreadyMember_WithoutUpdate_Rejected(t *testing.T) {
	certPath := writeTempCert(t, "MOCK-PEM")
	chSvc := &mockChannelSvc{
		addOrganisation: func(channelID string, org channel.Organization) error {
			return channel.ErrAlreadyMember
		},
	}

	withOrgAddFlags("Org2MSP", "Org 2", certPath, "sandbox", "member", "", false, func() {
		err := runOrgAdd(&bytes.Buffer{}, chSvc, activeNetSvc())
		if err == nil || !strings.Contains(err.Error(), "--update") {
			t.Errorf("attendu erreur '--update', obtenu %v", err)
		}
	})
}

/// @brief  Organisation déjà membre avec --update déclenche la mise à jour
/// @input  ErrAlreadyMember sur premier appel, nil sur second, update=true
/// @expect message de mise à jour affiché, second appel effectué
func TestRunOrgAdd_AlreadyMember_WithUpdate_OK(t *testing.T) {
	certPath := writeTempCert(t, "MOCK-PEM")
	callCount := 0
	chSvc := &mockChannelSvc{
		addOrganisation: func(channelID string, org channel.Organization) error {
			callCount++
			if callCount == 1 {
				return channel.ErrAlreadyMember
			}
			return nil
		},
	}

	var buf bytes.Buffer
	withOrgAddFlags("Org2MSP", "Org 2", certPath, "sandbox", "member", "", true, func() {
		if err := runOrgAdd(&buf, chSvc, activeNetSvc()); err != nil {
			t.Fatalf("runOrgAdd avec --update: %v", err)
		}
	})

	if callCount != 2 {
		t.Errorf("attendu 2 appels à AddOrganisation (1 initial + 1 mise à jour), obtenu %d", callCount)
	}
	if !strings.Contains(buf.String(), "mise à jour") {
		t.Errorf("attendu message 'mise à jour' dans la sortie, obtenu : %s", buf.String())
	}
}

/// @brief  Fichier certificat introuvable retourne une erreur descriptive
/// @input  Chemin cert inexistant
/// @expect erreur "certificat introuvable"
func TestRunOrgAdd_CertNotFound_Rejected(t *testing.T) {
	chSvc := &mockChannelSvc{}

	withOrgAddFlags("Org2MSP", "Org 2", "/tmp/nonexistent-myr-test.pem", "sandbox", "member", "", false, func() {
		err := runOrgAdd(&bytes.Buffer{}, chSvc, activeNetSvc())
		if err == nil || !strings.Contains(err.Error(), "certificat introuvable") {
			t.Errorf("attendu erreur 'certificat introuvable', obtenu %v", err)
		}
	})
}

/// @brief  Canal non configuré si aucun réseau actif et --channel absent
/// @input  getActive retourne nil, channelFlag=""
/// @expect erreur "canal non configuré"
func TestRunOrgAdd_NoChannel_Rejected(t *testing.T) {
	certPath := writeTempCert(t, "MOCK-PEM")
	chSvc := &mockChannelSvc{}
	netSvc := &mockNetworkSvc{
		getActive: func() (*network.NetworkProfile, error) { return nil, nil },
	}

	withOrgAddFlags("Org2MSP", "Org 2", certPath, "", "member", "", false, func() {
		err := runOrgAdd(&bytes.Buffer{}, chSvc, netSvc)
		if err == nil || !strings.Contains(err.Error(), "canal non configuré") {
			t.Errorf("attendu erreur 'canal non configuré', obtenu %v", err)
		}
	})
}

/// @brief  ErrAlreadyMember avec --update mais second appel échoue propage l'erreur
/// @input  ErrAlreadyMember sur premier appel, erreur générique sur second, update=true
/// @expect erreur du second appel propagée
func TestRunOrgAdd_AlreadyMember_WithUpdate_SecondCallFails(t *testing.T) {
	certPath := writeTempCert(t, "MOCK-PEM")
	callCount := 0
	chSvc := &mockChannelSvc{
		addOrganisation: func(channelID string, org channel.Organization) error {
			callCount++
			if callCount == 1 {
				return channel.ErrAlreadyMember
			}
			return errors.New("fabric update failed")
		},
	}

	withOrgAddFlags("Org2MSP", "Org 2", certPath, "sandbox", "member", "", true, func() {
		err := runOrgAdd(&bytes.Buffer{}, chSvc, activeNetSvc())
		if err == nil {
			t.Error("attendu erreur sur second appel, obtenu nil")
		}
	})
}
