package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ProgrammistNik/Go_Test_Task/internal/domain"
	"github.com/shopspring/decimal"
)

type ExpenseService interface {
	Create(context.Context, domain.CreateExpense) (domain.Expense, error)
	List(context.Context, domain.ExpenseFilter) ([]domain.Expense, error)
	FindLatest(context.Context, int64, string) (domain.Expense, error)
	Update(context.Context, int64, *int64, domain.UpdateExpense) (domain.Expense, error)
	Delete(context.Context, int64, *int64) error
}

type Agent struct {
	interpreter *Interpreter
	service     ExpenseService
}

func New(interpreter *Interpreter, service ExpenseService) *Agent {
	return &Agent{interpreter: interpreter, service: service}
}

func (agent *Agent) Handle(ctx context.Context, userID int64, message string) string {
	action, err := agent.interpreter.Parse(ctx, message)
	if err != nil {
		return "Не получилось разобрать сообщение. Например: «кофе 350» или «покажи траты за неделю»."
	}
	switch action.Type {
	case "create":
		return agent.create(ctx, userID, action)
	case "list":
		return agent.list(ctx, userID, action)
	case "update_last":
		return agent.updateLatest(ctx, userID, action)
	case "delete_latest":
		return agent.deleteLatest(ctx, userID, action)
	default:
		return "Не понял, что нужно сделать. Можно добавить, показать, изменить или удалить трату."
	}
}

func (agent *Agent) create(ctx context.Context, userID int64, action Action) string {
	spentAt, err := time.Parse(time.DateOnly, action.SpentAt)
	if err != nil {
		return "Не получилось определить дату траты."
	}
	expense, err := agent.service.Create(ctx, domain.CreateExpense{
		UserID: userID, Amount: action.Amount, Category: action.Category, SpentAt: spentAt, Comment: action.Comment,
	})
	if err != nil {
		return "Не получилось сохранить трату."
	}
	return fmt.Sprintf("Записал: %s ₽, %s, %s.", expense.Amount.StringFixed(2), categoryName(expense.Category), expense.SpentAt.Format("02.01.2006"))
}

func (agent *Agent) list(ctx context.Context, userID int64, action Action) string {
	filter := domain.ExpenseFilter{UserID: &userID, Limit: 100}
	if action.Category.Valid() {
		filter.Category = &action.Category
	}
	if action.DateFrom != "" {
		from, err := time.Parse(time.DateOnly, action.DateFrom)
		if err == nil {
			filter.From = &from
		}
	}
	if action.DateTo != "" {
		to, err := time.Parse(time.DateOnly, action.DateTo)
		if err == nil {
			to = to.AddDate(0, 0, 1)
			filter.To = &to
		}
	}
	expenses, err := agent.service.List(ctx, filter)
	if err != nil {
		return "Не получилось получить список трат."
	}
	if len(expenses) == 0 {
		return "За этот период трат не найдено."
	}
	total := decimal.Zero
	lines := make([]string, 0, min(len(expenses), 10)+1)
	for index, expense := range expenses {
		total = total.Add(expense.Amount)
		if index < 10 {
			lines = append(lines, fmt.Sprintf("%s — %s ₽, %s", expense.SpentAt.Format("02.01"), expense.Amount.StringFixed(2), expense.Comment))
		}
	}
	lines = append(lines, fmt.Sprintf("Итого: %s ₽", total.StringFixed(2)))
	return strings.Join(lines, "\n")
}

func (agent *Agent) updateLatest(ctx context.Context, userID int64, action Action) string {
	expense, err := agent.service.FindLatest(ctx, userID, action.Target)
	if err != nil {
		return latestError(err)
	}
	updated, err := agent.service.Update(ctx, expense.ID, &userID, domain.UpdateExpense{Amount: &action.Amount})
	if err != nil {
		return "Не получилось изменить трату."
	}
	return fmt.Sprintf("Изменил последнюю трату: теперь %s ₽.", updated.Amount.StringFixed(2))
}

func (agent *Agent) deleteLatest(ctx context.Context, userID int64, action Action) string {
	expense, err := agent.service.FindLatest(ctx, userID, action.Target)
	if err != nil {
		return latestError(err)
	}
	if err := agent.service.Delete(ctx, expense.ID, &userID); err != nil {
		return "Не получилось удалить трату."
	}
	return fmt.Sprintf("Удалил: %s, %s ₽.", expense.Comment, expense.Amount.StringFixed(2))
}

func latestError(err error) string {
	if errors.Is(err, domain.ErrExpenseNotFound) {
		return "Подходящая трата не найдена."
	}
	return "Не получилось найти трату."
}

func categoryName(category domain.Category) string {
	names := map[domain.Category]string{
		domain.CategoryFood:          "еда",
		domain.CategoryTransport:     "транспорт",
		domain.CategoryHousing:       "жильё",
		domain.CategoryEntertainment: "развлечения",
		domain.CategoryHealth:        "здоровье",
		domain.CategoryShopping:      "покупки",
		domain.CategoryOther:         "другое",
	}
	return names[category]
}
