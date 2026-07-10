// domain/role/entity.go
package role

import "time"

// Permission identifie une action protégée (routes REST / commandes CLI).
// Catalogue dérivé du découpage déjà existant dans adapters/in/rest/server.go
// (groupes contrib/auth/adminOnly) — pas une granularité par endpoint individuel.
type Permission string

const (
	// PermRead couvre la lecture authentifiée (GET /api/* pour un utilisateur connecté).
	PermRead Permission = "read"
	// PermWrite couvre l'écriture des assets (components, connections, modules,
	// interfaces, licences — le groupe "contributor" existant de server.go).
	PermWrite Permission = "write"
	// PermNetworkAdmin couvre la création/gestion de réseaux, organisations et nœuds
	// (myr network/org/node — UCADM01-05).
	PermNetworkAdmin Permission = "network.admin"
	// PermRoleAdmin couvre la gestion des rôles eux-mêmes (myr role ...).
	PermRoleAdmin Permission = "role.admin"
	// PermIdentityAdmin couvre la gestion des identités/comptes (myr identity set-role,
	// validation des demandes de compte).
	PermIdentityAdmin Permission = "identity.admin"
	// PermAdmin couvre les opérations globales (/api/admin/*).
	PermAdmin Permission = "admin"
)

// AllPermissions liste le catalogue complet, utilisé pour valider les entrées CLI.
var AllPermissions = []Permission{PermRead, PermWrite, PermNetworkAdmin, PermRoleAdmin, PermIdentityAdmin, PermAdmin}

// IsValidPermission vérifie qu'une permission appartient au catalogue connu.
func IsValidPermission(p string) bool {
	for _, known := range AllPermissions {
		if string(known) == p {
			return true
		}
	}
	return false
}

// Role représente un rôle attribuable à une organisation ou un utilisateur,
// avec un ensemble de permissions personnalisable (RBAC dynamique).
type Role struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"` // ex: "reader", "contributor", "consumer"
	Permissions []Permission `json:"permissions"`
	// BuiltIn = true pour les 4 rôles historiques (reader/contributor/auditor/admin) —
	// non supprimables, pour ne pas casser la compatibilité existante.
	BuiltIn   bool      `json:"built_in"`
	CreatedAt time.Time `json:"created_at"`
}

// Has indique si le rôle porte la permission donnée.
func (r *Role) Has(p Permission) bool {
	for _, existing := range r.Permissions {
		if existing == p {
			return true
		}
	}
	return false
}

// Noms des 4 rôles historiques (domain/identity.Role{Reader,Contributor,Auditor,Admin}).
const (
	NameReader      = "reader"
	NameContributor = "contributor"
	NameAuditor     = "auditor"
	NameAdmin       = "admin"
)
