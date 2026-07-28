// adapters/out/fabric/tests/peer_provisioner_test.go
// Vérifie que l'adapter Fabric — et lui seul — porte la connaissance du
// layout MSP/TLS et des instructions de démarrage d'un peer : le CLI ne doit
// jamais avoir à connaître cette mise en page.
package fabric_test

import (
	"path/filepath"
	"strings"
	"testing"

	"myr-core/adapters/out/fabric"
	"myr-core/domain/network"
)

// / @brief  PeerMSPFiles construit le layout de fichiers Fabric (MSP + TLS) à partir des credentials générés
// / @input  PeerCredentials avec tous les champs PEM remplis
// / @expect les 5 chemins attendus sont présents avec le contenu correspondant
func TestPeerMSPFiles_ExpectedLayout(t *testing.T) {
	creds := &network.PeerCredentials{
		PeerID:   "node1.org1.example.com",
		SignCert: "SIGN-CERT",
		SignKey:  "SIGN-KEY",
		CACert:   "CA-CERT",
		TLSCert:  "TLS-CERT",
		TLSKey:   "TLS-KEY",
	}

	files := fabric.PeerMSPFiles(creds)

	expected := map[string]string{
		filepath.Join("msp", "signcerts", "cert.pem"): "SIGN-CERT",
		filepath.Join("msp", "keystore", "key.pem"):   "SIGN-KEY",
		filepath.Join("msp", "cacerts", "ca.pem"):     "CA-CERT",
		filepath.Join("tls", "server.crt"):            "TLS-CERT",
		filepath.Join("tls", "server.key"):            "TLS-KEY",
	}
	if len(files) != len(expected) {
		t.Fatalf("attendu %d fichiers, obtenu %d", len(expected), len(files))
	}
	for rel, want := range expected {
		got, ok := files[rel]
		if !ok {
			t.Errorf("fichier manquant : %s", rel)
			continue
		}
		if got != want {
			t.Errorf("%s : got %q, want %q", rel, got, want)
		}
	}
}

// / @brief  PeerStartupInstructions produit un texte contenant l'identité du peer et le placeholder <out> à substituer par le CLI
// / @input  peerID="node1.org1.example.com"
// / @expect le texte contient le peerID et le placeholder "<out>"
func TestPeerStartupInstructions_ContainsPeerIDAndPlaceholder(t *testing.T) {
	text := fabric.PeerStartupInstructions("node1.org1.example.com")

	if !strings.Contains(text, "node1.org1.example.com") {
		t.Errorf("attendu l'identité du peer dans les instructions, obtenu : %s", text)
	}
	if !strings.Contains(text, "<out>") {
		t.Errorf("attendu le placeholder '<out>' (substitué par le CLI), obtenu : %s", text)
	}
}
