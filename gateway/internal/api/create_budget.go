package api

import (
	"github.com/97Dmitry/go-hw/ledger"
	"time"
)

type CreateBudgetRequest struct {
	Category string  `json:"category"`
	Limit    float64 `json:"limit"`
}

func (r CreateBudgetRequest) ToDomain(now time.Time) ledger.Budget {
	return ledger.Budget{Category: r.Category, Limit: r.Limit, Period: ledger.MonthPeriod(now)}
}

type BudgetResponse struct {
	ID       int64         `json:"id"`
	Category string        `json:"category"`
	Limit    float64       `json:"limit"`
	Period   ledger.Period `json:"period"`
}

func BudgetFromDomain(b ledger.Budget) BudgetResponse {
	return BudgetResponse{ID: b.ID, Category: b.Category, Limit: b.Limit, Period: b.Period}
}
