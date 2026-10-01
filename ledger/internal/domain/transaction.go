package domain

import (
	"errors"
	"time"
)

var ErrInvalidAmount = errors.New("amount must not be negative")

// Transaction — финансовая операция.
type Transaction struct {
	ID          int64
	Amount      float64
	Category    string
	Description string
	Date        time.Time
}

func (t Transaction) Validate() error {
	if t.Amount < 0 {
		return ErrInvalidAmount
	}

	return nil
}
