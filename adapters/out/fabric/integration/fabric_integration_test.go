// adapters/out/fabric/fabric_integration_test.go
//
// Tests d'intégration contre un vrai réseau Hyperledger Fabric.
// Nécessite le tag build "integration" et les fichiers dans testdata/.
//
// Lancer avec :
//   go test -tags integration -v -timeout 60s ./adapters/out/fabric/...
//
// Prérequis dans testdata/ :
//   testnet.env     — variables de connexion (FABRIC_*)
//   tls-cert.pem    — certificat TLS du peer
//   client-cert.pem — certificat d'enrôlement de l'identité cliente
//   client-key.pem  — clé privée cliente

//go:build integration

package fabric_test

import (
	"bufio"
	"fmt"
	"myr-core/adapters/out/fabric"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"myr-core/domain/model"
)

// ── helpers ──────────────────────────────────────────────────────────────────

// testdataDir retourne le chemin absolu vers testdata/ (sibling de integration/,
// dans adapters/out/fabric/testdata/). Priorité à MYR_FABRIC_TESTDATA_DIR :
// un binaire de test compilé (`go test -c`) et exécuté à distance (scripts/test_remote.ps1)
// n'a plus le chemin source d'origine — testdata/ est alors copié à côté du binaire
// et son chemin passé explicitement par la variable d'environnement.
func testdataDir(t *testing.T) string {
	t.Helper()
	if dir := os.Getenv("MYR_FABRIC_TESTDATA_DIR"); dir != "" {
		return dir
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("impossible de déterminer le répertoire courant : %v", err)
	}
	return filepath.Join(wd, "..", "testdata")
}

// loadEnvFile lit un fichier .env simple (KEY=VALUE, commentaires #).
func loadEnvFile(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	m := make(map[string]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			m[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return m, scanner.Err()
}

// integrationConfig charge la fabric.Config depuis testdata/testnet.env + chemins PEM relatifs.
func integrationConfig(t *testing.T) fabric.Config {
	t.Helper()
	td := testdataDir(t)
	envPath := filepath.Join(td, "testnet.env")
	env, err := loadEnvFile(envPath)
	if err != nil {
		t.Fatalf("impossible de lire testnet.env : %v", err)
	}
	get := func(key string) string {
		if v, ok := env[key]; ok {
			return v
		}
		return ""
	}
	return fabric.Config{
		PeerEndpoint:  get("FABRIC_PEER_ENDPOINT"),
		GatewayPeer:   get("FABRIC_GATEWAY_PEER"),
		MSPID:         get("FABRIC_MSP_ID"),
		FabricChannel: get("FABRIC_CHANNEL_NAME"),
		ChaincodeName: get("FABRIC_CHAINCODE_NAME"),
		TLSCertPath:   filepath.Join(td, "tls-cert.pem"),
		CertPath:      filepath.Join(td, "client-cert.pem"),
		KeyPath:       filepath.Join(td, "client-key.pem"),
	}
}

// newGateway ouvre une connexion réelle ; le test est skipé si la config est incomplète.
func newGateway(t *testing.T) *fabric.GatewayClient {
	t.Helper()
	cfg := integrationConfig(t)
	if cfg.PeerEndpoint == "" {
		t.Skip("FABRIC_PEER_ENDPOINT non défini dans testnet.env")
	}
	gw, err := fabric.NewGatewayClient(cfg)
	if err != nil {
		t.Fatalf("connexion Fabric échouée : %v", err)
	}
	t.Cleanup(func() { gw.Close() })
	return gw
}

// testAssetID génère un ID unique pour éviter les collisions entre exécutions.
func testAssetID() string {
	return fmt.Sprintf("myr-test-%d", time.Now().UnixNano())
}

// ── Tests ─────────────────────────────────────────────────────────────────────

// TestIntegration_GatewayConnect vérifie que la connexion gRPC + TLS + identité
// s'établit correctement sans erreur.
func TestIntegration_GatewayConnect(t *testing.T) {
	newGateway(t) // échoue le test si la connexion ne s'ouvre pas
	t.Log("connexion Gateway Fabric établie avec succès")
}

// TestIntegration_StoreAndGet écrit un asset puis le relit.
func TestIntegration_StoreAndGet(t *testing.T) {
	gw := newGateway(t)
	bc := fabric.NewFabricBlockchain(gw)

	cfg := integrationConfig(t)
	id := testAssetID()
	asset := &model.Model3D{
		ID:        id,
		Name:      "Test Integration Myr",
		ChannelID: cfg.FabricChannel,
		Hash:      "sha256:integration-test",
	}

	// Écriture
	if err := bc.StoreModelRecord(asset); err != nil {
		t.Fatalf("StoreModelRecord : %v", err)
	}
	t.Logf("asset %q soumis sur le canal %q", id, cfg.FabricChannel)

	// Lecture
	got, err := bc.GetModelRecord(id, "")
	if err != nil {
		t.Fatalf("GetModelRecord : %v", err)
	}
	if got.ID != id {
		t.Errorf("ID attendu %q, obtenu %q", id, got.ID)
	}
	if got.Name != asset.Name {
		t.Errorf("Name attendu %q, obtenu %q", asset.Name, got.Name)
	}
	t.Logf("asset relu : ID=%s Name=%s", got.ID, got.Name)
}

// TestIntegration_ListModelRecords récupère la liste des assets sur le canal.
func TestIntegration_ListModelRecords(t *testing.T) {
	gw := newGateway(t)
	bc := fabric.NewFabricBlockchain(gw)

	cfg := integrationConfig(t)
	list, err := bc.ListModelRecords(cfg.FabricChannel)
	if err != nil {
		t.Fatalf("ListModelRecords : %v", err)
	}
	t.Logf("%d asset(s) trouvé(s) sur le canal %q", len(list), cfg.FabricChannel)
	for _, a := range list {
		t.Logf("  - %s  %s", a.ID, a.Name)
	}
}

// TestIntegration_VerifyIntegrity soumet un asset puis vérifie son hash.
func TestIntegration_VerifyIntegrity(t *testing.T) {
	gw := newGateway(t)
	bc := fabric.NewFabricBlockchain(gw)

	cfg := integrationConfig(t)
	id := testAssetID()
	hash := "sha256:verify-integrity-test"
	asset := &model.Model3D{
		ID:        id,
		Name:      "Integrity Test",
		ChannelID: cfg.FabricChannel,
		Hash:      hash,
	}

	if err := bc.StoreModelRecord(asset); err != nil {
		t.Fatalf("StoreModelRecord : %v", err)
	}

	// Hash correct → true
	ok, err := bc.VerifyIntegrity(id, hash, "")
	if err != nil {
		t.Fatalf("VerifyIntegrity (hash correct) : %v", err)
	}
	if !ok {
		t.Error("VerifyIntegrity : true attendu pour le hash correct, obtenu false")
	}

	// Hash incorrect → false
	ok, err = bc.VerifyIntegrity(id, "sha256:wrong", "")
	if err != nil {
		t.Fatalf("VerifyIntegrity (hash incorrect) : %v", err)
	}
	if ok {
		t.Error("VerifyIntegrity : false attendu pour le hash incorrect, obtenu true")
	}
	t.Logf("intégrité vérifiée pour l'asset %q", id)
}

// TestIntegration_GetModelRecord_NotFound vérifie le comportement sur un ID inexistant.
func TestIntegration_GetModelRecord_NotFound(t *testing.T) {
	gw := newGateway(t)
	bc := fabric.NewFabricBlockchain(gw)

	_, err := bc.GetModelRecord("myr-inexistant-000000000", "")
	if err == nil {
		t.Error("une erreur était attendue pour un ID inexistant, obtenu nil")
	} else {
		t.Logf("erreur attendue reçue : %v", err)
	}
}
