// domain/payment/ports.go — interfaces secondaires (out-ports) du domaine payment
package payment

// PaymentGatewayPort est le contrat que tout adapter de paiement doit respecter.
type PaymentGatewayPort interface {
	Transfer(p *Payment) error
	GetHistory(identityID string) ([]*Payment, error)
}
