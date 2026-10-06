package main

import (
	"fmt"
	"os"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "gw-exchanger: %v\n", err)
		os.Exit(1)
	}
}
