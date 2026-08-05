// domain/identity/service.go
package identity

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Service struct {
	walletDir    string // ~/.Myr/wallets/
	ca           CAPort
	requestStore RequestStore // nil = demandes non persistées
}

func NewService(walletDir string, ca CAPort) *Service {
	return &Service{walletDir: walletDir, ca: ca}
}

// WithRequestStore injecte le store de persistance des demandes de compte.
func (s *Service) WithRequestStore(rs RequestStore) *Service {
	s.requestStore = rs
	return s
}

// WalletDir retourne le chemin du répertoire des wallets locaux.
func (s *Service) WalletDir() string { return s.walletDir }

// NewWithCA retourne un nouveau Service avec le même walletDir mais un CA différent.
func (s *Service) NewWithCA(ca CAPort) *Service {
	return &Service{walletDir: s.walletDir, ca: ca}
}

// ListLocalWallets parcourt ~/.Myr/wallets/ et retourne les wallets trouvés.
func (s *Service) ListLocalWallets() ([]WalletEntry, error) {
	entries, err := os.ReadDir(s.walletDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var wallets []WalletEntry
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		handle := e.Name() // ex: alice@Org1
		mspDir := filepath.Join(s.walletDir, handle, "msp")

		// Lire le statut depuis le certificat local
		status := StatusPending
		certPath := filepath.Join(mspDir, "signcerts", "cert.pem")
		if data, err := os.ReadFile(certPath); err == nil {
			if st := statusFromCert(data); st != "" {
				status = st
			}
		}

		name, orgID := parseHandle(handle)
		wallets = append(wallets, WalletEntry{
			Handle: handle,
			Name:   name,
			OrgID:  orgID,
			Status: status,
			MSPDir: mspDir,
		})
	}
	return wallets, nil
}

// Register enregistre une nouvelle identité auprès de la CA.
func (s *Service) Register(ctx context.Context, req RegisterRequest) (string, error) {
	if s.ca == nil {
		return "", fmt.Errorf("aucun CA configuré — ajoutez un ca_endpoint au réseau actif")
	}
	return s.ca.Register(ctx, req)
}

// Enroll effectue l'enrollment et sauvegarde le wallet localement.
func (s *Service) Enroll(ctx context.Context, name, secret, orgID string) (WalletEntry, error) {
	if s.ca == nil {
		return WalletEntry{}, fmt.Errorf("aucun CA configuré")
	}
	// L'identifiant CA d'une identité auto-enregistrée est "pseudo@org" (voir
	// AutoRegister) — l'authentification HTTP Basic de l'enrollment doit viser
	// ce même identifiant, pas le pseudo seul, sous peine de "user not found"
	// côté CA dès que le client suit la forme documentée {name, secret, org_id}.
	handle := name + "@" + strings.TrimSuffix(orgID, "MSP")
	certPEM, keyPEM, caCertPEM, err := s.ca.Enroll(ctx, handle, secret)
	if err != nil {
		return WalletEntry{}, fmt.Errorf("enrollment : %w", err)
	}

	mspDir := filepath.Join(s.walletDir, handle, "msp")
	if err := saveWallet(mspDir, certPEM, keyPEM, caCertPEM); err != nil {
		return WalletEntry{}, fmt.Errorf("sauvegarde wallet : %w", err)
	}

	return WalletEntry{
		Handle: handle,
		Name:   name,
		OrgID:  orgID,
		Status: StatusPending,
		MSPDir: mspDir,
	}, nil
}

// GetStatus interroge la CA pour obtenir le statut courant de l'identité.
func (s *Service) GetStatus(ctx context.Context, wallet WalletEntry) (string, error) {
	if s.ca == nil {
		return "", fmt.Errorf("aucun CA configuré")
	}
	// Comme pour Enroll : l'identifiant CA est "pseudo@org" (wallet.Handle),
	// pas le pseudo seul (wallet.Name) — sans quoi la CA ne retrouve pas
	// l'identité (même cause que le bug d'enrollment corrigé ci-dessus).
	return s.ca.GetStatus(ctx, wallet.Handle)
}

// LoadGuestWallet charge un wallet invité depuis des fichiers de certificat existants
// (pré-enrôlés par l'admin) et le sauvegarde localement sous le handle "guest@<org>".
// Le wallet invité a Myr.role=reader : il permet la consultation sans écriture.
// caCertPath est optionnel (chaîne vide acceptée).
func (s *Service) LoadGuestWallet(certPath, keyPath, caCertPath, orgID string) (WalletEntry, error) {
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return WalletEntry{}, fmt.Errorf("certificat invité introuvable (%s) : %w", certPath, err)
	}
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return WalletEntry{}, fmt.Errorf("clé invité introuvable (%s) : %w", keyPath, err)
	}
	var caCertPEM string
	if caCertPath != "" {
		if data, err := os.ReadFile(caCertPath); err == nil {
			caCertPEM = string(data)
		}
	}

	org := strings.TrimSuffix(orgID, "MSP")
	handle := "guest@" + org
	mspDir := filepath.Join(s.walletDir, handle, "msp")
	if err := saveWallet(mspDir, string(certPEM), string(keyPEM), caCertPEM); err != nil {
		return WalletEntry{}, fmt.Errorf("sauvegarde wallet invité : %w", err)
	}
	return WalletEntry{
		Handle: handle,
		Name:   "guest",
		OrgID:  orgID,
		Status: StatusActive,
		MSPDir: mspDir,
	}, nil
}

// ReEnroll re-enrôle l'identité pour récupérer un certificat mis à jour.
func (s *Service) ReEnroll(ctx context.Context, wallet WalletEntry) (WalletEntry, error) {
	if s.ca == nil {
		return WalletEntry{}, fmt.Errorf("aucun CA configuré")
	}
	certPath := filepath.Join(wallet.MSPDir, "signcerts", "cert.pem")
	keyPath, err := findKeyFile(filepath.Join(wallet.MSPDir, "keystore"))
	if err != nil {
		return WalletEntry{}, fmt.Errorf("clé privée introuvable : %w", err)
	}
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return WalletEntry{}, fmt.Errorf("certificat local introuvable : %w", err)
	}
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return WalletEntry{}, fmt.Errorf("clé privée illisible : %w", err)
	}

	newCert, err := s.ca.ReEnroll(ctx, wallet.Name, string(certPEM), string(keyPEM))
	if err != nil {
		return WalletEntry{}, fmt.Errorf("re-enrollment : %w", err)
	}

	// Remplacer le certificat
	err = os.WriteFile(certPath, []byte(newCert), 0600)
	if err != nil {
		return WalletEntry{}, fmt.Errorf("mise à jour certificat : %w", err)
	}

	status := StatusPending
	if st := statusFromCert([]byte(newCert)); st != "" {
		status = st
	}
	wallet.Status = status
	return wallet, nil
}

// SubmitRequest enregistre une demande d'accès soumise par un inconnu.
func (s *Service) SubmitRequest(req AccountRequest) (*AccountRequest, error) {
	if req.Pseudo == "" || req.Email == "" || req.OrgID == "" {
		return nil, fmt.Errorf("pseudo, email et org_id sont requis pour une demande de compte")
	}
	req.ID = fmt.Sprintf("req-%d", time.Now().UnixNano())
	req.Status = RequestPending
	req.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	if s.requestStore != nil {
		if err := s.requestStore.Save(&req); err != nil {
			return nil, fmt.Errorf("enregistrement demande : %w", err)
		}
	}
	return &req, nil
}

// AutoRegister enregistre automatiquement l'identité dans la CA Fabric et retourne
// le secret d'enrollment. L'identité est créée avec Myr.status=pending, Myr.role=reader.
// L'admin peut ensuite passer le statut à "active" via la CA pour activer le compte.
// req doit être la demande déjà persistée par SubmitRequest (ID renseigné) : en cas de
// succès, son statut est mis à jour en RequestApproved et réécrit via le requestStore,
// pour que la demande n'apparaisse plus comme "pending" alors que l'identité est active.
func (s *Service) AutoRegister(ctx context.Context, req *AccountRequest, role string) (string, error) {
	if s.ca == nil {
		return "", fmt.Errorf("aucun CA configuré — impossible d'enregistrer automatiquement")
	}
	org := strings.TrimSuffix(req.OrgID, "MSP")
	regReq := RegisterRequest{
		Name:        req.Pseudo + "@" + org,
		OrgID:       req.OrgID,
		Role:        role,
		DisplayName: req.DisplayName,
		Email:       req.Email,
	}
	secret, err := s.ca.Register(ctx, regReq)
	if err != nil {
		return "", err
	}
	req.Status = RequestApproved
	if s.requestStore != nil {
		if err := s.requestStore.Save(req); err != nil {
			return "", fmt.Errorf("persistance statut demande : %w", err)
		}
	}
	return secret, nil
}

// SetRole change le rôle enregistré d'une identité existante auprès de la CA
// (attribut Myr.role). Le nouveau rôle ne s'applique qu'aux certificats émis
// après l'appel — l'identité doit se ré-enrôler (ReEnroll) pour l'obtenir dans
// son certificat actif : c'est une propriété de la Fabric CA, pas une
// limitation de myr.
func (s *Service) SetRole(ctx context.Context, name, newRole string) error {
	if s.ca == nil {
		return fmt.Errorf("aucun CA configuré — impossible de modifier le rôle")
	}
	if name == "" {
		return fmt.Errorf("identité requise")
	}
	if newRole == "" {
		return fmt.Errorf("nouveau rôle requis")
	}
	return s.ca.UpdateAttributes(ctx, name, map[string]string{"Myr.role": newRole})
}

// ListRequests retourne toutes les demandes enregistrées.
func (s *Service) ListRequests() ([]*AccountRequest, error) {
	if s.requestStore == nil {
		return nil, nil
	}
	return s.requestStore.FindAll()
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func saveWallet(mspDir, certPEM, keyPEM, caCertPEM string) error {
	dirs := []string{
		filepath.Join(mspDir, "signcerts"),
		filepath.Join(mspDir, "keystore"),
		filepath.Join(mspDir, "cacerts"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0700); err != nil {
			return err
		}
	}
	writes := map[string]string{
		filepath.Join(mspDir, "signcerts", "cert.pem"): certPEM,
		filepath.Join(mspDir, "keystore", "key.pem"):   keyPEM,
		filepath.Join(mspDir, "cacerts", "ca.pem"):     caCertPEM,
	}
	for path, data := range writes {
		if err := os.WriteFile(path, []byte(data), 0600); err != nil {
			return err
		}
	}
	return nil
}

func statusFromCert(certPEM []byte) string {
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return ""
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return ""
	}
	// Les attributs Myr.status sont dans les extensions OID personnalisées
	// ou dans les Subject Alternative Names selon la config de la CA.
	// Pour l'instant on lit le SubjectCommonName et on cherche les extensions.
	_ = cert
	return "" // sera enrichi quand le format d'attribut CA sera connu
}

func parseHandle(handle string) (name, orgID string) {
	parts := strings.SplitN(handle, "@", 2)
	if len(parts) == 2 {
		return parts[0], parts[1] + "MSP"
	}
	return handle, ""
}

func findKeyFile(keystoreDir string) (string, error) {
	entries, err := os.ReadDir(keystoreDir)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if !e.IsDir() {
			return filepath.Join(keystoreDir, e.Name()), nil
		}
	}
	return "", fmt.Errorf("aucun fichier de clé dans %s", keystoreDir)
}
