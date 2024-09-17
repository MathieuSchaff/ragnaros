package main

import (
	"fmt"
	"os"
)

const usage = `Usage:
  ragnaros watch                     sample the machine every second, save incidents
  ragnaros report <incident.jsonl>   summarize a saved incident`

func main() {
	if len(os.Args) < 2 {
		exitWithUsage()
	}

	var err error
	switch os.Args[1] {
	case "watch":
		err = runWatch()
	case "report":
		if len(os.Args) != 3 {
			exitWithUsage()
		}
		err = runReport(os.Args[2])
	default:
		exitWithUsage()
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "ragnaros:", err)
		os.Exit(1)
	}
}

func exitWithUsage() {
	fmt.Fprintln(os.Stderr, usage)
	os.Exit(2)
}
