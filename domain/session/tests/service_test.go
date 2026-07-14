// domain/session/tests/service_test.go — tests unitaires du service session
package session_test

import (
	"errors"
	"testing"

	"myr/domain/session"
)

// ── Mock Store ────────────────────────────────────────────────────────────────

type mockStore struct {
	stored   *session.Session
	saveErr  error
	loadErr  error
	clearErr error
}

func (m *mockStore) Save(s *session.Session) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.stored = s
	return nil
}

func (m *mockStore) Load() (*session.Session, error) {
	if m.loadErr != nil {
		return nil, m.loadErr
	}
	return m.stored, nil
}

func (m *mockStore) Clear() error {
	if m.clearErr != nil {
		return m.clearErr
	}
	m.stored = nil
	return nil
}

// ── Create ────────────────────────────────────────────────────────────────────

// / @brief  Création d'une session avec nom et orgID valides
// / @input  mockStore vide, nom "alice", orgID "Org1MSP"
// / @expect retourne une session avec le bon nom, orgID et CreatedAt non nul
func TestCreate_Success(t *testing.T) {
	store := &mockStore{}
	svc := session.NewService(store)

	sess, err := svc.Create("alice", "Org1MSP")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if sess.Name != "alice" {
		t.Errorf("Name: got %q, want alice", sess.Name)
	}
	if sess.OrgID != "Org1MSP" {
		t.Errorf("OrgID: got %q, want Org1MSP", sess.OrgID)
	}
	if sess.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
}

// / @brief  Création refusée si le nom est vide
// / @input  mockStore vide, nom vide
// / @expect retourne une erreur de validation
func TestCreate_EmptyName(t *testing.T) {
	svc := session.NewService(&mockStore{})
	_, err := svc.Create("", "Org1MSP")
	if err == nil {
		t.Error("want error for empty name")
	}
}

// / @brief  Création réussie sans orgID (champ optionnel)
// / @input  mockStore vide, nom "alice", orgID vide
// / @expect retourne une session avec orgID vide, sans erreur
func TestCreate_OrgIDOptional(t *testing.T) {
	svc := session.NewService(&mockStore{})
	sess, err := svc.Create("alice", "")
	if err != nil {
		t.Fatalf("Create without orgID: %v", err)
	}
	if sess.OrgID != "" {
		t.Errorf("OrgID: got %q, want empty", sess.OrgID)
	}
}

// / @brief  Création échoue si le store retourne une erreur à la sauvegarde
// / @input  mockStore avec saveErr "disk full"
// / @expect retourne une erreur propagée depuis le store
func TestCreate_StoreError(t *testing.T) {
	store := &mockStore{saveErr: errors.New("disk full")}
	svc := session.NewService(store)
	_, err := svc.Create("alice", "Org1MSP")
	if err == nil {
		t.Error("want error when store.Save fails")
	}
}

// ── Current ───────────────────────────────────────────────────────────────────

// / @brief  Récupération de la session courante stockée
// / @input  mockStore avec session {Name:"bob", OrgID:"Org2MSP"} préchargée
// / @expect retourne la session "bob" sans erreur
func TestCurrent_ReturnsSession(t *testing.T) {
	store := &mockStore{stored: &session.Session{Name: "bob", OrgID: "Org2MSP"}}
	svc := session.NewService(store)

	sess, err := svc.Current()
	if err != nil {
		t.Fatalf("Current: %v", err)
	}
	if sess == nil || sess.Name != "bob" {
		t.Errorf("got %v, want session bob", sess)
	}
}

// / @brief  Récupération retourne nil si aucune session n'est stockée
// / @input  mockStore vide (stored nil)
// / @expect retourne nil sans erreur
func TestCurrent_NoSession(t *testing.T) {
	svc := session.NewService(&mockStore{})
	sess, err := svc.Current()
	if err != nil {
		t.Fatalf("Current: %v", err)
	}
	if sess != nil {
		t.Errorf("got %v, want nil when no session", sess)
	}
}

// / @brief  Récupération de la session courante échoue si le store est en erreur
// / @input  mockStore avec loadErr "corrupted"
// / @expect retourne une erreur propagée depuis le store
func TestCurrent_StoreError(t *testing.T) {
	store := &mockStore{loadErr: errors.New("corrupted")}
	svc := session.NewService(store)
	_, err := svc.Current()
	if err == nil {
		t.Error("want error when store.Load fails")
	}
}

// ── Logout ────────────────────────────────────────────────────────────────────

// / @brief  Déconnexion efface la session courante
// / @input  mockStore avec session "alice" préchargée
// / @expect session nil après déconnexion, sans erreur
func TestLogout_ClearsSession(t *testing.T) {
	store := &mockStore{stored: &session.Session{Name: "alice"}}
	svc := session.NewService(store)

	if err := svc.Logout(); err != nil {
		t.Fatalf("Logout: %v", err)
	}
	sess, _ := svc.Current()
	if sess != nil {
		t.Error("session should be nil after logout")
	}
}

// / @brief  Déconnexion échoue si le store retourne une erreur au vidage
// / @input  mockStore avec clearErr "permission denied"
// / @expect retourne une erreur propagée depuis le store
func TestLogout_StoreError(t *testing.T) {
	store := &mockStore{clearErr: errors.New("permission denied")}
	svc := session.NewService(store)
	if err := svc.Logout(); err == nil {
		t.Error("want error when store.Clear fails")
	}
}
