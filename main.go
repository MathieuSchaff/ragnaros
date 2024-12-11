package main

import (
	"fmt"
	"os"
)

const usage = `Usage:
  ragnaros watch [options]           measure every second, save a file when it lags
  ragnaros report <incident.jsonl>   sum up a saved file

Options of watch: ragnaros watch -h`

func main() {
	if len(os.Args) < 2 {
		exitWithUsage()
	}

	var err error
	switch os.Args[1] {
	case "watch":
		err = runWatch(os.Args[2:])
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
