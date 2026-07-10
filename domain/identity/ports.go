// domain/identity/ports.go — ports de sortie du domaine identity
package identity

import "context"

// CAPort est le port de sortie vers l'adapter Fabric CA.
type CAPort interface {
	Register(ctx context.Context, req RegisterRequest) (secret string, err error)
	Enroll(ctx context.Context, name, secret string) (certPEM, keyPEM, caCertPEM string, err error)
	GetStatus(ctx context.Context, name string) (status string, err error)
	ReEnroll(ctx context.Context, name, certPEM, keyPEM string) (newCertPEM string, err error)
	// UpdateAttributes modifie les attributs enregistrés d'une identité existante
	// (ex: Myr.role). Le changement ne s'applique qu'aux certificats émis après
	// l'appel — un ré-enrôlement (ReEnroll) est nécessaire pour qu'il prenne effet
	// dans le certificat actif de l'identité (propriété de la Fabric CA).
	UpdateAttributes(ctx context.Context, name string, attrs map[string]string) error
}

// RequestStore est le port de sortie pour la persistance des demandes de compte.
type RequestStore interface {
	Save(req *AccountRequest) error
	FindAll() ([]*AccountRequest, error)
}
