// adapters/out/ipfs/integration/ipfs_integration_test.go
//
// Tests d'intégration contre un vrai daemon IPFS (kubo).
// Le test est automatiquement ignoré (t.Skip) si le daemon n'est pas accessible.
//
// Lancer avec :
//   go test -tags integration -v -timeout 60s ./adapters/out/ipfs/integration/...
//
// Prérequis : daemon kubo en écoute sur http://127.0.0.1:5001
//   (ou surcharger via IPFS_API_ENDPOINT=http://host:port)

//go:build integration

package ipfs_integration_test

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"myr/adapters/out/ipfs"
)

// checkDaemon vérifie que le daemon IPFS est accessible ; skip si absent.
func checkDaemon(t *testing.T) {
	t.Helper()
	cfg := ipfs.ConfigFromEnv()
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Post(cfg.APIEndpoint+"/api/v0/id", "", nil)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Skipf("daemon IPFS non accessible sur %s — skip (démarrer kubo ou définir IPFS_API_ENDPOINT)", cfg.APIEndpoint)
	}
	resp.Body.Close()
}

func newStorage(t *testing.T) *ipfs.IPFSStorage {
	t.Helper()
	return ipfs.New(ipfs.ConfigFromEnv())
}

// ── Upload / Download / Delete ────────────────────────────────────────────────

/// @brief  Cycle complet upload → download → delete sur un vrai daemon IPFS
/// @input  Fichier temporaire de 1 KiB ; daemon kubo local
/// @expect CID non vide ; contenu téléchargé identique à l'original ; dépinnage sans erreur
func TestIPFS_UploadDownloadDelete(t *testing.T) {
	checkDaemon(t)
	storage := newStorage(t)

	// Créer un fichier temporaire avec contenu identifiable
	content := fmt.Sprintf("myr-integration-test %s %d", t.Name(), time.Now().UnixNano())
	srcPath := filepath.Join(t.TempDir(), "upload.txt")
	if err := os.WriteFile(srcPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	// Upload → CID
	cid, err := storage.Upload(srcPath)
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if cid == "" {
		t.Fatal("CID vide après upload")
	}
	t.Logf("CID obtenu : %s", cid)

	// Download → vérification du contenu
	destPath := filepath.Join(t.TempDir(), "downloaded.txt")
	if err := storage.Download(cid, destPath); err != nil {
		t.Fatalf("Download: %v", err)
	}
	data, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("lecture fichier téléchargé : %v", err)
	}
	if string(data) != content {
		t.Errorf("contenu: got %q, want %q", string(data), content)
	}

	// Delete (dépinner)
	if err := storage.Delete(cid); err != nil {
		t.Errorf("Delete: %v", err)
	}
	t.Logf("CID %s dépinné avec succès", cid)
}

/// @brief  Uploader deux fois le même contenu produit le même CID (déduplication IPFS)
/// @input  Même fichier soumis deux fois
/// @expect Les deux CIDs sont identiques (adressage par contenu)
func TestIPFS_UploadSameFileTwice_ReturnsSameCID(t *testing.T) {
	checkDaemon(t)
	storage := newStorage(t)

	srcPath := filepath.Join(t.TempDir(), "dedup.txt")
	if err := os.WriteFile(srcPath, []byte("contenu idempotent ipfs"), 0600); err != nil {
		t.Fatal(err)
	}

	cid1, err := storage.Upload(srcPath)
	if err != nil {
		t.Fatalf("Upload 1: %v", err)
	}
	cid2, err := storage.Upload(srcPath)
	if err != nil {
		t.Fatalf("Upload 2: %v", err)
	}
	if cid1 != cid2 {
		t.Errorf("CIDs différents pour le même contenu : %q vs %q", cid1, cid2)
	}
	t.Logf("CID idempotent confirmé : %s", cid1)
}

/// @brief  Uploader un fichier binaire (non-texte) fonctionne correctement
/// @input  Fichier de 4 KiB rempli de bytes aléatoires (0x00–0xFF)
/// @expect CID non vide ; contenu identique après download
func TestIPFS_UploadBinaryFile(t *testing.T) {
	checkDaemon(t)
	storage := newStorage(t)

	// Générer 4 KiB de données binaires déterministes
	content := make([]byte, 4096)
	for i := range content {
		content[i] = byte(i % 256)
	}
	srcPath := filepath.Join(t.TempDir(), "binary.bin")
	if err := os.WriteFile(srcPath, content, 0600); err != nil {
		t.Fatal(err)
	}

	cid, err := storage.Upload(srcPath)
	if err != nil {
		t.Fatalf("Upload binaire: %v", err)
	}

	destPath := filepath.Join(t.TempDir(), "binary-dl.bin")
	if err := storage.Download(cid, destPath); err != nil {
		t.Fatalf("Download binaire: %v", err)
	}
	data, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("lecture: %v", err)
	}
	if len(data) != len(content) {
		t.Errorf("taille: got %d, want %d", len(data), len(content))
	}
	for i, b := range data {
		if b != content[i] {
			t.Errorf("byte[%d]: got 0x%02x, want 0x%02x", i, b, content[i])
			break
		}
	}
	t.Logf("fichier binaire 4 KiB round-trip OK, CID=%s", cid)
}

// ── Cas d'erreur ──────────────────────────────────────────────────────────────

/// @brief  Télécharger un CID inexistant retourne une erreur
/// @input  CID syntaxiquement valide mais absent du réseau
/// @expect retourne une erreur non nil
func TestIPFS_Download_UnknownCID_Errors(t *testing.T) {
	checkDaemon(t)
	storage := newStorage(t)

	// CID formaté comme CIDv1 mais avec contenu inexistant
	fakeCID := "bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi"
	destPath := filepath.Join(t.TempDir(), "out.bin")
	err := storage.Download(fakeCID, destPath)
	if err == nil {
		t.Error("attendu une erreur pour un CID inexistant")
	} else {
		t.Logf("erreur attendue reçue : %v", err)
	}
}

/// @brief  Dépinner un CID inconnu est idempotent (retourne nil)
/// @input  CID inexistant sur le daemon local
/// @expect nil (le daemon répond 500 "not pinned" — comportement idempotent documenté)
func TestIPFS_Delete_NotPinned_IsIdempotent(t *testing.T) {
	checkDaemon(t)
	storage := newStorage(t)

	// CID qui n'a jamais été pinné localement
	if err := storage.Delete("bafybeiczsscdsbs7ffqz55asqdf3smv6klcw3gofszvwlyarci47bgf354"); err != nil {
		t.Errorf("Delete CID non pinné : attendu nil (idempotent), obtenu %v", err)
	}
}

/// @brief  Uploader un fichier inexistant retourne une erreur
/// @input  chemin "/no/such/file.bin"
/// @expect retourne une erreur d'ouverture de fichier
func TestIPFS_Upload_MissingFile_Errors(t *testing.T) {
	checkDaemon(t)
	storage := newStorage(t)

	_, err := storage.Upload("/no/such/file-ipfs-integration.bin")
	if err == nil {
		t.Error("attendu une erreur pour un fichier source inexistant")
	}
}
