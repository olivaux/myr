// domain/session/entity.go
package session

import "time"

// Session représente le compte local actif de l'utilisateur.
// Créé une seule fois au premier lancement, persisté en JSON.
// Les droits sont gérés côté réseau par l'administrateur.
type Session struct {
	Name      string    `json:"name"`
	OrgID     string    `json:"org_id"`
	CreatedAt time.Time `json:"created_at"`
}
