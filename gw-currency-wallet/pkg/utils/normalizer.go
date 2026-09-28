package utils

import (
	"fmt"
	"strings"
)

var SupportedCurrency = map[string]struct{}{
	"USD": {},
	"EUR": {},
	"RUB": {},
}

func NormalizeCurrency(currency string) (string, error) {
	currency = strings.ToUpper(strings.TrimSpace(currency)) // приводим к верхнему регистру и убераем лишние пробелы
	if _, ok := SupportedCurrency[currency]; ok != true {
		return "", fmt.Errorf("unsupported currency")
	}
	return currency, nil
}

// округление до 2-х знаков после запятой
func Round2(value float64) float64 {
	return float64(int64(value*100+0.5)) / 100
}
