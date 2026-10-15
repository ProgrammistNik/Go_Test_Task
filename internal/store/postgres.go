package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ProgrammistNik/Go_Test_Task/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

const expenseColumns = `id, user_id, amount::text, category, spent_at, comment, created_at, updated_at`

type Postgres struct {
	pool *pgxpool.Pool
}

type rowScanner interface {
	Scan(dest ...any) error
}

func NewPostgres(ctx context.Context, databaseURL string) (*Postgres, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database config: %w", err)
	}
	cfg.MaxConns = 10
	cfg.MinConns = 2
	cfg.MaxConnLifetime = 30 * time.Minute
	cfg.MaxConnIdleTime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &Postgres{pool: pool}, nil
}

func (p *Postgres) Close() {
	p.pool.Close()
}

func (p *Postgres) Create(ctx context.Context, input domain.CreateExpense) (domain.Expense, error) {
	query := `
		INSERT INTO expenses (user_id, amount, category, spent_at, comment)
		VALUES ($1, $2::numeric, $3, $4, $5)
		RETURNING ` + expenseColumns

	return scanExpense(p.pool.QueryRow(
		ctx,
		query,
		input.UserID,
		input.Amount.StringFixed(2),
		input.Category,
		input.SpentAt,
		input.Comment,
	))
}

func (p *Postgres) Get(ctx context.Context, id int64, userID *int64) (domain.Expense, error) {
	query := `SELECT ` + expenseColumns + ` FROM expenses WHERE id = $1 AND ($2::bigint IS NULL OR user_id = $2)`
	expense, err := scanExpense(p.pool.QueryRow(ctx, query, id, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Expense{}, domain.ErrExpenseNotFound
	}
	if err != nil {
		return domain.Expense{}, fmt.Errorf("get expense: %w", err)
	}
	return expense, nil
}

func (p *Postgres) List(ctx context.Context, filter domain.ExpenseFilter) ([]domain.Expense, error) {
	query := `
		SELECT ` + expenseColumns + `
		FROM expenses
		WHERE ($1::bigint IS NULL OR user_id = $1)
		  AND ($2::text IS NULL OR category = $2)
		  AND ($3::date IS NULL OR spent_at >= $3)
		  AND ($4::date IS NULL OR spent_at <= $4)
		ORDER BY spent_at DESC, id DESC
		LIMIT $5`

	rows, err := p.pool.Query(ctx, query, filter.UserID, filter.Category, filter.From, filter.To, filter.Limit)
	if err != nil {
		return nil, fmt.Errorf("list expenses: %w", err)
	}
	defer rows.Close()

	expenses := make([]domain.Expense, 0)
	for rows.Next() {
		expense, scanErr := scanExpense(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan expense: %w", scanErr)
		}
		expenses = append(expenses, expense)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate expenses: %w", err)
	}
	return expenses, nil
}

func (p *Postgres) FindLatest(ctx context.Context, userID int64, search string) (domain.Expense, error) {
	query := `
		SELECT ` + expenseColumns + `
		FROM expenses
		WHERE user_id = $1
		  AND ($2 = '' OR comment ILIKE '%' || $2 || '%' OR category = LOWER($2))
		ORDER BY spent_at DESC, id DESC
		LIMIT 1`

	expense, err := scanExpense(p.pool.QueryRow(ctx, query, userID, search))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Expense{}, domain.ErrExpenseNotFound
	}
	if err != nil {
		return domain.Expense{}, fmt.Errorf("find latest expense: %w", err)
	}
	return expense, nil
}

func (p *Postgres) Update(
	ctx context.Context,
	id int64,
	userID *int64,
	input domain.UpdateExpense,
) (domain.Expense, error) {
	query := `
		UPDATE expenses
		SET amount = COALESCE($3::numeric, amount),
		    category = COALESCE($4::text, category),
		    spent_at = COALESCE($5::date, spent_at),
		    comment = CASE WHEN $6 THEN $7 ELSE comment END,
		    updated_at = NOW()
		WHERE id = $1 AND ($2::bigint IS NULL OR user_id = $2)
		RETURNING ` + expenseColumns

	var amount *string
	if input.Amount != nil {
		value := input.Amount.StringFixed(2)
		amount = &value
	}

	expense, err := scanExpense(p.pool.QueryRow(
		ctx,
		query,
		id,
		userID,
		amount,
		input.Category,
		input.SpentAt,
		input.Comment != nil,
		input.Comment,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Expense{}, domain.ErrExpenseNotFound
	}
	if err != nil {
		return domain.Expense{}, fmt.Errorf("update expense: %w", err)
	}
	return expense, nil
}

func (p *Postgres) Delete(ctx context.Context, id int64, userID *int64) error {
	result, err := p.pool.Exec(
		ctx,
		`DELETE FROM expenses WHERE id = $1 AND ($2::bigint IS NULL OR user_id = $2)`,
		id,
		userID,
	)
	if err != nil {
		return fmt.Errorf("delete expense: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrExpenseNotFound
	}
	return nil
}

func (p *Postgres) Ping(ctx context.Context) error {
	if err := p.pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}
	return nil
}

func scanExpense(row rowScanner) (domain.Expense, error) {
	var expense domain.Expense
	var amount string
	if err := row.Scan(
		&expense.ID,
		&expense.UserID,
		&amount,
		&expense.Category,
		&expense.SpentAt,
		&expense.Comment,
		&expense.CreatedAt,
		&expense.UpdatedAt,
	); err != nil {
		return domain.Expense{}, err
	}

	parsedAmount, err := decimal.NewFromString(amount)
	if err != nil {
		return domain.Expense{}, fmt.Errorf("parse stored amount: %w", err)
	}
	expense.Amount = parsedAmount
	return expense, nil
}
