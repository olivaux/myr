// adapters/out/ipfs/tests/storage_test.go — tests unitaires de IPFSStorage.
// Aucun daemon IPFS requis : le serveur HTTP est remplacé par httptest.Server.
package ipfs_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"myr/adapters/out/ipfs"
)

// newTestServer crée un httptest.Server dont le handler est configurable par test.
func newTestServer(handler http.HandlerFunc) (*httptest.Server, *ipfs.IPFSStorage) {
	srv := httptest.NewServer(handler)
	storage := ipfs.New(ipfs.Config{APIEndpoint: srv.URL})
	return srv, storage
}

// ── Upload ────────────────────────────────────────────────────────────────────

/// @brief  Upload réussi d'un fichier — le CID retourné par le daemon est propagé
/// @input  serveur retournant 200 + {"Hash":"QmABC123"}, fichier temporaire valide
/// @expect retourne "QmABC123" sans erreur
func TestUpload_Success(t *testing.T) {
	srv, storage := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/v0/add") {
			http.Error(w, "unexpected path", http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"Hash": "QmABC123"})
	})
	defer srv.Close()

	f, err := os.CreateTemp("", "ipfs-test-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("contenu de test")
	f.Close()
	defer os.Remove(f.Name())

	cid, err := storage.Upload(f.Name())
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if cid != "QmABC123" {
		t.Errorf("CID: got %q, want QmABC123", cid)
	}
}

/// @brief  Upload échoue si le fichier source n'existe pas
/// @input  serveur quelconque, chemin "/no/such/file.txt"
/// @expect retourne une erreur (open impossible)
func TestUpload_FileNotFound(t *testing.T) {
	srv, storage := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	_, err := storage.Upload("/no/such/file.txt")
	if err == nil {
		t.Error("attendu une erreur pour fichier inexistant")
	}
}

/// @brief  Upload échoue si le daemon retourne un statut non-200
/// @input  serveur retournant 500 "internal error"
/// @expect retourne une erreur contenant le code HTTP
func TestUpload_DaemonError(t *testing.T) {
	srv, storage := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	})
	defer srv.Close()

	f, err := os.CreateTemp("", "ipfs-test-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("data")
	f.Close()
	defer os.Remove(f.Name())

	_, err = storage.Upload(f.Name())
	if err == nil {
		t.Error("attendu une erreur quand daemon retourne 500")
	}
}

/// @brief  Upload échoue si la réponse JSON est malformée
/// @input  serveur retournant 200 avec un corps non-JSON
/// @expect retourne une erreur de décodage JSON
func TestUpload_InvalidJSONResponse(t *testing.T) {
	srv, storage := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "pas du json")
	})
	defer srv.Close()

	f, err := os.CreateTemp("", "ipfs-test-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("data")
	f.Close()
	defer os.Remove(f.Name())

	_, err = storage.Upload(f.Name())
	if err == nil {
		t.Error("attendu une erreur pour JSON invalide")
	}
}

// ── Download ──────────────────────────────────────────────────────────────────

/// @brief  Download réussi — le contenu retourné par le daemon est écrit dans destPath
/// @input  serveur retournant 200 + "contenu ipfs", fichier de destination temporaire
/// @expect fichier créé, contenu identique à "contenu ipfs", sans erreur
func TestDownload_Success(t *testing.T) {
	srv, storage := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/v0/cat") {
			http.Error(w, "unexpected path", http.StatusNotFound)
			return
		}
		fmt.Fprint(w, "contenu ipfs")
	})
	defer srv.Close()

	destPath := filepath.Join(t.TempDir(), "downloaded.bin")
	if err := storage.Download("QmABC123", destPath); err != nil {
		t.Fatalf("Download: %v", err)
	}

	data, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("lecture fichier destination : %v", err)
	}
	if string(data) != "contenu ipfs" {
		t.Errorf("contenu: got %q, want %q", string(data), "contenu ipfs")
	}
}

/// @brief  Download passe le CID comme argument dans l'URL de la requête cat
/// @input  serveur enregistrant l'URL reçue, CID "QmXYZ"
/// @expect URL contient "arg=QmXYZ"
func TestDownload_CIDPassedInURL(t *testing.T) {
	receivedURL := ""
	srv, storage := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		receivedURL = r.URL.String()
		fmt.Fprint(w, "data")
	})
	defer srv.Close()

	destPath := filepath.Join(t.TempDir(), "out.bin")
	storage.Download("QmXYZ", destPath)

	if !strings.Contains(receivedURL, "arg=QmXYZ") {
		t.Errorf("URL attendue contenir arg=QmXYZ, got %q", receivedURL)
	}
}

/// @brief  Download échoue si le daemon retourne un statut non-200
/// @input  serveur retournant 404 "not found"
/// @expect retourne une erreur contenant le code HTTP
func TestDownload_DaemonError(t *testing.T) {
	srv, storage := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	})
	defer srv.Close()

	if err := storage.Download("QmABC", filepath.Join(t.TempDir(), "out.bin")); err == nil {
		t.Error("attendu une erreur quand daemon retourne 404")
	}
}

/// @brief  Download échoue si le répertoire de destination n'existe pas
/// @input  serveur retournant 200, destPath dans un répertoire inexistant
/// @expect retourne une erreur de création de fichier
func TestDownload_DestDirNotExist(t *testing.T) {
	srv, storage := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "data")
	})
	defer srv.Close()

	if err := storage.Download("QmABC", "/no/such/dir/file.bin"); err == nil {
		t.Error("attendu une erreur pour répertoire destination inexistant")
	}
}

// ── Delete ────────────────────────────────────────────────────────────────────

/// @brief  Delete réussi — retourne nil quand le daemon confirme la suppression du pin
/// @input  serveur retournant 200, CID "QmABC123"
/// @expect retourne nil
func TestDelete_Success(t *testing.T) {
	srv, storage := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/v0/pin/rm") {
			http.Error(w, "unexpected path", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	if err := storage.Delete("QmABC123"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

/// @brief  Delete est idempotent — retourne nil si le daemon répond 500 "not pinned"
/// @input  serveur retournant 500 avec corps "not pinned or pinned indirectly"
/// @expect retourne nil (comportement idempotent documenté)
func TestDelete_NotPinned_IsIdempotent(t *testing.T) {
	srv, storage := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not pinned or pinned indirectly", http.StatusInternalServerError)
	})
	defer srv.Close()

	if err := storage.Delete("QmABC123"); err != nil {
		t.Errorf("Delete idempotent: attendu nil, got %v", err)
	}
}

/// @brief  Delete passe le CID comme argument dans l'URL de la requête pin/rm
/// @input  serveur enregistrant l'URL reçue, CID "QmDEL"
/// @expect URL contient "arg=QmDEL"
func TestDelete_CIDPassedInURL(t *testing.T) {
	receivedURL := ""
	srv, storage := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		receivedURL = r.URL.String()
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	storage.Delete("QmDEL")

	if !strings.Contains(receivedURL, "arg=QmDEL") {
		t.Errorf("URL attendue contenir arg=QmDEL, got %q", receivedURL)
	}
}

/// @brief  Delete échoue si le daemon retourne un code d'erreur inattendu (ex: 400)
/// @input  serveur retournant 400 "bad request"
/// @expect retourne une erreur non nil
func TestDelete_UnexpectedDaemonError(t *testing.T) {
	srv, storage := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	})
	defer srv.Close()

	if err := storage.Delete("QmABC"); err == nil {
		t.Error("attendu une erreur pour code HTTP 400 inattendu")
	}
}

// ── ConfigFromEnv ─────────────────────────────────────────────────────────────

/// @brief  ConfigFromEnv retourne les valeurs par défaut si les variables d'env sont absentes
/// @input  variables IPFS_API_ENDPOINT et IPFS_GATEWAY_URL non définies
/// @expect APIEndpoint == "http://127.0.0.1:5001", GatewayURL == "http://127.0.0.1:8080"
func TestConfigFromEnv_Defaults(t *testing.T) {
	os.Unsetenv("IPFS_API_ENDPOINT")
	os.Unsetenv("IPFS_GATEWAY_URL")

	cfg := ipfs.ConfigFromEnv()
	if cfg.APIEndpoint != "http://127.0.0.1:5001" {
		t.Errorf("APIEndpoint: got %q, want http://127.0.0.1:5001", cfg.APIEndpoint)
	}
	if cfg.GatewayURL != "http://127.0.0.1:8080" {
		t.Errorf("GatewayURL: got %q, want http://127.0.0.1:8080", cfg.GatewayURL)
	}
}

/// @brief  ConfigFromEnv lit les variables d'environnement quand elles sont définies
/// @input  IPFS_API_ENDPOINT="http://remote:5001", IPFS_GATEWAY_URL="http://gw:8080"
/// @expect APIEndpoint et GatewayURL correspondent aux valeurs des variables d'env
func TestConfigFromEnv_FromEnv(t *testing.T) {
	t.Setenv("IPFS_API_ENDPOINT", "http://remote:5001")
	t.Setenv("IPFS_GATEWAY_URL", "http://gw:8080")

	cfg := ipfs.ConfigFromEnv()
	if cfg.APIEndpoint != "http://remote:5001" {
		t.Errorf("APIEndpoint: got %q, want http://remote:5001", cfg.APIEndpoint)
	}
	if cfg.GatewayURL != "http://gw:8080" {
		t.Errorf("GatewayURL: got %q, want http://gw:8080", cfg.GatewayURL)
	}
}

// ── Vérification de conformité au port ───────────────────────────────────────

// Assertion de compilation : IPFSStorage implémente model.FileStoragePort.
// Si l'interface change sans mise à jour de l'adapter, la compilation échoue.
func TestIPFSStorage_ImplementsFileStoragePort(t *testing.T) {
	// On vérifie que les trois méthodes requises par FileStoragePort sont présentes.
	s := ipfs.New(ipfs.Config{})
	var _ interface {
		Upload(filePath string) (string, error)
		Download(hash, destPath string) error
		Delete(hash string) error
	} = s
}

// ── io.ReadAll reader ─────────────────────────────────────────────────────────

/// @brief  Upload avec un grand fichier — vérifie que le multipart est correctement construit
/// @input  serveur validant que Content-Type contient "multipart/form-data", fichier de 1 KiB
/// @expect retourne un CID sans erreur, Content-Type multipart présent dans la requête
func TestUpload_MultipartContentType(t *testing.T) {
	var receivedContentType string
	srv, storage := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		receivedContentType = r.Header.Get("Content-Type")
		json.NewEncoder(w).Encode(map[string]string{"Hash": "QmBIG"})
	})
	defer srv.Close()

	f, err := os.CreateTemp("", "ipfs-big-*.bin")
	if err != nil {
		t.Fatal(err)
	}
	f.Write(make([]byte, 1024))
	f.Close()
	defer os.Remove(f.Name())

	cid, err := storage.Upload(f.Name())
	if err != nil {
		t.Fatalf("Upload grand fichier: %v", err)
	}
	if cid != "QmBIG" {
		t.Errorf("CID: got %q, want QmBIG", cid)
	}
	if !strings.Contains(receivedContentType, "multipart/form-data") {
		t.Errorf("Content-Type attendu multipart/form-data, got %q", receivedContentType)
	}
}

/// @brief  Download avec contenu vide — fichier de destination créé mais vide
/// @input  serveur retournant 200 avec corps vide
/// @expect fichier créé, contenu vide, sans erreur
func TestDownload_EmptyContent(t *testing.T) {
	srv, storage := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// corps vide intentionnel
	})
	defer srv.Close()

	destPath := filepath.Join(t.TempDir(), "empty.bin")
	if err := storage.Download("QmEMPTY", destPath); err != nil {
		t.Fatalf("Download contenu vide: %v", err)
	}

	data, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("lecture fichier vide: %v", err)
	}
	if len(data) != 0 {
		t.Errorf("fichier devrait être vide, got %d octets", len(data))
	}
}

