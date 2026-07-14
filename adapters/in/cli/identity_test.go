// adapters/in/cli/identity_test.go — tests des handlers CLI identité
package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"myr/domain/identity"
	"myr/domain/network"
)

/// @brief  runIdentityWallets liste les wallets locaux
/// @input  service retournant 1 wallet
/// @expect Sortie contient le handle du wallet
func TestRunIdentityWallets_NominalCase(t *testing.T) {
	svc := &mockIdentitySvc{listLocalWallets: func() ([]identity.WalletEntry, error) {
		return []identity.WalletEntry{{Handle: "alice@Org1MSP", Name: "alice", OrgID: "Org1MSP", Status: identity.StatusActive}}, nil
	}}
	var buf bytes.Buffer
	if err := runIdentityWallets(&buf, svc); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "alice@Org1MSP") {
		t.Fatalf("unexpected output: %q", buf.String())
	}
}

/// @brief  runIdentityStatus découpe le handle pseudo@org et interroge la CA
/// @input  handle="alice@Org1"
/// @expect GetStatus reçoit Name="alice" OrgID="Org1MSP", sortie contient le statut
func TestRunIdentityStatus_NominalCase(t *testing.T) {
	var got identity.WalletEntry
	svc := &mockIdentitySvc{getStatus: func(_ context.Context, wallet identity.WalletEntry) (string, error) {
		got = wallet
		return identity.StatusActive, nil
	}}
	var buf bytes.Buffer
	if err := runIdentityStatus(&buf, svc, "alice@Org1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "alice" || got.OrgID != "Org1MSP" {
		t.Fatalf("unexpected wallet forwarded: %+v", got)
	}
	if !strings.Contains(buf.String(), "active") {
		t.Fatalf("unexpected output: %q", buf.String())
	}
}

/// @brief  runIdentityStatus rejette un handle sans "@"
/// @input  handle="alice"
/// @expect Erreur retournée, GetStatus jamais appelé
func TestRunIdentityStatus_InvalidHandle_Rejected(t *testing.T) {
	called := false
	svc := &mockIdentitySvc{getStatus: func(context.Context, identity.WalletEntry) (string, error) {
		called = true
		return "", nil
	}}
	var buf bytes.Buffer
	if err := runIdentityStatus(&buf, svc, "alice"); err == nil {
		t.Fatalf("expected error for invalid handle")
	}
	if called {
		t.Fatalf("GetStatus should not be called for an invalid handle")
	}
}

/// @brief  runIdentityEnroll enrôle l'identité et affiche le handle + statut du wallet
/// @input  name="alice", secret="s3cr3t", orgID="Org1MSP"
/// @expect Enroll reçoit les 3 valeurs, sortie contient le handle
func TestRunIdentityEnroll_NominalCase(t *testing.T) {
	var gotName, gotSecret, gotOrg string
	svc := &mockIdentitySvc{enroll: func(_ context.Context, name, secret, orgID string) (identity.WalletEntry, error) {
		gotName, gotSecret, gotOrg = name, secret, orgID
		return identity.WalletEntry{Handle: name + "@" + orgID, Status: identity.StatusPending}, nil
	}}
	var buf bytes.Buffer
	err := runIdentityEnroll(&buf, svc, "alice", "s3cr3t", "Org1MSP")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotName != "alice" || gotSecret != "s3cr3t" || gotOrg != "Org1MSP" {
		t.Fatalf("unexpected forwarded args: name=%q secret=%q org=%q", gotName, gotSecret, gotOrg)
	}
	if !strings.Contains(buf.String(), "alice@Org1MSP") {
		t.Fatalf("unexpected output: %q", buf.String())
	}
}

/// @brief  runIdentityEnroll propage une erreur de secret invalide
/// @input  service retournant une erreur
/// @expect L'erreur est retournée
func TestRunIdentityEnroll_InvalidSecret_Rejected(t *testing.T) {
	wantErr := errors.New("secret invalide")
	svc := &mockIdentitySvc{enroll: func(context.Context, string, string, string) (identity.WalletEntry, error) {
		return identity.WalletEntry{}, wantErr
	}}
	var buf bytes.Buffer
	err := runIdentityEnroll(&buf, svc, "alice", "bad", "Org1MSP")
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}

/// @brief  runIdentityReEnroll retrouve le wallet local par handle et le ré-enrôle
/// @input  handle="alice@Org1MSP" présent dans ListLocalWallets
/// @expect ReEnroll est appelé avec ce wallet, sortie confirme le nouveau statut
func TestRunIdentityReEnroll_NominalCase(t *testing.T) {
	wallet := identity.WalletEntry{Handle: "alice@Org1MSP", Name: "alice", OrgID: "Org1MSP", MSPDir: "/tmp/msp"}
	var gotWallet identity.WalletEntry
	svc := &mockIdentitySvc{
		listLocalWallets: func() ([]identity.WalletEntry, error) { return []identity.WalletEntry{wallet}, nil },
		reEnroll: func(_ context.Context, w identity.WalletEntry) (identity.WalletEntry, error) {
			gotWallet = w
			w.Status = identity.StatusActive
			return w, nil
		},
	}
	var buf bytes.Buffer
	if err := runIdentityReEnroll(&buf, svc, "alice@Org1MSP"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotWallet.MSPDir != "/tmp/msp" {
		t.Fatalf("expected the local wallet entry to be forwarded, got %+v", gotWallet)
	}
	if !strings.Contains(buf.String(), "active") {
		t.Fatalf("unexpected output: %q", buf.String())
	}
}

/// @brief  runIdentityReEnroll rejette un handle sans wallet local (pas encore enrôlé)
/// @input  ListLocalWallets ne contient pas le handle demandé
/// @expect Erreur retournée, ReEnroll jamais appelé
func TestRunIdentityReEnroll_NoLocalWallet_Rejected(t *testing.T) {
	called := false
	svc := &mockIdentitySvc{
		listLocalWallets: func() ([]identity.WalletEntry, error) { return nil, nil },
		reEnroll: func(context.Context, identity.WalletEntry) (identity.WalletEntry, error) {
			called = true
			return identity.WalletEntry{}, nil
		},
	}
	var buf bytes.Buffer
	if err := runIdentityReEnroll(&buf, svc, "bob@Org1MSP"); err == nil {
		t.Fatalf("expected error when no local wallet matches the handle")
	}
	if called {
		t.Fatalf("ReEnroll should not be called")
	}
}

/// @brief  runIdentityRequest, réseau sans auto-enregistrement : la demande reste en attente
/// @input  nsvc=nil (pas de réseau actif)
/// @expect SubmitRequest est appelé, AutoRegister jamais, sortie mentionne "attente"/statut pending
func TestRunIdentityRequest_NoAutoRegister_Pending(t *testing.T) {
	autoCalled := false
	svc := &mockIdentitySvc{
		submitRequest: func(req identity.AccountRequest) (*identity.AccountRequest, error) {
			req.ID = "req-1"
			req.Status = identity.RequestPending
			return &req, nil
		},
		autoRegister: func(context.Context, identity.AccountRequest, string) (string, error) {
			autoCalled = true
			return "", nil
		},
	}
	var buf bytes.Buffer
	req := identity.AccountRequest{Pseudo: "bob", Email: "bob@example.com", OrgID: "Org1MSP"}
	if err := runIdentityRequest(&buf, svc, nil, req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if autoCalled {
		t.Fatalf("AutoRegister should not be called when no network service is available")
	}
	if !strings.Contains(buf.String(), "req-1") {
		t.Fatalf("unexpected output: %q", buf.String())
	}
}

/// @brief  runIdentityRequest, réseau avec AllowAutoRegister=true : la demande est auto-approuvée
/// @input  nsvc.GetActive() retourne un profil avec AllowAutoRegister=true
/// @expect AutoRegister est appelé et le secret d'enrôlement apparaît dans la sortie
func TestRunIdentityRequest_AutoRegister_Approved(t *testing.T) {
	svc := &mockIdentitySvc{
		submitRequest: func(req identity.AccountRequest) (*identity.AccountRequest, error) {
			req.ID = "req-2"
			req.Status = identity.RequestPending
			return &req, nil
		},
		autoRegister: func(_ context.Context, req identity.AccountRequest, role string) (string, error) {
			return "generated-secret", nil
		},
	}
	nsvc := &mockNetworkSvc{getActive: func() (*network.NetworkProfile, error) {
		return &network.NetworkProfile{AllowAutoRegister: true, AutoRegisterRole: "reader"}, nil
	}}
	var buf bytes.Buffer
	req := identity.AccountRequest{Pseudo: "carol", Email: "carol@example.com", OrgID: "Org1MSP"}
	if err := runIdentityRequest(&buf, svc, nsvc, req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "generated-secret") {
		t.Fatalf("expected the enrollment secret in output, got: %q", buf.String())
	}
}

/// @brief  runIdentityRequests liste les demandes d'accès en attente
/// @input  service retournant 1 demande
/// @expect Sortie contient le pseudo de la demande
func TestRunIdentityRequests_NominalCase(t *testing.T) {
	svc := &mockIdentitySvc{listRequests: func() ([]*identity.AccountRequest, error) {
		return []*identity.AccountRequest{{ID: "req-1", Pseudo: "dave", Status: identity.RequestPending}}, nil
	}}
	var buf bytes.Buffer
	if err := runIdentityRequests(&buf, svc); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "dave") {
		t.Fatalf("unexpected output: %q", buf.String())
	}
}

/// @brief  runIdentitySetRole change le rôle et rappelle la nécessité d'un ré-enrôlement
/// @input  id="alice@org1", role="contributor"
/// @expect SetRole reçoit les deux valeurs, sortie mentionne le ré-enrôlement
func TestRunIdentitySetRole_NominalCase(t *testing.T) {
	var gotID, gotRole string
	svc := &mockIdentitySvc{setRole: func(_ context.Context, name, newRole string) error {
		gotID, gotRole = name, newRole
		return nil
	}}
	var buf bytes.Buffer
	if err := runIdentitySetRole(&buf, svc, "alice@org1", "contributor"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotID != "alice@org1" || gotRole != "contributor" {
		t.Fatalf("unexpected forwarded args: id=%q role=%q", gotID, gotRole)
	}
	if !strings.Contains(buf.String(), "ré-enrôlement") {
		t.Fatalf("unexpected output: %q", buf.String())
	}
}
