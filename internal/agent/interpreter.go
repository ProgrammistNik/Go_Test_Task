package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ProgrammistNik/Go_Test_Task/internal/domain"
	"github.com/shopspring/decimal"
)

type Action struct {
	Type     string          `json:"action"`
	Amount   decimal.Decimal `json:"amount"`
	Category domain.Category `json:"category"`
	SpentAt  string          `json:"spent_at"`
	DateFrom string          `json:"date_from"`
	DateTo   string          `json:"date_to"`
	Comment  string          `json:"comment"`
	Target   string          `json:"target"`
}

type LLM interface {
	Complete(context.Context, string, string) (string, error)
}

type Interpreter struct {
	llm LLM
	now func() time.Time
}

var amountPattern = regexp.MustCompile(`(?:^|\s)(\d+(?:[.,]\d{1,2})?)(?:\s|$)`)

func NewInterpreter(llm LLM) *Interpreter {
	return &Interpreter{llm: llm, now: time.Now}
}

func (i *Interpreter) Parse(ctx context.Context, message string) (Action, error) {
	if i.llm == nil {
		return i.parseFallback(message)
	}
	today := i.now().Format(time.DateOnly)
	system := fmt.Sprintf(`Ты преобразуешь сообщения о семейных расходах в JSON. Сегодня %s. Допустимые action: create, list, update_last, delete_latest. Допустимые category: food, transport, housing, entertainment, health, shopping, other. Для create заполни amount, category, spent_at, comment. Для list заполни date_from, date_to и category, если она указана. date_to включительно. Для update_last заполни amount и target. Для delete_latest заполни target. Верни только JSON.`, today)
	content, err := i.llm.Complete(ctx, system, message)
	if err != nil {
		return Action{}, err
	}
	var action Action
	if err := json.Unmarshal([]byte(content), &action); err != nil {
		return Action{}, fmt.Errorf("decode llm response: %w", err)
	}
	return normalizeAction(action, i.now()), nil
}

func (i *Interpreter) parseFallback(message string) (Action, error) {
	lower := strings.ToLower(strings.TrimSpace(message))
	now := i.now()
	if strings.HasPrefix(lower, "удали") {
		return Action{Type: "delete_latest", Target: strings.TrimSpace(strings.TrimPrefix(lower, "удали"))}, nil
	}
	if strings.Contains(lower, "последн") && (strings.Contains(lower, "не ") || strings.Contains(lower, "было")) {
		amount, err := extractLastAmount(lower)
		return Action{Type: "update_last", Amount: amount}, err
	}
	if strings.Contains(lower, "покажи") || strings.Contains(lower, "сколько") || strings.Contains(lower, "траты") {
		from, to := periodFromText(lower, now)
		return Action{Type: "list", Category: categoryFromText(lower), DateFrom: from.Format(time.DateOnly), DateTo: to.Format(time.DateOnly)}, nil
	}
	amount, err := extractLastAmount(lower)
	if err != nil {
		return Action{}, errors.New("не получилось определить сумму")
	}
	spentAt := now
	if strings.Contains(lower, "вчера") {
		spentAt = now.AddDate(0, 0, -1)
	}
	comment := amountPattern.ReplaceAllString(lower, " ")
	comment = strings.TrimSpace(strings.ReplaceAll(comment, "вчера", ""))
	category := categoryFromText(comment)
	if !category.Valid() {
		category = domain.CategoryOther
	}
	return Action{Type: "create", Amount: amount, Category: category, SpentAt: spentAt.Format(time.DateOnly), Comment: comment}, nil
}

func normalizeAction(action Action, now time.Time) Action {
	if action.Type == "create" && !action.Category.Valid() {
		action.Category = domain.CategoryOther
	}
	if action.Type == "create" && action.SpentAt == "" {
		action.SpentAt = now.Format(time.DateOnly)
	}
	if action.Type == "list" && action.DateFrom == "" {
		from, to := periodFromText("", now)
		action.DateFrom = from.Format(time.DateOnly)
		action.DateTo = to.Format(time.DateOnly)
	}
	return action
}

func extractLastAmount(text string) (decimal.Decimal, error) {
	matches := amountPattern.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return decimal.Zero, errors.New("amount not found")
	}
	value := strings.ReplaceAll(matches[len(matches)-1][1], ",", ".")
	if _, err := strconv.ParseFloat(value, 64); err != nil {
		return decimal.Zero, err
	}
	return decimal.NewFromString(value)
}

func categoryFromText(text string) domain.Category {
	categories := []struct {
		category domain.Category
		words    []string
	}{
		{domain.CategoryFood, []string{"еда", "еду", "кофе", "кафе", "продукт", "ресторан"}},
		{domain.CategoryTransport, []string{"такси", "метро", "транспорт", "бензин"}},
		{domain.CategoryHousing, []string{"жиль", "аренд", "квартир", "коммун"}},
		{domain.CategoryEntertainment, []string{"развлеч", "кино", "игр", "концерт"}},
		{domain.CategoryHealth, []string{"здоров", "врач", "аптек", "лекар"}},
		{domain.CategoryShopping, []string{"покуп", "одежд", "магазин"}},
	}
	for _, item := range categories {
		for _, word := range item.words {
			if strings.Contains(text, word) {
				return item.category
			}
		}
	}
	return ""
}

func periodFromText(text string, now time.Time) (time.Time, time.Time) {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	switch {
	case strings.Contains(text, "недел"):
		weekday := (int(today.Weekday()) + 6) % 7
		return today.AddDate(0, 0, -weekday), today
	case strings.Contains(text, "месяц"):
		return time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, today.Location()), today
	default:
		return today.AddDate(0, 0, -29), today
	}
}
