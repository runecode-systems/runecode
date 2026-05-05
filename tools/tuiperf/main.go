//go:build linux

package main

import (
	"fmt"
	"os"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "tuiperf failed: %v\n", err)
		os.Exit(1)
	}
}
