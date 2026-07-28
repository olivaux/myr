// adapters/in/cli/session_test.go — tests des handlers CLI session (compte local machine)
package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"myr-core/domain/session"
)

// ── mock SessionService (Pattern B — fn-func) ──────────────────────────────────

type mockSessionSvc struct {
	create  func(name, orgID string) (*session.Session, error)
	current func() (*session.Session, error)
	logout  func() error
}

func (m *mockSessionSvc) Create(name, orgID string) (*session.Session, error) {
	if m.create != nil {
		return m.create(name, orgID)
	}
	return &session.Session{Name: name, OrgID: orgID, CreatedAt: time.Now()}, nil
}
func (m *mockSessionSvc) Current() (*session.Session, error) {
	if m.current != nil {
		return m.current()
	}
	return nil, nil
}
func (m *mockSessionSvc) Logout() error {
	if m.logout != nil {
		return m.logout()
	}
	return nil
}

/// @brief  runSessionCreate crée le compte local et affiche nom + org
/// @input  name="alice", orgID="Org1MSP"
/// @expect Create reçoit les deux valeurs, sortie contient nom et org
func TestRunSessionCreate_NominalCase(t *testing.T) {
	var gotName, gotOrg string
	svc := &mockSessionSvc{create: func(name, orgID string) (*session.Session, error) {
		gotName, gotOrg = name, orgID
		return &session.Session{Name: name, OrgID: orgID}, nil
	}}
	var buf bytes.Buffer
	if err := runSessionCreate(&buf, svc, "alice", "Org1MSP"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotName != "alice" || gotOrg != "Org1MSP" {
		t.Fatalf("unexpected forwarded args: name=%q org=%q", gotName, gotOrg)
	}
	out := buf.String()
	if !strings.Contains(out, "alice") || !strings.Contains(out, "Org1MSP") {
		t.Fatalf("unexpected output: %q", out)
	}
}

/// @brief  runSessionCreate propage l'erreur si un compte local existe déjà
/// @input  service retournant une erreur
/// @expect L'erreur est retournée
func TestRunSessionCreate_AlreadyExists_Rejected(t *testing.T) {
	wantErr := errors.New("un compte local existe déjà")
	svc := &mockSessionSvc{create: func(string, string) (*session.Session, error) { return nil, wantErr }}
	var buf bytes.Buffer
	err := runSessionCreate(&buf, svc, "alice", "Org1MSP")
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}

/// @brief  runSessionShow affiche le compte local existant
/// @input  Current() retourne une session
/// @expect Sortie contient le nom du compte
func TestRunSessionShow_NominalCase(t *testing.T) {
	svc := &mockSessionSvc{current: func() (*session.Session, error) {
		return &session.Session{Name: "alice", OrgID: "Org1MSP", CreatedAt: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}, nil
	}}
	var buf bytes.Buffer
	if err := runSessionShow(&buf, svc); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "alice") {
		t.Fatalf("unexpected output: %q", buf.String())
	}
}

/// @brief  runSessionShow affiche un message dédié quand aucun compte local n'existe
/// @input  Current() retourne (nil, nil)
/// @expect Sortie mentionne l'absence de compte, pas d'erreur
func TestRunSessionShow_NoAccount(t *testing.T) {
	svc := &mockSessionSvc{current: func() (*session.Session, error) { return nil, nil }}
	var buf bytes.Buffer
	if err := runSessionShow(&buf, svc); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "Aucun compte local") {
		t.Fatalf("unexpected output: %q", buf.String())
	}
}

/// @brief  runSessionLogout efface le compte local et confirme en sortie
/// @input  service Logout réussi
/// @expect Sortie confirme l'effacement
func TestRunSessionLogout_NominalCase(t *testing.T) {
	called := false
	svc := &mockSessionSvc{logout: func() error { called = true; return nil }}
	var buf bytes.Buffer
	if err := runSessionLogout(&buf, svc); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatalf("expected Logout to be called")
	}
	if !strings.Contains(buf.String(), "effacé") {
		t.Fatalf("unexpected output: %q", buf.String())
	}
}
