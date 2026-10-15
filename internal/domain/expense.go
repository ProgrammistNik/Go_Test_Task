package domain

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
)

type Category string

const (
	CategoryFood          Category = "food"
	CategoryTransport     Category = "transport"
	CategoryHousing       Category = "housing"
	CategoryEntertainment Category = "entertainment"
	CategoryHealth        Category = "health"
	CategoryShopping      Category = "shopping"
	CategoryOther         Category = "other"
)

var ErrExpenseNotFound = errors.New("expense not found")

type Expense struct {
	ID        int64           `json:"id"`
	UserID    int64           `json:"user_id"`
	Amount    decimal.Decimal `json:"amount"`
	Category  Category        `json:"category"`
	SpentAt   time.Time       `json:"spent_at"`
	Comment   string          `json:"comment"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type CreateExpense struct {
	UserID   int64
	Amount   decimal.Decimal
	Category Category
	SpentAt  time.Time
	Comment  string
}

type UpdateExpense struct {
	Amount   *decimal.Decimal
	Category *Category
	SpentAt  *time.Time
	Comment  *string
}

type ExpenseFilter struct {
	UserID   *int64
	Category *Category
	From     *time.Time
	To       *time.Time
	Limit    int
}

func (c Category) Valid() bool {
	switch c {
	case CategoryFood, CategoryTransport, CategoryHousing, CategoryEntertainment,
		CategoryHealth, CategoryShopping, CategoryOther:
		return true
	default:
		return false
	}
}
