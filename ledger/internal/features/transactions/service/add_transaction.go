package service

import (
	"fmt"
	"ledger/internal/core/domain"
	inmemoryRepository "ledger/internal/features/transactions/repository/inmemory"
)

func AddTransaction(tx domain.Transaction) error {
	if err := inmemoryRepository.AddTransaction(tx); err != nil {
		return fmt.Errorf("error adding transaction: %v", err)
	}

	return nil
}

func ListTransactions() []domain.Transaction {
	return inmemoryRepository.ListTransactions()
}
