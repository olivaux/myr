// domain/identity/port_in.go
package identity

import "context"

// IdentityService est le port d'entrée du domaine identity.
// Implémenté par le service, consommé par les adapters in (CLI, REST).
type IdentityService interface {
	ListLocalWallets() ([]WalletEntry, error)
	Register(ctx context.Context, req RegisterRequest) (secret string, err error)
	Enroll(ctx context.Context, name, secret, orgID string) (WalletEntry, error)
	GetStatus(ctx context.Context, wallet WalletEntry) (string, error)
	ReEnroll(ctx context.Context, wallet WalletEntry) (WalletEntry, error)
	// LoadGuestWallet charge un wallet invité (lecture seule) depuis des fichiers
	// pré-enrôlés par l'admin, sans passer par la CA.
	LoadGuestWallet(certPath, keyPath, caCertPath, orgID string) (WalletEntry, error)
	// SubmitRequest enregistre une demande d'accès soumise par un inconnu.
	// L'admin peut ensuite la consulter et créer le compte dans la CA Fabric.
	SubmitRequest(req AccountRequest) (*AccountRequest, error)
	// AutoRegister enregistre automatiquement l'identité dans la CA Fabric
	// (requiert des credentials admin dans la config) et retourne le secret d'enrollment.
	// role définit les droits attribués (ex: RoleContributor) ; vide = RoleReader.
	// Utilisé quand AllowAutoRegister=true dans le profil réseau actif.
	AutoRegister(ctx context.Context, req AccountRequest, role string) (secret string, err error)
	// ListRequests retourne toutes les demandes en attente (usage admin).
	ListRequests() ([]*AccountRequest, error)
	WalletDir() string
	// SetRole change le rôle enregistré d'une identité existante auprès de la CA.
	// Le nouveau rôle ne s'applique qu'aux prochains certificats émis — un
	// ré-enrôlement (ReEnroll) est nécessaire pour l'appliquer immédiatement.
	SetRole(ctx context.Context, name, newRole string) error
}
