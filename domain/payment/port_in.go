// domain/payment/port_in.go — port d'entrée : interface exposée aux adapters entrants
package payment

// PaymentService est le port d'entrée du domaine payment.
type PaymentService interface {
	Pay(from, to, modelID string, amount float64) (*Payment, error)
	GetHistory(identityID string) ([]*Payment, error)
}
