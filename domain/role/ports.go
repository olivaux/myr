// domain/role/ports.go — port de sortie du domaine role
package role

// Repo persiste les rôles.
type Repo interface {
	Save(r *Role) error
	FindAll() ([]*Role, error)
	FindByName(name string) (*Role, error)
	Delete(name string) error
}
