// domain/role/tests/service_test.go — tests unitaires du service RBAC
package role_test

import (
	"testing"

	"myr-core/domain/role"
)

// ── Mock Repo (Pattern A — struct concrète, état en mémoire) ─────────────────

type mockRepo struct {
	roles map[string]*role.Role
}

func newMockRepo() *mockRepo {
	return &mockRepo{roles: make(map[string]*role.Role)}
}

func (m *mockRepo) Save(r *role.Role) error {
	m.roles[r.Name] = r
	return nil
}

func (m *mockRepo) FindAll() ([]*role.Role, error) {
	out := make([]*role.Role, 0, len(m.roles))
	for _, r := range m.roles {
		out = append(out, r)
	}
	return out, nil
}

func (m *mockRepo) FindByName(name string) (*role.Role, error) {
	r, ok := m.roles[name]
	if !ok {
		return nil, nil
	}
	return r, nil
}

func (m *mockRepo) Delete(name string) error {
	delete(m.roles, name)
	return nil
}

// ── Seed des rôles intégrés ───────────────────────────────────────────────────

// / @brief  NewService amorce les 4 rôles intégrés dans un repo vide
// / @input  mockRepo vide
// / @expect List retourne reader, contributor, auditor, admin, tous BuiltIn=true
func TestNewService_SeedsBuiltinRoles(t *testing.T) {
	repo := newMockRepo()
	svc := role.NewService(repo)

	for _, name := range []string{role.NameReader, role.NameContributor, role.NameAuditor, role.NameAdmin} {
		r, err := svc.Get(name)
		if err != nil {
			t.Fatalf("Get(%q): %v", name, err)
		}
		if !r.BuiltIn {
			t.Errorf("%q: attendu BuiltIn=true", name)
		}
	}
}

// / @brief  NewService est idempotent : ne réécrase pas des rôles intégrés déjà présents
// / @input  mockRepo contenant déjà "admin" avec des permissions personnalisées
// / @expect les permissions existantes de "admin" ne sont pas réinitialisées
func TestNewService_SeedIdempotent(t *testing.T) {
	repo := newMockRepo()
	repo.roles[role.NameAdmin] = &role.Role{Name: role.NameAdmin, Permissions: []role.Permission{role.PermRead}, BuiltIn: true}
	role.NewService(repo)

	r, _ := repo.FindByName(role.NameAdmin)
	if len(r.Permissions) != 1 || r.Permissions[0] != role.PermRead {
		t.Errorf("le seed ne devrait pas écraser un rôle intégré déjà présent, got %v", r.Permissions)
	}
}

// ── Create ────────────────────────────────────────────────────────────────────

// / @brief  Create ajoute un rôle personnalisé avec l'ensemble de permissions donné
// / @input  mockRepo amorcé, nom "consumer", permissions [read]
// / @expect le rôle est créé, non intégré, avec les permissions fournies
func TestCreate_Success(t *testing.T) {
	svc := role.NewService(newMockRepo())
	r, err := svc.Create("consumer", []role.Permission{role.PermRead})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if r.BuiltIn {
		t.Error("un rôle créé via Create ne doit pas être BuiltIn")
	}
	if !r.Has(role.PermRead) {
		t.Error("le rôle devrait porter PermRead")
	}
}

// / @brief  Create refuse un nom déjà utilisé (y compris un rôle intégré)
// / @input  mockRepo amorcé, tentative de Create("reader", ...)
// / @expect retourne ErrAlreadyExists
func TestCreate_AlreadyExists(t *testing.T) {
	svc := role.NewService(newMockRepo())
	_, err := svc.Create(role.NameReader, []role.Permission{role.PermRead})
	if err == nil {
		t.Fatal("erreur attendue pour un nom déjà utilisé")
	}
}

// / @brief  Create refuse un nom invalide (majuscules, caractères interdits)
// / @input  mockRepo amorcé, nom "Consumer!"
// / @expect retourne ErrInvalidName
func TestCreate_InvalidName(t *testing.T) {
	svc := role.NewService(newMockRepo())
	_, err := svc.Create("Consumer!", []role.Permission{role.PermRead})
	if err == nil {
		t.Fatal("erreur attendue pour un nom invalide")
	}
}

// / @brief  Create refuse une permission inconnue du catalogue
// / @input  mockRepo amorcé, permission "delete-everything"
// / @expect retourne ErrInvalidPermission
func TestCreate_InvalidPermission(t *testing.T) {
	svc := role.NewService(newMockRepo())
	_, err := svc.Create("consumer", []role.Permission{"delete-everything"})
	if err == nil {
		t.Fatal("erreur attendue pour une permission inconnue")
	}
}

// ── Update ────────────────────────────────────────────────────────────────────

// / @brief  Update remplace l'ensemble de permissions d'un rôle personnalisé
// / @input  rôle "consumer" créé avec [read], mise à jour vers [read, write]
// / @expect le rôle porte désormais exactement [read, write]
func TestUpdate_Success(t *testing.T) {
	svc := role.NewService(newMockRepo())
	svc.Create("consumer", []role.Permission{role.PermRead})

	updated, err := svc.Update("consumer", []role.Permission{role.PermRead, role.PermWrite})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if !updated.Has(role.PermWrite) {
		t.Error("le rôle devrait porter PermWrite après mise à jour")
	}
}

// / @brief  Update refuse de modifier un rôle intégré
// / @input  mockRepo amorcé, tentative de Update("admin", ...)
// / @expect retourne ErrBuiltIn
func TestUpdate_BuiltInRejected(t *testing.T) {
	svc := role.NewService(newMockRepo())
	_, err := svc.Update(role.NameAdmin, []role.Permission{role.PermRead})
	if err != role.ErrBuiltIn {
		t.Errorf("erreur attendue ErrBuiltIn, obtenu %v", err)
	}
}

// / @brief  Update refuse un rôle inexistant
// / @input  mockRepo amorcé, tentative de Update("ghost", ...)
// / @expect retourne ErrNotFound
func TestUpdate_NotFound(t *testing.T) {
	svc := role.NewService(newMockRepo())
	_, err := svc.Update("ghost", []role.Permission{role.PermRead})
	if err == nil {
		t.Fatal("erreur attendue pour un rôle inexistant")
	}
}

// ── Delete ────────────────────────────────────────────────────────────────────

// / @brief  Delete supprime un rôle personnalisé
// / @input  rôle "consumer" créé, suppression par nom
// / @expect Get("consumer") retourne ErrNotFound après suppression
func TestDelete_Success(t *testing.T) {
	svc := role.NewService(newMockRepo())
	svc.Create("consumer", []role.Permission{role.PermRead})

	if err := svc.Delete("consumer"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := svc.Get("consumer"); err == nil {
		t.Error("le rôle ne devrait plus exister après Delete")
	}
}

// / @brief  Delete refuse de supprimer un rôle intégré
// / @input  mockRepo amorcé, tentative de Delete("reader")
// / @expect retourne ErrBuiltIn, le rôle reste présent
func TestDelete_BuiltInRejected(t *testing.T) {
	svc := role.NewService(newMockRepo())
	err := svc.Delete(role.NameReader)
	if err != role.ErrBuiltIn {
		t.Errorf("erreur attendue ErrBuiltIn, obtenu %v", err)
	}
	if _, err := svc.Get(role.NameReader); err != nil {
		t.Error("le rôle intégré devrait toujours exister")
	}
}

// ── HasPermission ─────────────────────────────────────────────────────────────

// / @brief  HasPermission reflète fidèlement les permissions par défaut des rôles intégrés
// / @input  mockRepo amorcé
// / @expect reader→read oui/write non ; contributor→read+write oui ; admin→tout oui
func TestHasPermission_BuiltinDefaults(t *testing.T) {
	svc := role.NewService(newMockRepo())

	cases := []struct {
		role string
		perm role.Permission
		want bool
	}{
		{role.NameReader, role.PermRead, true},
		{role.NameReader, role.PermWrite, false},
		{role.NameContributor, role.PermWrite, true},
		{role.NameContributor, role.PermNetworkAdmin, false},
		{role.NameAuditor, role.PermRead, true},
		{role.NameAdmin, role.PermNetworkAdmin, true},
		{role.NameAdmin, role.PermRoleAdmin, true},
	}
	for _, c := range cases {
		got := svc.HasPermission(c.role, c.perm)
		if got != c.want {
			t.Errorf("HasPermission(%q, %q): got %v, want %v", c.role, c.perm, got, c.want)
		}
	}
}

// / @brief  HasPermission retourne false pour un rôle inconnu (fail-closed)
// / @input  mockRepo amorcé, rôle "ghost" jamais créé
// / @expect retourne false sans erreur
func TestHasPermission_UnknownRole(t *testing.T) {
	svc := role.NewService(newMockRepo())
	if svc.HasPermission("ghost", role.PermRead) {
		t.Error("un rôle inconnu ne devrait porter aucune permission")
	}
}

// ── List ──────────────────────────────────────────────────────────────────────

// / @brief  List retourne les 4 rôles intégrés plus les rôles personnalisés créés
// / @input  mockRepo amorcé, création de "consumer" et "manufacturer"
// / @expect List retourne 6 rôles au total
func TestList_IncludesBuiltinAndCustom(t *testing.T) {
	svc := role.NewService(newMockRepo())
	svc.Create("consumer", []role.Permission{role.PermRead})
	svc.Create("manufacturer", []role.Permission{role.PermRead, role.PermWrite})

	all, err := svc.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) != 6 {
		t.Errorf("got %d rôles, want 6 (4 intégrés + 2 personnalisés)", len(all))
	}
}
