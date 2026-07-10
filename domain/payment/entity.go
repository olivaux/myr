// domain/payment/entity.go
package payment

import (
	"crypto/rand"
	"fmt"
	"time"
)

type Payment struct {
	ID        string
	From      string
	To        string
	ModelID   string
	Amount    float64
	CreatedAt time.Time
}

func generateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}
