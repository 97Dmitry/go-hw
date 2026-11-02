package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/97Dmitry/go-hw/ledger/internal/domain"
)

type Store struct{ DB *sql.DB }

func (s *Store) SetBudget(ctx context.Context, b domain.Budget) (domain.Budget, error) {
	b.Period = domain.MonthPeriod(b.Period.Start)
	err := s.DB.QueryRowContext(ctx, `INSERT INTO budgets(category,month,limit_amount) VALUES($1,$2,$3) ON CONFLICT(category,month) DO UPDATE SET limit_amount=EXCLUDED.limit_amount RETURNING id,limit_amount`, b.Category, b.Period.Start, b.Limit).Scan(&b.ID, &b.Limit)
	return b, err
}
func (s *Store) ListBudgets(ctx context.Context) ([]domain.Budget, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id,category,limit_amount,month FROM budgets ORDER BY category,month`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]domain.Budget, 0)
	for rows.Next() {
		var b domain.Budget
		var month time.Time
		if err := rows.Scan(&b.ID, &b.Category, &b.Limit, &month); err != nil {
			return nil, err
		}
		b.Period = domain.MonthPeriod(month)
		result = append(result, b)
	}
	return result, rows.Err()
}
func (s *Store) AddTransaction(ctx context.Context, t domain.Transaction) (domain.Transaction, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return t, err
	}
	defer tx.Rollback()
	period := domain.MonthPeriod(t.Date)
	var limit float64
	err = tx.QueryRowContext(ctx, `SELECT limit_amount FROM budgets WHERE category=$1 AND month=$2 FOR UPDATE`, t.Category, period.Start).Scan(&limit)
	if errors.Is(err, sql.ErrNoRows) {
		return t, domain.ErrBudgetNotFound
	}
	if err != nil {
		return t, err
	}
	// Compare in PostgreSQL NUMERIC so rounding matches the persisted cents.
	var allowed bool
	err = tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(amount),0)+$4::numeric(14,2) <= $5::numeric(14,2) FROM expenses WHERE category=$1 AND date >= $2 AND date < $3`, t.Category, period.Start, period.End, t.Amount, limit).Scan(&allowed)
	if err != nil {
		return t, err
	}
	if !allowed {
		return t, domain.ErrBudgetExceeded
	}
	err = tx.QueryRowContext(ctx, `INSERT INTO expenses(amount,category,description,date) VALUES($1,$2,$3,$4) RETURNING id,amount,date`, t.Amount, t.Category, t.Description, t.Date.UTC()).Scan(&t.ID, &t.Amount, &t.Date)
	if err != nil {
		return t, err
	}
	return t, tx.Commit()
}
func (s *Store) ListTransactions(ctx context.Context) ([]domain.Transaction, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id,amount,category,COALESCE(description,''),date FROM expenses ORDER BY date DESC,id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]domain.Transaction, 0)
	for rows.Next() {
		var t domain.Transaction
		if err := rows.Scan(&t.ID, &t.Amount, &t.Category, &t.Description, &t.Date); err != nil {
			return nil, err
		}
		result = append(result, t)
	}
	return result, rows.Err()
}
func (s *Store) Summary(ctx context.Context, from, to time.Time) ([]domain.CategoryTotal, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT category,SUM(amount) FROM expenses WHERE date >= $1 AND date < $2 GROUP BY category ORDER BY category`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]domain.CategoryTotal, 0)
	for rows.Next() {
		var r domain.CategoryTotal
		if err := rows.Scan(&r.Category, &r.Total); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}
