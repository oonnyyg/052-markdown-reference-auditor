package main

import (
	"flag"
	"fmt"
	"os"

	"example.com/markdown-reference-auditor/internal/doccheck"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	fs := flag.NewFlagSet("doccheck", flag.ContinueOnError)
	root := fs.String("root", "", "documentation root directory")
	entry := fs.String("entry", "", "entry Markdown file (path relative to -root)")
	report := fs.String("report", "", "JSON report output path (must not be inside -root)")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	result, err := doccheck.Run(*root, *entry, *report)
	if err != nil {
		fmt.Fprintln(os.Stderr, "doccheck:", err)
		return 2
	}

	if result.HasProblem() {
		if err := doccheck.WriteReport(*report, result); err != nil {
			fmt.Fprintln(os.Stderr, "doccheck:", err)
			return 2
		}
		fmt.Fprintf(os.Stderr, "doccheck: %d link problem(s), %d unreachable page(s); report written to %s\n",
			result.ProblemCount(), len(result.Unreachable), *report)
		return 1
	}
	return 0
}
