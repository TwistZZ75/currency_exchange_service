package pkg

import (
	"fmt"
	"strings"
)

var SupportedCurrencies = map[string]struct{}{
	"USD": {},
	"EUR": {},
	"RUB": {},
}

// NormalizeCurrency приводит код валюты к верхнему регистру и валидирует его.
func NormalizeCurrency(c string) (string, error) {
	c = strings.ToUpper(strings.TrimSpace(c))
	if _, ok := SupportedCurrencies[c]; !ok {
		return "", fmt.Errorf("unsupported currency: %s", c)
	}
	return c, nil
}

// RoundRate округляет курс до 4 знаков
func RoundRate(v float64) float64 {
	return float64(int64(v*10000+0.5)) / 10000
}
