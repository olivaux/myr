// domain/role/port_in.go — port d'entrée du domaine role
package role

// RoleService est le port d'entrée du domaine role (RBAC dynamique).
type RoleService interface {
	// Create crée un nouveau rôle personnalisé avec l'ensemble de permissions donné.
	Create(name string, permissions []Permission) (*Role, error)
	// Update remplace l'ensemble de permissions d'un rôle existant non intégré.
	Update(name string, permissions []Permission) (*Role, error)
	// Delete supprime un rôle personnalisé. Refuse les 4 rôles intégrés (ErrBuiltIn).
	Delete(name string) error
	// Get retourne un rôle par son nom.
	Get(name string) (*Role, error)
	// List retourne tous les rôles (intégrés + personnalisés).
	List() ([]*Role, error)
	// HasPermission indique si le rôle nommé porte la permission donnée.
	// Un rôle inconnu ne porte aucune permission (retourne false, pas d'erreur) —
	// utilisé directement dans les points de contrôle d'accès (REST requireRole).
	HasPermission(roleName string, p Permission) bool
}
