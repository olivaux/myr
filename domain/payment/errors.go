package payment

import "errors"

var ErrNotFound = errors.New("payment: not found")
var ErrInsufficientFunds = errors.New("payment: insufficient funds")
