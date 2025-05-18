package utils

import "github.com/shopspring/decimal"

func Decimal(amount string) decimal.Decimal {
	return decimal.RequireFromString(amount)
}
