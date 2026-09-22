package inmemory_repository

import (
	"fmt"
	"ledger/internal/core/domain"
)

var db []domain.Transaction

func AddTransaction(tx domain.Transaction) error {
	if tx.Amount < 0 {
		return fmt.Errorf("invalid amount")
	}

	db = append(db, tx)

	return nil
}

func ListTransactions() []domain.Transaction {
	return db
}
