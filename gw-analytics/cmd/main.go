package main

import (
	"fmt"
	"os"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "gw-analytics: %v\n", err)
		os.Exit(1)
	}
}
