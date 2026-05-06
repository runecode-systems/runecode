//go:build linux

package main

import (
	"errors"
	"fmt"
	"os"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		var usageErr usageError
		if errors.As(err, &usageErr) {
			fmt.Fprintf(os.Stderr, "tuiperf usage error: %v\n", err)
			os.Exit(2)
		}
		fmt.Fprintf(os.Stderr, "tuiperf failed: %v\n", err)
		os.Exit(1)
	}
}
