package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/97Dmitry/go-hw/ledger/internal/domain"
)

type queryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

type Store struct{ DB *sql.DB }
type BudgetRepository struct {
	query queryer
	lock  bool
}
type TransactionRepository struct{ query queryer }

func (s *Store) Budgets() *BudgetRepository           { return &BudgetRepository{query: s.DB} }
func (s *Store) Transactions() *TransactionRepository { return &TransactionRepository{query: s.DB} }

func (s *Store) WithinTransaction(ctx context.Context, fn func(domain.BudgetRepository, domain.TransactionRepository) error) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := fn(&BudgetRepository{query: tx, lock: true}, &TransactionRepository{query: tx}); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *BudgetRepository) Set(ctx context.Context, b domain.Budget) (domain.Budget, error) {
	err := s.query.QueryRowContext(ctx, `INSERT INTO budgets(category,month,limit_amount) VALUES($1,$2,$3) ON CONFLICT(category,month) DO UPDATE SET limit_amount=EXCLUDED.limit_amount RETURNING id,limit_amount`, b.Category, b.Period.Start, b.Limit).Scan(&b.ID, &b.Limit)
	return b, err
}
func (s *BudgetRepository) List(ctx context.Context) ([]domain.Budget, error) {
	rows, err := s.query.QueryContext(ctx, `SELECT id,category,limit_amount,month FROM budgets ORDER BY category,month`)
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
func (s *TransactionRepository) Add(ctx context.Context, t domain.Transaction) (domain.Transaction, error) {
	err := s.query.QueryRowContext(ctx, `INSERT INTO expenses(amount,category,description,date) VALUES($1,$2,$3,$4) RETURNING id,amount,date`, t.Amount, t.Category, t.Description, t.Date.UTC()).Scan(&t.ID, &t.Amount, &t.Date)
	return t, err
}
func (s *TransactionRepository) List(ctx context.Context) ([]domain.Transaction, error) {
	rows, err := s.query.QueryContext(ctx, `SELECT id,amount,category,COALESCE(description,''),date FROM expenses ORDER BY date DESC,id DESC`)
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
func (s *TransactionRepository) Summary(ctx context.Context, from, to time.Time) ([]domain.CategoryTotal, error) {
	rows, err := s.query.QueryContext(ctx, `SELECT category,SUM(amount) FROM expenses WHERE date >= $1 AND date < $2 GROUP BY category ORDER BY category`, from, to)
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

func (s *BudgetRepository) Add(ctx context.Context, b domain.Budget) (domain.Budget, error) {
	err := s.query.QueryRowContext(ctx, `INSERT INTO budgets(category,month,limit_amount) VALUES($1,$2,$3) ON CONFLICT(category,month) DO NOTHING RETURNING id,limit_amount`, b.Category, b.Period.Start, b.Limit).Scan(&b.ID, &b.Limit)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Budget{}, domain.ErrBudgetExists
	}
	return b, err
}

func (s *BudgetRepository) GetByCategory(ctx context.Context, category string, date time.Time) (domain.Budget, error) {
	b := domain.Budget{Category: category, Period: domain.MonthPeriod(date)}
	query := `SELECT id,limit_amount FROM budgets WHERE category=$1 AND month=$2`
	if s.lock {
		query += ` FOR UPDATE`
	}
	err := s.query.QueryRowContext(ctx, query, category, b.Period.Start).Scan(&b.ID, &b.Limit)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Budget{}, domain.ErrBudgetNotFound
	}
	return b, err
}

func (s *TransactionRepository) SumByCategory(ctx context.Context, category string, period domain.Period) (int64, error) {
	var cents int64
	err := s.query.QueryRowContext(ctx, `SELECT (COALESCE(SUM(amount),0)*100)::bigint FROM expenses WHERE category=$1 AND date >= $2 AND date < $3`, category, period.Start, period.End).Scan(&cents)
	return cents, err
}

var _ domain.BudgetRepository = (*BudgetRepository)(nil)
var _ domain.TransactionRepository = (*TransactionRepository)(nil)
var _ domain.UnitOfWork = (*Store)(nil)
