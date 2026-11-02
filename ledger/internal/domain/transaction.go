package domain

import (
	"errors"
	"math"
	"time"
)

var ErrInvalidAmount = errors.New("amount must be positive, have at most two decimal places and fit NUMERIC(14,2)")
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
	if !validMoney(t.Amount) {
		return ErrInvalidAmount
	}

	if t.Category == "" {
		return ErrInvalidCategory
	}

	return nil
}

// validMoney rejects values PostgreSQL would round or fail to store.
func validMoney(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0.01 && value <= 999999999999.99 && value == math.Round(value*100)/100
}
