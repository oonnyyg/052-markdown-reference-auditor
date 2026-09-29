// Command doccheck audits relative links between markdown documentation files.
//
// Usage:
//
//	doccheck -root docs -entry index.md -report report.json
//
// Exit codes: 0 no problems, 1 broken links or unreachable pages,
// 2 invalid invocation arguments.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"example.com/markdown-reference-auditor/doccheck"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	fs := flag.NewFlagSet("doccheck", flag.ContinueOnError)
	root := fs.String("root", "", "documentation root directory")
	entry := fs.String("entry", "", "entry markdown file (relative to root, or absolute)")
	report := fs.String("report", "", "JSON report output path")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: doccheck -root DIR -entry FILE.md -report report.json")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintf(os.Stderr, "doccheck: unexpected arguments: %v\n", fs.Args())
		return 2
	}

	scan, err := doccheck.ScanDocs(*root, *entry)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}

	res, err := doccheck.Check(scan)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		var in *doccheck.InputError
		if errors.As(err, &in) {
			return 2
		}
		return 2
	}

	if !res.HasProblems() {
		fmt.Println("doccheck: no broken links or unreachable pages")
		return 0
	}

	if err := doccheck.WriteReport(*report, scan, res); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}

	fmt.Printf("doccheck: problems found; report written to %s\n", *report)
	printSummary(res)
	return 1
}

func printSummary(res *doccheck.CheckResult) {
	for _, f := range res.Links {
		if f.Status == doccheck.StatusOK {
			continue
		}
		fmt.Fprintf(os.Stdout, "%s:%d:%d: %s (%s)\n", f.Source, f.Line, f.Column, f.Status, f.Target)
	}
	for _, rel := range res.Unreachable {
		fmt.Fprintf(os.Stdout, "%s: unreachable_page\n", rel)
	}
}
