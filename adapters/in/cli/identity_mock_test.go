// adapters/in/cli/identity_mock_test.go — mock partagé IdentityService (Pattern B — fn-func)
package cli

import (
	"context"

	"myr-core/domain/identity"
)

type mockIdentitySvc struct {
	listLocalWallets func() ([]identity.WalletEntry, error)
	register         func(ctx context.Context, req identity.RegisterRequest) (string, error)
	enroll           func(ctx context.Context, name, secret, orgID string) (identity.WalletEntry, error)
	getStatus        func(ctx context.Context, wallet identity.WalletEntry) (string, error)
	reEnroll         func(ctx context.Context, wallet identity.WalletEntry) (identity.WalletEntry, error)
	loadGuestWallet  func(certPath, keyPath, caCertPath, orgID string) (identity.WalletEntry, error)
	submitRequest    func(req identity.AccountRequest) (*identity.AccountRequest, error)
	autoRegister     func(ctx context.Context, req *identity.AccountRequest, role string) (string, error)
	listRequests     func() ([]*identity.AccountRequest, error)
	walletDir        func() string
	setRole          func(ctx context.Context, name, newRole string) error
}

func (m *mockIdentitySvc) ListLocalWallets() ([]identity.WalletEntry, error) {
	if m.listLocalWallets != nil {
		return m.listLocalWallets()
	}
	return nil, nil
}
func (m *mockIdentitySvc) Register(ctx context.Context, req identity.RegisterRequest) (string, error) {
	if m.register != nil {
		return m.register(ctx, req)
	}
	return "secret-test", nil
}
func (m *mockIdentitySvc) Enroll(ctx context.Context, name, secret, orgID string) (identity.WalletEntry, error) {
	if m.enroll != nil {
		return m.enroll(ctx, name, secret, orgID)
	}
	return identity.WalletEntry{Handle: name + "@" + orgID, Name: name, OrgID: orgID, Status: identity.StatusPending}, nil
}
func (m *mockIdentitySvc) GetStatus(ctx context.Context, wallet identity.WalletEntry) (string, error) {
	if m.getStatus != nil {
		return m.getStatus(ctx, wallet)
	}
	return identity.StatusActive, nil
}
func (m *mockIdentitySvc) ReEnroll(ctx context.Context, wallet identity.WalletEntry) (identity.WalletEntry, error) {
	if m.reEnroll != nil {
		return m.reEnroll(ctx, wallet)
	}
	return wallet, nil
}
func (m *mockIdentitySvc) LoadGuestWallet(certPath, keyPath, caCertPath, orgID string) (identity.WalletEntry, error) {
	if m.loadGuestWallet != nil {
		return m.loadGuestWallet(certPath, keyPath, caCertPath, orgID)
	}
	return identity.WalletEntry{}, nil
}
func (m *mockIdentitySvc) SubmitRequest(req identity.AccountRequest) (*identity.AccountRequest, error) {
	if m.submitRequest != nil {
		return m.submitRequest(req)
	}
	req.ID = "req-test"
	req.Status = identity.RequestPending
	return &req, nil
}
func (m *mockIdentitySvc) AutoRegister(ctx context.Context, req *identity.AccountRequest, role string) (string, error) {
	if m.autoRegister != nil {
		return m.autoRegister(ctx, req, role)
	}
	return "secret-test", nil
}
func (m *mockIdentitySvc) ListRequests() ([]*identity.AccountRequest, error) {
	if m.listRequests != nil {
		return m.listRequests()
	}
	return nil, nil
}
func (m *mockIdentitySvc) WalletDir() string {
	if m.walletDir != nil {
		return m.walletDir()
	}
	return ""
}
func (m *mockIdentitySvc) SetRole(ctx context.Context, name, newRole string) error {
	if m.setRole != nil {
		return m.setRole(ctx, name, newRole)
	}
	return nil
}
