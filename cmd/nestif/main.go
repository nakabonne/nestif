package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"strings"

	"github.com/nakabonne/nestif"
)

var (
	flagSet = flag.NewFlagSet("nestif", flag.ContinueOnError)

	verbose     = flagSet.Bool("v", false, "Verbose output")
	outJSON     = flagSet.Bool("json", false, "Emit json format")
	minDepth    = flagSet.Int("min-depth", 1, "Lower limit of nesting depth you want to check")
	ignoreIfErr = flagSet.Bool("ignore-err", false, `Ignore to check "if err != nil"`)

	usage = func() {
		fmt.Fprintln(os.Stderr, "usage: nestif [<flag> ...] <Go file or directory> ...")
		flagSet.PrintDefaults()
	}
)

func main() {
	flagSet.Usage = usage
	if err := flagSet.Parse(os.Args[1:]); err != nil {
		if err != flag.ErrHelp {
			fmt.Fprintln(os.Stderr, err)
		}
		return
	}

	checker := &nestif.Checker{
		MinDepth:    *minDepth,
		IgnoreIfErr: *ignoreIfErr,
	}
	if *verbose {
		checker.DebugMode()
	}
	// TODO: Support the "..." glob operator and be sure to run as a "..." when no args given.
	var issues []nestif.Issue
	for _, path := range flagSet.Args() {
		if isDir(path) {
			// TODO: Support directories as args.
		} else {
			is, err := analyze(checker, path)
			if err != nil {
				if *verbose {
					fmt.Println(err)
				}
				continue
			}
			issues = append(issues, is...)
		}
	}

	if *outJSON {
		js, err := json.Marshal(issues)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(string(js))
		return
	}
	for _, i := range issues {
		fmt.Printf("%s:%d:%d: %s\n", i.Pos.Filename, i.Pos.Line, i.Pos.Column, i.Message())
	}
}

func isDir(filename string) bool {
	fi, err := os.Stat(filename)
	return err == nil && fi.IsDir()
}

func analyze(checker *nestif.Checker, filepath string) ([]nestif.Issue, error) {
	src, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filepath, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	if len(f.Comments) > 0 && isGenerated(f.Comments[0].Text()) {
		fmt.Printf("%s is a generated file", filepath)
		return nil, err
	}

	return checker.Check(f, fset), nil
}

// isGenerated checks if a given file is generated by using a comment text.
func isGenerated(text string) bool {
	return strings.Contains(text, "Code generated") || strings.Contains(text, "DO NOT EDIT")
}
