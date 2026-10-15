package agent

import (
	"context"
	"testing"
	"time"

	"github.com/ProgrammistNik/Go_Test_Task/internal/domain"
)

func TestInterpreterFallback(t *testing.T) {
	fixedTime := time.Date(2026, time.October, 15, 12, 0, 0, 0, time.Local)
	tests := []struct {
		name     string
		message  string
		action   string
		amount   string
		category domain.Category
		date     string
	}{
		{name: "create", message: "кофе 350", action: "create", amount: "350", category: domain.CategoryFood, date: "2026-10-15"},
		{name: "create yesterday", message: "такси 1200 вчера", action: "create", amount: "1200", category: domain.CategoryTransport, date: "2026-10-14"},
		{name: "list month", message: "сколько потратил на еду в этом месяце", action: "list", category: domain.CategoryFood},
		{name: "update", message: "последнее было не 350, а 450", action: "update_last", amount: "450"},
		{name: "delete", message: "удали такси", action: "delete_latest"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			interpreter := NewInterpreter(nil)
			interpreter.now = func() time.Time { return fixedTime }
			action, err := interpreter.Parse(context.Background(), test.message)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if action.Type != test.action {
				t.Fatalf("action = %q, want %q", action.Type, test.action)
			}
			if test.amount != "" && !action.Amount.Equal(decimalFromString(t, test.amount)) {
				t.Errorf("amount = %s, want %s", action.Amount, test.amount)
			}
			if test.category != "" && action.Category != test.category {
				t.Errorf("category = %q, want %q", action.Category, test.category)
			}
			if test.date != "" && action.SpentAt != test.date {
				t.Errorf("spent_at = %q, want %q", action.SpentAt, test.date)
			}
		})
	}
}
