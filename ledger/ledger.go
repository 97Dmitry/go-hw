// Package ledger exposes the application contract and composition factories.
package ledger

import (
	"github.com/97Dmitry/go-hw/ledger/internal/domain"
	"github.com/97Dmitry/go-hw/ledger/internal/service"
)

type Transaction = domain.Transaction
type Budget = domain.Budget
type Period = domain.Period
type CategoryTotal = domain.CategoryTotal
type Service = service.Service

var ErrBudgetExceeded = domain.ErrBudgetExceeded
var ErrBudgetNotFound = domain.ErrBudgetNotFound
var MonthPeriod = domain.MonthPeriod
