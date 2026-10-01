package service

import (
	"fmt"
	"time"

	"github.com/97Dmitry/go-hw/ledger/internal/domain"
)

type BudgetRepository interface {
	Add(b domain.Budget) (domain.Budget, error)
	Set(b domain.Budget) (domain.Budget, error)
	List() []domain.Budget
	GetByCategory(category string, date time.Time) (domain.Budget, error)
}

type BudgetService struct {
	repo BudgetRepository
}

func NewBudgetService(repo BudgetRepository) *BudgetService {
	return &BudgetService{repo: repo}
}

func (s *BudgetService) Add(b domain.Budget) (domain.Budget, error) {
	if err := b.Validate(); err != nil {
		return domain.Budget{}, fmt.Errorf("validate budget: %w", err)
	}

	return s.repo.Add(b)
}

func (s *BudgetService) Set(b domain.Budget) (domain.Budget, error) {
	if err := b.Validate(); err != nil {
		return domain.Budget{}, fmt.Errorf("validate budget: %w", err)
	}

	return s.repo.Set(b)
}

func (s *BudgetService) GetByCategory(category string, date time.Time) (domain.Budget, error) {
	budget, err := s.repo.GetByCategory(category, date)

	if err != nil {
		return budget, fmt.Errorf("get budget by category: %w", err)
	}

	return budget, nil
}

func (s *BudgetService) List() []domain.Budget {
	return s.repo.List()
}
