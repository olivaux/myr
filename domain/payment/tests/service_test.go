// domain/payment/tests/service_test.go — tests unitaires du service paiement
package payment_test

import (
	"errors"
	"testing"

	"myr/domain/payment"
)

// ── Mock Gateway ──────────────────────────────────────────────────────────────

type mockGateway struct {
	transferErr error
	history     []*payment.Payment
	historyErr  error
}

func (m *mockGateway) Transfer(p *payment.Payment) error {
	if m.transferErr != nil {
		return m.transferErr
	}
	m.history = append(m.history, p)
	return nil
}

func (m *mockGateway) GetHistory(identityID string) ([]*payment.Payment, error) {
	if m.historyErr != nil {
		return nil, m.historyErr
	}
	return m.history, nil
}

// ── Pay ───────────────────────────────────────────────────────────────────────

// / @brief  Paiement réussi entre deux identités
// / @input  mockGateway sans erreur, paiement de "alice" vers "bob" pour "model-1" à 9.99
// / @expect retourne un paiement avec ID généré, from/to, modelID et montant corrects, sans erreur
func TestPay_Success(t *testing.T) {
	gw := &mockGateway{}
	svc := payment.NewService(gw)

	p, err := svc.Pay("alice", "bob", "model-1", 9.99)
	if err != nil {
		t.Fatalf("Pay: %v", err)
	}
	if p.ID == "" {
		t.Error("ID should be generated")
	}
	if p.From != "alice" || p.To != "bob" {
		t.Errorf("from/to: got %q/%q, want alice/bob", p.From, p.To)
	}
	if p.ModelID != "model-1" {
		t.Errorf("ModelID: got %q, want model-1", p.ModelID)
	}
	if p.Amount != 9.99 {
		t.Errorf("Amount: got %v, want 9.99", p.Amount)
	}
	if p.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
}

// / @brief  Paiement échoue si la passerelle retourne une erreur
// / @input  mockGateway avec transferErr "insufficient funds"
// / @expect retourne une erreur propagée depuis le gateway
func TestPay_GatewayError(t *testing.T) {
	gw := &mockGateway{transferErr: errors.New("insufficient funds")}
	svc := payment.NewService(gw)

	_, err := svc.Pay("alice", "bob", "model-1", 9.99)
	if err == nil {
		t.Error("want error when gateway.Transfer fails")
	}
}

// / @brief  Chaque paiement reçoit un identifiant unique
// / @input  mockGateway sans erreur, deux appels Pay successifs
// / @expect les deux paiements ont des IDs différents
func TestPay_UniqueIDs(t *testing.T) {
	svc := payment.NewService(&mockGateway{})
	p1, _ := svc.Pay("a", "b", "m1", 1.0)
	p2, _ := svc.Pay("a", "b", "m2", 2.0)
	if p1.ID == p2.ID {
		t.Error("each payment should get a unique ID")
	}
}

// ── GetHistory ────────────────────────────────────────────────────────────────

// / @brief  Historique vide quand aucun paiement n'a été effectué
// / @input  mockGateway sans historique
// / @expect retourne une liste vide sans erreur
func TestGetHistory_Empty(t *testing.T) {
	svc := payment.NewService(&mockGateway{})
	hist, err := svc.GetHistory("alice")
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(hist) != 0 {
		t.Errorf("got %d items, want 0", len(hist))
	}
}

// / @brief  Historique contient les paiements effectués
// / @input  mockGateway sans erreur, deux paiements de "alice" effectués
// / @expect retourne une liste de 2 paiements sans erreur
func TestGetHistory_AfterPay(t *testing.T) {
	gw := &mockGateway{}
	svc := payment.NewService(gw)

	svc.Pay("alice", "bob", "m1", 5.0)
	svc.Pay("alice", "carol", "m2", 3.0)

	hist, err := svc.GetHistory("alice")
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(hist) != 2 {
		t.Errorf("got %d items, want 2", len(hist))
	}
}

// / @brief  GetHistory propage l'erreur retournée par la passerelle
// / @input  mockGateway avec historyErr "network error"
// / @expect retourne une erreur propagée depuis le gateway
func TestGetHistory_GatewayError(t *testing.T) {
	gw := &mockGateway{historyErr: errors.New("network error")}
	svc := payment.NewService(gw)
	_, err := svc.GetHistory("alice")
	if err == nil {
		t.Error("want error when gateway.GetHistory fails")
	}
}
