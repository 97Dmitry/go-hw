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
type ImportSummary = service.ImportSummary
type ImportError = service.ImportError
type Service = service.Service

const DefaultImportWorkers = service.DefaultImportWorkers

var ErrBudgetExceeded = domain.ErrBudgetExceeded
var ErrBudgetNotFound = domain.ErrBudgetNotFound
var ErrInvalidImportWorkers = service.ErrInvalidImportWorkers
var MonthPeriod = domain.MonthPeriod
