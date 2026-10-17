package api

import (
	"github.com/97Dmitry/go-hw/ledger"
	"time"
)

type CreateTransactionRequest struct {
	Amount      float64   `json:"amount"`
	Category    string    `json:"category"`
	Description string    `json:"description"`
	Date        time.Time `json:"date"`
}

func (r CreateTransactionRequest) ToDomain() ledger.Transaction {
	return ledger.Transaction{Amount: r.Amount, Category: r.Category, Description: r.Description, Date: r.Date}
}

type TransactionResponse struct {
	ID          int64     `json:"id"`
	Amount      float64   `json:"amount"`
	Category    string    `json:"category"`
	Description string    `json:"description"`
	Date        time.Time `json:"date"`
}

func TransactionFromDomain(tx ledger.Transaction) TransactionResponse {
	return TransactionResponse{ID: tx.ID, Amount: tx.Amount, Category: tx.Category, Description: tx.Description, Date: tx.Date}
}
