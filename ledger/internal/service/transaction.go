package service

import (
	"fmt"
	"time"

	"github.com/97Dmitry/go-hw/ledger/internal/domain"
)

type BudgetGetter interface {
	GetByCategory(category string, date time.Time) (domain.Budget, error)
}

// TransactionRepository объявлен на стороне потребителя: сервис не зависит от конкретного хранилища.
type TransactionRepository interface {
	Add(tx domain.Transaction) (domain.Transaction, error)
	List() []domain.Transaction
	GetAllTransactionByCategory(category string, period domain.Period) []domain.Transaction
}

type TransactionService struct {
	repo    TransactionRepository
	budgets BudgetGetter
}

func NewTransactionService(repo TransactionRepository, budgets BudgetGetter) *TransactionService {
	return &TransactionService{repo: repo, budgets: budgets}
}

func (s *TransactionService) AddTransaction(tx domain.Transaction) (domain.Transaction, error) {
	if err := tx.Validate(); err != nil {
		return domain.Transaction{}, fmt.Errorf("validate transaction: %w", err)
	}

	budget, err := s.budgets.GetByCategory(tx.Category, tx.Date)
	if err != nil {
		return domain.Transaction{}, fmt.Errorf("get budget for category %q: %w", tx.Category, err)
	}

	var spent float64
	for _, t := range s.repo.GetAllTransactionByCategory(tx.Category, budget.Period) {
		spent += t.Amount
	}

	if spent+tx.Amount > budget.Limit {
		return domain.Transaction{}, fmt.Errorf("category %q: spent %.2f + %.2f > limit %.2f: %w",
			tx.Category, spent, tx.Amount, budget.Limit, domain.ErrBudgetExceeded)
	}

	saved, err := s.repo.Add(tx)
	if err != nil {
		return domain.Transaction{}, fmt.Errorf("add transaction: %w", err)
	}

	return saved, nil
}

func (s *TransactionService) ListTransactions() []domain.Transaction {
	return s.repo.List()
}
