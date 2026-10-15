package agent

import (
	"testing"

	"github.com/shopspring/decimal"
)

func decimalFromString(t *testing.T, value string) decimal.Decimal {
	t.Helper()
	result, err := decimal.NewFromString(value)
	if err != nil {
		t.Fatal(err)
	}
	return result
}
