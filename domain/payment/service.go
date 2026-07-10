// domain/payment/service.go
package payment

import (
	"fmt"
	"time"
)

type Service struct {
	gateway PaymentGatewayPort
}

func NewService(gateway PaymentGatewayPort) *Service {
	return &Service{gateway: gateway}
}

func (s *Service) Pay(from, to, modelID string, amount float64) (*Payment, error) {
	p := &Payment{
		ID:        generateID(),
		From:      from,
		To:        to,
		ModelID:   modelID,
		Amount:    amount,
		CreatedAt: time.Now(),
	}
	if err := s.gateway.Transfer(p); err != nil {
		return nil, fmt.Errorf("payment transfer: %w", err)
	}
	return p, nil
}

func (s *Service) GetHistory(identityID string) ([]*Payment, error) {
	return s.gateway.GetHistory(identityID)
}
