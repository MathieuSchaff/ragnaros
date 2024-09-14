package main

import (
	"fmt"
	"os"
)

func main() {
	if err := runWatch(); err != nil {
		fmt.Fprintln(os.Stderr, "ragnaros:", err)
		os.Exit(1)
	}
}
