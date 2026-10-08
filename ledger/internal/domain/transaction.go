package domain

import (
	"errors"
	"time"
)

var ErrInvalidAmount = errors.New("amount must not be negative")
var ErrInvalidCategory = errors.New("category must not be empty")

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

	if t.Category == "" {
		return ErrInvalidCategory
	}

	return nil
}
