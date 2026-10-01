package domain

import (
	"errors"
	"fmt"
	"time"
)

var (
	ErrInvalidPeriod  = errors.New("budget period must be a calendar month")
	ErrBudgetNotFound = errors.New("budget not found")
	ErrBudgetExists   = errors.New("budget for this category and month already exists")
	ErrBudgetExceeded = errors.New("budget exceeded")
)

// Period — полуоткрытый интервал [Start, End).
type Period struct {
	Start time.Time
	End   time.Time
}

// MonthPeriod возвращает календарный месяц (в UTC), в который попадает t.
func MonthPeriod(t time.Time) Period {
	t = t.UTC()
	start := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)

	return Period{Start: start, End: start.AddDate(0, 1, 0)}
}

func (p Period) InPeriod(date time.Time) bool {
	return !date.Before(p.Start) && date.Before(p.End)
}

type Budget struct {
	ID       int64
	Category string
	Limit    float64
	Period   Period
}

func (b *Budget) Validate() error {
	if b.Category == "" {
		return fmt.Errorf("invalid category")
	}

	if b.Limit < 0 {
		return fmt.Errorf("invalid limit")
	}

	month := MonthPeriod(b.Period.Start)
	if !b.Period.Start.Equal(month.Start) || !b.Period.End.Equal(month.End) {
		return ErrInvalidPeriod
	}

	return nil
}
