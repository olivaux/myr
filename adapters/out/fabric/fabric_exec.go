// adapters/out/fabric/fabric_exec.go
// Wrapper os/exec minimal vers les binaires Fabric déjà installés sur le serveur
// (~/fabric/bin : peer, orderer, configtxgen, configtxlator, osnadmin, fabric-ca-server,
// fabric-ca-client, cryptogen). Toute interaction Fabric qui ne peut pas passer par
// le Gateway SDK (config de canal, bootstrap) passe par ici — jamais depuis le domaine.
package fabric

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// fabricBase résout le répertoire racine de l'installation Fabric sur ce serveur.
// Priorité : FABRIC_BASE (env) > ~/fabric.
func fabricBase() string {
	if v := os.Getenv("FABRIC_BASE"); v != "" {
		return v
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "fabric")
}

// fabricBin retourne le chemin absolu vers un binaire Fabric (~/fabric/bin/<name>).
func fabricBin(name string) string {
	return filepath.Join(fabricBase(), "bin", name)
}

// runResult capture la sortie d'une commande exécutée.
type runResult struct {
	Stdout string
	Stderr string
}

// runBin exécute un binaire Fabric avec les arguments donnés. dir est le répertoire
// de travail (vide = répertoire courant). env ajoute des variables au-delà de celles
// du process courant (ex: FABRIC_CFG_PATH, CORE_PEER_*).
func runBin(ctx context.Context, name string, args []string, dir string, env []string) (runResult, error) {
	bin := fabricBin(name)
	if _, err := os.Stat(bin); err != nil {
		return runResult{}, fmt.Errorf("binaire Fabric introuvable : %s (vérifiez FABRIC_BASE)", bin)
	}
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return runResult{Stdout: stdout.String(), Stderr: stderr.String()},
			fmt.Errorf("%s %v : %w — %s", name, args, err, stderr.String())
	}
	return runResult{Stdout: stdout.String(), Stderr: stderr.String()}, nil
}

// nodeOUsConfigYAML active la classification NodeOUs (client/peer/admin/orderer)
// d'une MSP à partir de l'OU du certificat — Fabric CA définit automatiquement
// cet OU selon le "Type" utilisé à l'enregistrement (client/peer/orderer).
// Sans ce fichier, une MSP retombe en mode legacy où seul admincerts distingue
// un admin — aucune identité "peer" ou "client" ordinaire ne peut alors
// satisfaire une politique de canal du type OR('MSPID.admin','MSPID.peer','MSPID.client').
const nodeOUsConfigYAML = `NodeOUs:
  Enable: true
  ClientOUIdentifier:
    Certificate: cacerts/ca.pem
    OrganizationalUnitIdentifier: client
  PeerOUIdentifier:
    Certificate: cacerts/ca.pem
    OrganizationalUnitIdentifier: peer
  AdminOUIdentifier:
    Certificate: cacerts/ca.pem
    OrganizationalUnitIdentifier: admin
  OrdererOUIdentifier:
    Certificate: cacerts/ca.pem
    OrganizationalUnitIdentifier: orderer
`

// retry ré-essaie fn jusqu'à attempts fois avec un délai fixe entre chaque essai.
// Utilisé pour les premières interactions avec un service tout juste démarré
// (conteneur Docker) : `docker-proxy` accepte la connexion TCP côté hôte avant
// même que le process interne (fabric-ca-server, orderer, peer) n'ait fini son
// initialisation — un simple test TCP n'est donc pas fiable comme readiness check.
func retry(attempts int, delay time.Duration, fn func() error) error {
	var lastErr error
	for i := 0; i < attempts; i++ {
		if err := fn(); err == nil {
			return nil
		} else {
			lastErr = err
		}
		time.Sleep(delay)
	}
	return lastErr
}

// checkOsnadminStatus détecte les échecs HTTP renvoyés par `osnadmin` : contrairement
// à `peer`/`configtxgen`/`configtxlator`, `osnadmin` sort toujours avec le code 0 et
// se contente d'imprimer "Status: <code>" suivi du corps JSON — une erreur HTTP
// (400/500...) n'est donc JAMAIS détectée via le code de sortie, seulement en
// relisant sa sortie standard.
func checkOsnadminStatus(res runResult) error {
	firstLine, _, _ := strings.Cut(res.Stdout, "\n")
	firstLine = strings.TrimSpace(firstLine)
	var code int
	if _, err := fmt.Sscanf(firstLine, "Status: %d", &code); err != nil {
		return nil // format inattendu — ne pas bloquer sur une évolution du CLI
	}
	if code >= 300 {
		return fmt.Errorf("osnadmin a répondu %s", strings.TrimSpace(res.Stdout))
	}
	return nil
}

// isAlreadyJoinedErr détecte les erreurs "channel/ledger déjà rejoint" que
// peer/osnadmin renvoient si un essai précédent (notamment via retry) a en
// réalité réussi côté serveur avant qu'une erreur transitoire ne survienne
// côté client — un cas à traiter comme un succès, pas comme un échec.
func isAlreadyJoinedErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "already exists") || strings.Contains(msg, "already joined")
}

// waitCAReady attend que la Fabric CA réponde effectivement aux requêtes HTTPS
// (endpoint public /cainfo, sans authentification) — contrairement à un simple
// test TCP, ceci garantit que fabric-ca-server a terminé son initialisation
// (génération de sa clé/certificat racine) et pas seulement que le port Docker
// est mappé.
func waitCAReady(caURL string, timeout time.Duration) error {
	client := &http.Client{
		Timeout:   5 * time.Second,
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}, //nolint:gosec — vérification de disponibilité uniquement
	}
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		resp, err := client.Get(caURL + "/cainfo")
		if err == nil {
			resp.Body.Close()
			return nil
		}
		lastErr = err
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("CA non prête après %s : %w", timeout, lastErr)
}

// runDocker exécute `docker` ou `docker compose` avec les arguments donnés.
func runDocker(ctx context.Context, dir string, args ...string) (runResult, error) {
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return runResult{Stdout: stdout.String(), Stderr: stderr.String()},
			fmt.Errorf("docker %v : %w — %s", args, err, stderr.String())
	}
	return runResult{Stdout: stdout.String(), Stderr: stderr.String()}, nil
}
