package memory

import (
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/97Dmitry/go-hw/ledger/internal/domain"
)

// budgetKey — один бюджет на категорию в календарном месяце.
type budgetKey struct {
	category string
	year     int
	month    time.Month
}

func keyOf(category string, t time.Time) budgetKey {
	start := domain.MonthPeriod(t).Start

	return budgetKey{category: category, year: start.Year(), month: start.Month()}
}

// BudgetRepository — потокобезопасное in-memory хранилище бюджетов.
// ID выдаёт только репозиторий, начиная с 1: ID == 0 означает «ещё не сохранён».
type BudgetRepository struct {
	mu      sync.RWMutex
	budgets map[budgetKey]domain.Budget
	lastID  int64
}

func NewBudgetRepository() *BudgetRepository {
	return &BudgetRepository{
		budgets: make(map[budgetKey]domain.Budget),
	}
}

func (r *BudgetRepository) Add(b domain.Budget) (domain.Budget, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := keyOf(b.Category, b.Period.Start)
	if _, ok := r.budgets[key]; ok {
		return domain.Budget{}, domain.ErrBudgetExists
	}

	r.lastID++
	b.ID = r.lastID
	r.budgets[key] = b

	return b, nil
}

// Set создаёт бюджет или заменяет существующий на тот же месяц, сохраняя его ID.
func (r *BudgetRepository) Set(b domain.Budget) (domain.Budget, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := keyOf(b.Category, b.Period.Start)
	if existing, ok := r.budgets[key]; ok {
		b.ID = existing.ID
	} else {
		r.lastID++
		b.ID = r.lastID
	}
	r.budgets[key] = b

	return b, nil
}

// GetByCategory возвращает бюджет категории на месяц, в который попадает date.
func (r *BudgetRepository) GetByCategory(category string, date time.Time) (domain.Budget, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	b, ok := r.budgets[keyOf(category, date)]
	if !ok {
		return domain.Budget{}, domain.ErrBudgetNotFound
	}

	return b, nil
}

func (r *BudgetRepository) List() []domain.Budget {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return slices.Collect(maps.Values(r.budgets))
}
