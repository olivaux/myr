// adapters/in/cli/node_test.go — tests des handlers CLI nœuds (UCADM03/04)
package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"myr/domain/channel"
	"myr/domain/network"
)

// ── Helpers ───────────────────────────────────────────────────────────────────

func withNodeAddFlags(nodeType, addr, org, channelFlag, cert string, fn func()) {
	prevType, prevAddr, prevOrg, prevChannel, prevCert :=
		nodeAddType, nodeAddAddr, nodeAddOrg, nodeAddChannel, nodeAddCert
	nodeAddType = nodeType
	nodeAddAddr = addr
	nodeAddOrg = org
	nodeAddChannel = channelFlag
	nodeAddCert = cert
	fn()
	nodeAddType, nodeAddAddr, nodeAddOrg, nodeAddChannel, nodeAddCert =
		prevType, prevAddr, prevOrg, prevChannel, prevCert
}

func withNodeRemoveFlags(addr, channelFlag string, fn func()) {
	prevAddr, prevChannel := nodeRemoveAddr, nodeRemoveChannel
	nodeRemoveAddr = addr
	nodeRemoveChannel = channelFlag
	fn()
	nodeRemoveAddr, nodeRemoveChannel = prevAddr, prevChannel
}

// ── runNodeAdd ────────────────────────────────────────────────────────────────

/// @brief  Ajout nominal d'un peer à un canal Fabric (UCADM03 flux nominal)
/// @input  type=peer, addr=host:port valide, org=Org2MSP, canal=sandbox
/// @expect message "ajouté" affiché, AddNode appelé avec les bons paramètres
func TestRunNodeAdd_Peer_NominalCase(t *testing.T) {
	var capturedType channel.NodeType
	var capturedAddr string
	chSvc := &mockChannelSvc{
		addNode: func(channelID string, nodeType channel.NodeType, addr, orgMSP string, certs channel.NodeCerts) error {
			capturedType = nodeType
			capturedAddr = addr
			return nil
		},
	}
	netSvc := activeNetSvc()

	var buf bytes.Buffer
	withNodeAddFlags("peer", "203.0.113.10:7051", "Org2MSP", "", "", func() {
		if err := runNodeAdd(&buf, chSvc, netSvc); err != nil {
			t.Fatalf("runNodeAdd: %v", err)
		}
	})

	if capturedType != channel.NodeTypePeer {
		t.Errorf("attendu NodeTypePeer, obtenu %q", capturedType)
	}
	if capturedAddr != "203.0.113.10:7051" {
		t.Errorf("attendu addr '203.0.113.10:7051', obtenu %q", capturedAddr)
	}
	if !strings.Contains(buf.String(), "ajouté") {
		t.Errorf("attendu 'ajouté' dans la sortie, obtenu : %s", buf.String())
	}
}

/// @brief  Ajout nominal d'un orderer à un canal Fabric (UCADM03 flux orderer)
/// @input  type=orderer, addr=orderer.example.com:7050
/// @expect AddNode appelé avec NodeTypeOrderer
func TestRunNodeAdd_Orderer_NominalCase(t *testing.T) {
	var capturedType channel.NodeType
	chSvc := &mockChannelSvc{
		addNode: func(channelID string, nodeType channel.NodeType, addr, orgMSP string, certs channel.NodeCerts) error {
			capturedType = nodeType
			return nil
		},
	}

	withNodeAddFlags("orderer", "orderer.example.com:7050", "OrdererMSP", "", "", func() {
		if err := runNodeAdd(&bytes.Buffer{}, chSvc, activeNetSvc()); err != nil {
			t.Fatalf("runNodeAdd orderer: %v", err)
		}
	})

	if capturedType != channel.NodeTypeOrderer {
		t.Errorf("attendu NodeTypeOrderer, obtenu %q", capturedType)
	}
}

/// @brief  Type de nœud invalide retourne une erreur de validation
/// @input  type="relay" (invalide)
/// @expect erreur "type de nœud invalide", AddNode n'est pas appelé
func TestRunNodeAdd_InvalidType_Rejected(t *testing.T) {
	called := false
	chSvc := &mockChannelSvc{
		addNode: func(_ string, _ channel.NodeType, _, _ string, _ channel.NodeCerts) error {
			called = true
			return nil
		},
	}

	withNodeAddFlags("relay", "203.0.113.10:7051", "Org2MSP", "", "", func() {
		err := runNodeAdd(&bytes.Buffer{}, chSvc, activeNetSvc())
		if err == nil || !strings.Contains(err.Error(), "type de nœud invalide") {
			t.Errorf("attendu erreur 'type de nœud invalide', obtenu %v", err)
		}
	})

	if called {
		t.Error("AddNode ne doit pas être appelé avec un type invalide")
	}
}

/// @brief  Ajout avec certificat TLS — fichier lu et transmis dans NodeCerts
/// @input  type=peer, cert=fichier PEM temporaire
/// @expect NodeCerts.TLSCert contient le PEM du fichier
func TestRunNodeAdd_WithCert_NominalCase(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "tls.pem")
	pemContent := "-----BEGIN CERTIFICATE-----\nMOCK\n-----END CERTIFICATE-----\n"
	if err := os.WriteFile(certPath, []byte(pemContent), 0o600); err != nil {
		t.Fatal(err)
	}

	var capturedCerts channel.NodeCerts
	chSvc := &mockChannelSvc{
		addNode: func(_ string, _ channel.NodeType, _, _ string, certs channel.NodeCerts) error {
			capturedCerts = certs
			return nil
		},
	}

	withNodeAddFlags("peer", "203.0.113.10:7051", "Org2MSP", "", certPath, func() {
		if err := runNodeAdd(&bytes.Buffer{}, chSvc, activeNetSvc()); err != nil {
			t.Fatalf("runNodeAdd avec cert: %v", err)
		}
	})

	if capturedCerts.TLSCert != pemContent {
		t.Errorf("attendu TLSCert %q, obtenu %q", pemContent, capturedCerts.TLSCert)
	}
}

/// @brief  Fichier cert introuvable retourne une erreur
/// @input  cert="/tmp/nonexistent-myr-tls.pem"
/// @expect erreur "certificat TLS introuvable"
func TestRunNodeAdd_CertNotFound_Rejected(t *testing.T) {
	chSvc := &mockChannelSvc{}

	withNodeAddFlags("peer", "203.0.113.10:7051", "Org2MSP", "", "/tmp/nonexistent-myr-tls.pem", func() {
		err := runNodeAdd(&bytes.Buffer{}, chSvc, activeNetSvc())
		if err == nil || !strings.Contains(err.Error(), "certificat TLS introuvable") {
			t.Errorf("attendu erreur 'certificat TLS introuvable', obtenu %v", err)
		}
	})
}

/// @brief  Nœud inaccessible retourne l'erreur ErrNodeUnreachable traduite
/// @input  Service retourne ErrNodeUnreachable
/// @expect erreur "impossible de joindre"
func TestRunNodeAdd_NodeUnreachable_Rejected(t *testing.T) {
	chSvc := &mockChannelSvc{
		addNode: func(_ string, _ channel.NodeType, _, _ string, _ channel.NodeCerts) error {
			return channel.ErrNodeUnreachable
		},
	}

	withNodeAddFlags("peer", "203.0.113.10:7051", "Org2MSP", "", "", func() {
		err := runNodeAdd(&bytes.Buffer{}, chSvc, activeNetSvc())
		if err == nil || !strings.Contains(err.Error(), "impossible de joindre") {
			t.Errorf("attendu erreur 'impossible de joindre', obtenu %v", err)
		}
	})
}

/// @brief  ErrSyncTimeout affiche un avertissement mais ne retourne pas d'erreur
/// @input  Service retourne ErrSyncTimeout
/// @expect avertissement affiché, pas d'erreur retournée
func TestRunNodeAdd_SyncTimeout_Warning(t *testing.T) {
	chSvc := &mockChannelSvc{
		addNode: func(_ string, _ channel.NodeType, _, _ string, _ channel.NodeCerts) error {
			return channel.ErrSyncTimeout
		},
	}

	var buf bytes.Buffer
	withNodeAddFlags("peer", "203.0.113.10:7051", "Org2MSP", "", "", func() {
		if err := runNodeAdd(&buf, chSvc, activeNetSvc()); err != nil {
			t.Fatalf("attendu nil pour SyncTimeout, obtenu %v", err)
		}
	})

	if !strings.Contains(buf.String(), "Avertissement") {
		t.Errorf("attendu avertissement SyncTimeout dans la sortie, obtenu : %s", buf.String())
	}
}

/// @brief  Adapter Fabric non configuré retourne ErrFabricUnavailable traduit
/// @input  Service retourne ErrFabricUnavailable
/// @expect erreur "adaptateur blockchain non configuré"
func TestRunNodeAdd_FabricUnavailable_Rejected(t *testing.T) {
	chSvc := &mockChannelSvc{
		addNode: func(_ string, _ channel.NodeType, _, _ string, _ channel.NodeCerts) error {
			return channel.ErrFabricUnavailable
		},
	}

	withNodeAddFlags("peer", "203.0.113.10:7051", "Org2MSP", "", "", func() {
		err := runNodeAdd(&bytes.Buffer{}, chSvc, activeNetSvc())
		if err == nil || !strings.Contains(err.Error(), "adaptateur blockchain non configuré") {
			t.Errorf("attendu erreur 'adaptateur blockchain non configuré', obtenu %v", err)
		}
	})
}

/// @brief  Canal résolu via --channel explicite surcharge le réseau actif (DC-CLI-01)
/// @input  --channel "greenchannel", réseau actif sur "sandbox"
/// @expect AddNode appelé avec channelID "greenchannel"
func TestRunNodeAdd_ExplicitChannel(t *testing.T) {
	var capturedChannelID string
	chSvc := &mockChannelSvc{
		addNode: func(channelID string, _ channel.NodeType, _, _ string, _ channel.NodeCerts) error {
			capturedChannelID = channelID
			return nil
		},
	}

	withNodeAddFlags("peer", "203.0.113.10:7051", "Org2MSP", "greenchannel", "", func() {
		if err := runNodeAdd(&bytes.Buffer{}, chSvc, activeNetSvc()); err != nil {
			t.Fatalf("runNodeAdd: %v", err)
		}
	})

	if capturedChannelID != "greenchannel" {
		t.Errorf("attendu channelID 'greenchannel', obtenu %q", capturedChannelID)
	}
}

// ── runNodeRemove ─────────────────────────────────────────────────────────────

/// @brief  Retrait nominal d'un nœud membre du canal (UCADM04 flux nominal)
/// @input  addr=203.0.113.10:7051, canal=sandbox
/// @expect message "retiré" affiché, RemoveNode appelé avec les bons paramètres
func TestRunNodeRemove_NominalCase(t *testing.T) {
	var capturedAddr string
	chSvc := &mockChannelSvc{
		removeNode: func(channelID, addr string) error {
			capturedAddr = addr
			return nil
		},
	}

	var buf bytes.Buffer
	withNodeRemoveFlags("203.0.113.10:7051", "", func() {
		if err := runNodeRemove(&buf, chSvc, activeNetSvc()); err != nil {
			t.Fatalf("runNodeRemove: %v", err)
		}
	})

	if capturedAddr != "203.0.113.10:7051" {
		t.Errorf("attendu addr '203.0.113.10:7051', obtenu %q", capturedAddr)
	}
	if !strings.Contains(buf.String(), "retiré") {
		t.Errorf("attendu 'retiré' dans la sortie, obtenu : %s", buf.String())
	}
}

/// @brief  Nœud non membre du canal retourne l'erreur ErrNodeNotMember traduite
/// @input  Service retourne ErrNodeNotMember
/// @expect erreur mentionnant l'addr et le canal
func TestRunNodeRemove_NodeNotMember_Rejected(t *testing.T) {
	chSvc := &mockChannelSvc{
		removeNode: func(channelID, addr string) error {
			return channel.ErrNodeNotMember
		},
	}

	withNodeRemoveFlags("203.0.113.10:7051", "", func() {
		err := runNodeRemove(&bytes.Buffer{}, chSvc, activeNetSvc())
		if err == nil || !strings.Contains(err.Error(), "n'est pas membre actif") {
			t.Errorf("attendu erreur 'n'est pas membre actif', obtenu %v", err)
		}
	})
}

/// @brief  Retrait refusé si seuil minimum 3 nœuds actifs (RM27)
/// @input  Service retourne ErrMinNodesRequired
/// @expect erreur mentionnant RM27 et seuil minimum
func TestRunNodeRemove_RM27_MinNodesRequired_Rejected(t *testing.T) {
	chSvc := &mockChannelSvc{
		removeNode: func(channelID, addr string) error {
			return channel.ErrMinNodesRequired
		},
	}

	withNodeRemoveFlags("203.0.113.10:7051", "", func() {
		err := runNodeRemove(&bytes.Buffer{}, chSvc, activeNetSvc())
		if err == nil || !strings.Contains(err.Error(), "3 nœuds") {
			t.Errorf("attendu erreur seuil minimum 3 nœuds, obtenu %v", err)
		}
		if err != nil && !strings.Contains(err.Error(), "RM27") {
			t.Errorf("attendu mention 'RM27' dans l'erreur, obtenu %v", err)
		}
	})
}

/// @brief  Adapter Fabric non configuré retourne ErrFabricUnavailable traduit
/// @input  Service retourne ErrFabricUnavailable
/// @expect erreur "adaptateur blockchain non configuré"
func TestRunNodeRemove_FabricUnavailable_Rejected(t *testing.T) {
	chSvc := &mockChannelSvc{
		removeNode: func(channelID, addr string) error {
			return channel.ErrFabricUnavailable
		},
	}

	withNodeRemoveFlags("203.0.113.10:7051", "", func() {
		err := runNodeRemove(&bytes.Buffer{}, chSvc, activeNetSvc())
		if err == nil || !strings.Contains(err.Error(), "adaptateur blockchain non configuré") {
			t.Errorf("attendu erreur 'adaptateur blockchain non configuré', obtenu %v", err)
		}
	})
}

/// @brief  Politique d'endorsement non satisfaite retourne l'erreur appropriée
/// @input  Service retourne ErrEndorsementPolicy
/// @expect erreur "politique d'endorsement non satisfaite"
func TestRunNodeRemove_EndorsementPolicy_Rejected(t *testing.T) {
	chSvc := &mockChannelSvc{
		removeNode: func(channelID, addr string) error {
			return channel.ErrEndorsementPolicy
		},
	}

	withNodeRemoveFlags("203.0.113.10:7051", "", func() {
		err := runNodeRemove(&bytes.Buffer{}, chSvc, activeNetSvc())
		if err == nil || !strings.Contains(err.Error(), "politique d'endorsement") {
			t.Errorf("attendu erreur 'politique d'endorsement', obtenu %v", err)
		}
	})
}

/// @brief  Canal non configuré si aucun réseau actif et --channel absent
/// @input  getActive retourne nil, channelFlag=""
/// @expect erreur "canal non configuré"
func TestRunNodeRemove_NoChannel_Rejected(t *testing.T) {
	chSvc := &mockChannelSvc{}
	netSvc := &mockNetworkSvc{
		getActive: func() (*network.NetworkProfile, error) { return nil, nil },
	}

	withNodeRemoveFlags("203.0.113.10:7051", "", func() {
		err := runNodeRemove(&bytes.Buffer{}, chSvc, netSvc)
		if err == nil || !strings.Contains(err.Error(), "canal non configuré") {
			t.Errorf("attendu erreur 'canal non configuré', obtenu %v", err)
		}
	})
}

// ── runNodeProvision ──────────────────────────────────────────────────────────

// withNodeProvisionFlags positionne les variables globales cobra pour la durée de fn,
// puis les restaure — même pattern que withOrgAddFlags / withNodeAddFlags.
func withNodeProvisionFlags(nodeID, hostname, secret, outDir, netID string, fn func()) {
	prevID, prevHostname, prevSecret, prevOut, prevNetwork :=
		nodeProvisionID, nodeProvisionHostname, nodeProvisionSecret, nodeProvisionOutDir, nodeProvisionNetwork
	nodeProvisionID = nodeID
	nodeProvisionHostname = hostname
	nodeProvisionSecret = secret
	nodeProvisionOutDir = outDir
	nodeProvisionNetwork = netID
	fn()
	nodeProvisionID, nodeProvisionHostname, nodeProvisionSecret, nodeProvisionOutDir, nodeProvisionNetwork =
		prevID, prevHostname, prevSecret, prevOut, prevNetwork
}

// withNetworkSvc injecte un mock dans la variable globale networkSvc pour
// la durée de fn, puis restaure la valeur précédente.
func withNetworkSvc(svc network.NetworkService, fn func()) {
	prev := networkSvc
	networkSvc = svc
	fn()
	networkSvc = prev
}

// sampleCreds retourne des PeerCredentials avec tous les champs remplis, y
// compris un layout de fichiers générique représentatif de celui que
// produirait un adapter (ex: adapters/out/fabric).
func sampleCreds(nodeID string) *network.PeerCredentials {
	return &network.PeerCredentials{
		PeerID:   nodeID,
		SignCert: "-----BEGIN CERTIFICATE-----\nSIGN-MOCK\n-----END CERTIFICATE-----\n",
		SignKey:  "-----BEGIN PRIVATE KEY-----\nSIGN-KEY\n-----END PRIVATE KEY-----\n",
		CACert:   "-----BEGIN CERTIFICATE-----\nCA-MOCK\n-----END CERTIFICATE-----\n",
		TLSCert:  "-----BEGIN CERTIFICATE-----\nTLS-MOCK\n-----END CERTIFICATE-----\n",
		TLSKey:   "-----BEGIN PRIVATE KEY-----\nTLS-KEY\n-----END PRIVATE KEY-----\n",
		Files: map[string]string{
			filepath.Join("msp", "signcerts", "cert.pem"): "-----BEGIN CERTIFICATE-----\nSIGN-MOCK\n-----END CERTIFICATE-----\n",
			filepath.Join("msp", "keystore", "key.pem"):   "-----BEGIN PRIVATE KEY-----\nSIGN-KEY\n-----END PRIVATE KEY-----\n",
			filepath.Join("msp", "cacerts", "ca.pem"):     "-----BEGIN CERTIFICATE-----\nCA-MOCK\n-----END CERTIFICATE-----\n",
			filepath.Join("tls", "server.crt"):            "-----BEGIN CERTIFICATE-----\nTLS-MOCK\n-----END CERTIFICATE-----\n",
			filepath.Join("tls", "server.key"):            "-----BEGIN PRIVATE KEY-----\nTLS-KEY\n-----END PRIVATE KEY-----\n",
		},
		StartupInstructions: "Étapes suivantes : démarrez le nœud avec <out>.",
	}
}

// dummyCmd retourne une commande cobra minimale pour appeler runNodeProvision.
func dummyCmd() *cobra.Command { return &cobra.Command{} }

/// @brief  Provisioning nominal d'un nœud : le provisioner retourne des credentials valides, fichiers écrits sur disque
/// @input  nodeID="node1.org1.example.com", outDir=t.TempDir(), AddPeer retourne sampleCreds
/// @expect nil retourné, tous les fichiers listés dans creds.Files créés sur disque
func TestRunNodeProvision_NominalCase(t *testing.T) {
	dir := t.TempDir()
	creds := sampleCreds("node1.org1.example.com")

	svc := &mockNetworkSvc{
		addPeer: func(_ string, req network.AddPeerRequest) (*network.PeerCredentials, error) {
			return creds, nil
		},
	}

	var err error
	withNetworkSvc(svc, func() {
		withNodeProvisionFlags("node1.org1.example.com", "", "", dir, "", func() {
			err = runNodeProvision(dummyCmd(), nil)
		})
	})

	if err != nil {
		t.Fatalf("runNodeProvision: %v", err)
	}

	for rel := range creds.Files {
		if _, statErr := os.Stat(filepath.Join(dir, rel)); os.IsNotExist(statErr) {
			t.Errorf("fichier attendu manquant : %s", rel)
		}
	}
}

/// @brief  outDir absent → répertoire nommé d'après le node-id
/// @input  nodeID="node1.org1.example.com", outDir=""
/// @expect nil retourné, répertoire "node1.org1.example.com" créé dans le CWD
func TestRunNodeProvision_DefaultOutDir(t *testing.T) {
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(orig) }()

	creds := sampleCreds("node1.org1.example.com")
	svc := &mockNetworkSvc{
		addPeer: func(_ string, _ network.AddPeerRequest) (*network.PeerCredentials, error) {
			return creds, nil
		},
	}

	var runErr error
	withNetworkSvc(svc, func() {
		withNodeProvisionFlags("node1.org1.example.com", "", "", "", "", func() {
			runErr = runNodeProvision(dummyCmd(), nil)
		})
	})

	if runErr != nil {
		t.Fatalf("runNodeProvision avec outDir vide: %v", runErr)
	}
	if _, statErr := os.Stat(filepath.Join(tmp, "node1.org1.example.com")); os.IsNotExist(statErr) {
		t.Errorf("répertoire %q non créé", "node1.org1.example.com")
	}
}

/// @brief  AddPeer retourne une erreur → runNodeProvision la propage, aucun fichier créé
/// @input  AddPeer retourne "CA unreachable", outDir=t.TempDir()
/// @expect erreur contenant "CA unreachable", aucun fichier créé
func TestRunNodeProvision_ServiceError_Rejected(t *testing.T) {
	dir := t.TempDir()
	svc := &mockNetworkSvc{
		addPeer: func(_ string, _ network.AddPeerRequest) (*network.PeerCredentials, error) {
			return nil, fmt.Errorf("CA unreachable")
		},
	}

	var err error
	withNetworkSvc(svc, func() {
		withNodeProvisionFlags("node1.org1.example.com", "", "", dir, "", func() {
			err = runNodeProvision(dummyCmd(), nil)
		})
	})

	if err == nil || !strings.Contains(err.Error(), "CA unreachable") {
		t.Errorf("attendu erreur 'CA unreachable', obtenu %v", err)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Error("aucun fichier ne doit être créé si AddPeer échoue")
	}
}

/// @brief  networkSvc nil → runNodeProvision retourne une erreur explicite (protection guard)
/// @input  networkSvc = nil
/// @expect erreur "service réseau non disponible"
func TestRunNodeProvision_NilService_Rejected(t *testing.T) {
	var err error
	withNetworkSvc(nil, func() {
		withNodeProvisionFlags("node1.org1.example.com", "", "", t.TempDir(), "", func() {
			err = runNodeProvision(dummyCmd(), nil)
		})
	})

	if err == nil || !strings.Contains(err.Error(), "service réseau non disponible") {
		t.Errorf("attendu erreur 'service réseau non disponible', obtenu %v", err)
	}
}

/// @brief  Hostname et node-id correctement transmis à AddPeer dans AddPeerRequest
/// @input  nodeID="node1.org1.example.com", hostname="192.168.1.10"
/// @expect AddPeer reçoit PeerID="node1.org1.example.com" et Hostname="192.168.1.10"
func TestRunNodeProvision_RequestFieldsTransmitted(t *testing.T) {
	dir := t.TempDir()
	var captured network.AddPeerRequest

	svc := &mockNetworkSvc{
		addPeer: func(_ string, req network.AddPeerRequest) (*network.PeerCredentials, error) {
			captured = req
			return sampleCreds(req.PeerID), nil
		},
	}

	withNetworkSvc(svc, func() {
		withNodeProvisionFlags("node1.org1.example.com", "192.168.1.10", "mysecret", dir, "net-1", func() {
			_ = runNodeProvision(dummyCmd(), nil)
		})
	})

	if captured.PeerID != "node1.org1.example.com" {
		t.Errorf("PeerID: got %q, want %q", captured.PeerID, "node1.org1.example.com")
	}
	if captured.Hostname != "192.168.1.10" {
		t.Errorf("Hostname: got %q, want %q", captured.Hostname, "192.168.1.10")
	}
	if captured.Secret != "mysecret" {
		t.Errorf("Secret: got %q, want %q", captured.Secret, "mysecret")
	}
}

/// @brief  writeNodeCredentialFiles écrit chaque entrée de creds.Files avec le bon contenu, sans connaître leur mise en page
/// @input  outDir=t.TempDir(), sampleCreds avec Files rempli
/// @expect chaque fichier de creds.Files existe avec le contenu attendu
func TestWriteNodeCredentialFiles_CreatesExpectedFiles(t *testing.T) {
	dir := t.TempDir()
	creds := sampleCreds("node1.org1.example.com")

	if err := writeNodeCredentialFiles(dir, creds); err != nil {
		t.Fatalf("writeNodeCredentialFiles: %v", err)
	}

	for rel, want := range creds.Files {
		data, err := os.ReadFile(filepath.Join(dir, rel))
		if err != nil {
			t.Errorf("fichier manquant %s: %v", rel, err)
			continue
		}
		if string(data) != want {
			t.Errorf("contenu inattendu dans %s\ngot  %q\nwant %q", rel, string(data), want)
		}
	}
}

/// @brief  writeNodeCredentialFiles crée les répertoires parents si absents (mkdir -p)
/// @input  outDir imbriqué inexistant "nested/node-output"
/// @expect pas d'erreur, fichiers créés dans le sous-répertoire
func TestWriteNodeCredentialFiles_CreatesParentDirs(t *testing.T) {
	base := t.TempDir()
	subDir := filepath.Join(base, "nested", "node-output")
	creds := sampleCreds("node2.org1.example.com")

	if err := writeNodeCredentialFiles(subDir, creds); err != nil {
		t.Fatalf("writeNodeCredentialFiles avec sous-répertoire inexistant: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(subDir, "msp", "signcerts", "cert.pem")); statErr != nil {
		t.Errorf("cert.pem non créé dans répertoire imbriqué: %v", statErr)
	}
}

/// @brief  writeNodeCredentialFiles écrit les fichiers avec permissions 0600 (Linux/macOS uniquement)
/// @input  outDir=t.TempDir(), sampleCreds
/// @expect tous les fichiers ont des permissions 0600 (lecture seule propriétaire)
func TestWriteNodeCredentialFiles_FilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permissions POSIX 0600 non applicables sous Windows — validées sur le serveur Linux")
	}
	dir := t.TempDir()
	creds := sampleCreds("node1.org1.example.com")
	if err := writeNodeCredentialFiles(dir, creds); err != nil {
		t.Fatalf("writeNodeCredentialFiles: %v", err)
	}

	for rel := range creds.Files {
		p := filepath.Join(dir, rel)
		info, err := os.Stat(p)
		if err != nil {
			t.Errorf("stat %s: %v", p, err)
			continue
		}
		if mode := info.Mode().Perm(); mode != 0o600 {
			t.Errorf("permissions %s: got %o, want 0600", p, mode)
		}
	}
}
