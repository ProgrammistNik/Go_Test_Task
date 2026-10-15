package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/ProgrammistNik/Go_Test_Task/internal/domain"
)

var ErrInvalidExpense = errors.New("invalid expense")

type ExpenseStore interface {
	Create(context.Context, domain.CreateExpense) (domain.Expense, error)
	Get(context.Context, int64, *int64) (domain.Expense, error)
	List(context.Context, domain.ExpenseFilter) ([]domain.Expense, error)
	FindLatest(context.Context, int64, string) (domain.Expense, error)
	Update(context.Context, int64, *int64, domain.UpdateExpense) (domain.Expense, error)
	Delete(context.Context, int64, *int64) error
}

type ExpenseService struct {
	store ExpenseStore
}

func NewExpenseService(store ExpenseStore) *ExpenseService {
	return &ExpenseService{store: store}
}

func (s *ExpenseService) Create(ctx context.Context, input domain.CreateExpense) (domain.Expense, error) {
	input.Comment = strings.TrimSpace(input.Comment)
	input.SpentAt = day(input.SpentAt)
	if input.UserID <= 0 || !input.Amount.IsPositive() || !input.Category.Valid() || input.SpentAt.IsZero() {
		return domain.Expense{}, ErrInvalidExpense
	}
	return s.store.Create(ctx, input)
}

func (s *ExpenseService) Get(ctx context.Context, id int64, userID *int64) (domain.Expense, error) {
	if id <= 0 {
		return domain.Expense{}, ErrInvalidExpense
	}
	return s.store.Get(ctx, id, userID)
}

func (s *ExpenseService) List(ctx context.Context, filter domain.ExpenseFilter) ([]domain.Expense, error) {
	if filter.Category != nil && !filter.Category.Valid() {
		return nil, ErrInvalidExpense
	}
	if filter.Limit <= 0 {
		filter.Limit = 100
	}
	if filter.Limit > 500 {
		filter.Limit = 500
	}
	if filter.From != nil {
		value := day(*filter.From)
		filter.From = &value
	}
	if filter.To != nil {
		value := day(*filter.To)
		filter.To = &value
	}
	if filter.From != nil && filter.To != nil && filter.From.After(*filter.To) {
		return nil, ErrInvalidExpense
	}
	return s.store.List(ctx, filter)
}

func (s *ExpenseService) FindLatest(
	ctx context.Context,
	userID int64,
	search string,
) (domain.Expense, error) {
	if userID <= 0 {
		return domain.Expense{}, ErrInvalidExpense
	}
	return s.store.FindLatest(ctx, userID, strings.TrimSpace(search))
}

func (s *ExpenseService) Update(
	ctx context.Context,
	id int64,
	userID *int64,
	input domain.UpdateExpense,
) (domain.Expense, error) {
	if id <= 0 {
		return domain.Expense{}, ErrInvalidExpense
	}
	if input.Amount == nil && input.Category == nil && input.SpentAt == nil && input.Comment == nil {
		return domain.Expense{}, ErrInvalidExpense
	}
	if input.Amount != nil && !input.Amount.IsPositive() {
		return domain.Expense{}, ErrInvalidExpense
	}
	if input.Category != nil && !input.Category.Valid() {
		return domain.Expense{}, ErrInvalidExpense
	}
	if input.SpentAt != nil {
		value := day(*input.SpentAt)
		input.SpentAt = &value
	}
	if input.Comment != nil {
		value := strings.TrimSpace(*input.Comment)
		input.Comment = &value
	}
	return s.store.Update(ctx, id, userID, input)
}

func (s *ExpenseService) Delete(ctx context.Context, id int64, userID *int64) error {
	if id <= 0 {
		return ErrInvalidExpense
	}
	return s.store.Delete(ctx, id, userID)
}

func day(value time.Time) time.Time {
	year, month, date := value.Date()
	return time.Date(year, month, date, 0, 0, 0, 0, time.Local)
}
