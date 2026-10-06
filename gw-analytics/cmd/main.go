package main

import (
	"fmt"
	"os"
)

// @title           gw-analytics API
// @version         1.0
// @description     Аналитика крупных денежных переводов: агрегации по периодам.
// @host            localhost:8082
// @BasePath        /
func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "gw-analytics: %v\n", err)
		os.Exit(1)
	}
}
