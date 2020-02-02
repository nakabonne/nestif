// Copyright 2020 Ryo Nakao <nakabonne@gmail.com>.
//
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"go/build"
	"go/parser"
	"go/token"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/nakabonne/nestif"
	flag "github.com/spf13/pflag"
)

var (
	flagSet = flag.NewFlagSet("nestif", flag.ContinueOnError)

	usage = func() {
		fmt.Fprintln(os.Stderr, "usage: nestif [<flag> ...] <Go files or directories or packages> ...")
		flagSet.PrintDefaults()
	}

	errformat = func(file string, line, col int, msg string) string {
		return fmt.Sprintf("%s:%d:%d: %s", file, line, col, msg)
	}
)

type app struct {
	verbose         bool
	outJSON         bool
	minComplexity   int
	top             int
	excludeDirs     []string
	excludePatterns []*regexp.Regexp
	stdout          io.Writer
	stderr          io.Writer
}

func main() {
	a := &app{
		stdout: os.Stdout,
		stderr: os.Stderr,
	}
	flagSet.BoolVarP(&a.verbose, "verbose", "v", false, "verbose output")
	flagSet.BoolVar(&a.outJSON, "json", false, "emit json format")
	flagSet.IntVar(&a.minComplexity, "min", 1, "minimum complexity to show")
	flagSet.IntVar(&a.top, "top", 10, "show only the top N most complex if statements")
	flagSet.StringSliceVarP(&a.excludeDirs, "exclude-dirs", "e", []string{}, "regexps of directories to be excluded for checking; comma-separated list")
	flagSet.Usage = usage
	if err := flagSet.Parse(os.Args[1:]); err != nil {
		if err != flag.ErrHelp {
			fmt.Fprintln(a.stderr, err)
		}
		return
	}

	os.Exit(a.run(flagSet.Args()))
}

func (a *app) run(args []string) int {
	issues, err := a.check(args)
	if err != nil {
		fmt.Fprintln(a.stderr, err)
		return 1
	}
	sort.Slice(issues, func(i, j int) bool {
		return issues[i].Complexity > issues[j].Complexity
	})

	a.write(issues)
	return 0
}

func (a *app) check(args []string) ([]nestif.Issue, error) {
	a.excludePatterns = make([]*regexp.Regexp, 0, len(a.excludeDirs))
	for _, d := range a.excludeDirs {
		p, err := regexp.Compile(d)
		if err != nil {
			return nil, fmt.Errorf("failed to parse exclude dir pattern: %v", err)
		}
		a.excludePatterns = append(a.excludePatterns, p)
	}

	checker := &nestif.Checker{
		MinComplexity: a.minComplexity,
	}
	if a.verbose {
		checker.DebugMode(a.stderr)
	}

	// TODO: Reduce allocation.
	var files, dirs, pkgs []string
	// Check all files recursively when no args given.
	if len(args) == 0 {
		dirs = append(dirs, allPackagesInFS("./...", a.stderr)...)
	}
	for _, arg := range args {
		if strings.HasSuffix(arg, "/...") && isDir(arg[:len(arg)-len("/...")]) {
			dirs = append(dirs, allPackagesInFS(arg, a.stderr)...)
		} else if isDir(arg) {
			dirs = append(dirs, arg)
		} else if exists(arg) {
			files = append(files, arg)
		} else {
			pkgs = append(pkgs, arg)
		}
	}

	var issues []nestif.Issue
	for _, f := range files {
		is, err := a.checkFile(checker, f)
		if err != nil {
			a.debug(err)
			continue
		}
		issues = append(issues, is...)
	}
	for _, d := range dirs {
		is, err := a.checkDir(checker, d)
		if err != nil {
			a.debug(err)
			continue
		}
		issues = append(issues, is...)
	}
	for _, p := range pkgs {
		is, err := a.checkPackage(checker, p)
		if err != nil {
			fmt.Fprintln(a.stdout, err)
			continue
		}
		issues = append(issues, is...)
	}
	return issues, nil
}

func (a *app) checkFile(checker *nestif.Checker, path string) ([]nestif.Issue, error) {
	dir := filepath.Dir(path)
	for _, p := range a.excludePatterns {
		if p.MatchString(dir) {
			return []nestif.Issue{}, nil
		}
	}

	src, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	if len(f.Comments) > 0 && isGenerated(src) {
		return nil, fmt.Errorf("%s is a generated file", path)
	}

	return checker.Check(f, fset), nil
}

// Copyright (c) 2013 The Go Authors. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file or at
// https://developers.google.com/open-source/licenses/bsd.
func (a *app) checkDir(checker *nestif.Checker, dirname string) ([]nestif.Issue, error) {
	for _, p := range a.excludePatterns {
		if p.MatchString(dirname) {
			return []nestif.Issue{}, nil
		}
	}
	pkg, err := build.ImportDir(dirname, 0)
	if err != nil {
		if _, nogo := err.(*build.NoGoError); nogo {
			// Don't complain if the failure is due to no Go source files.
			return nil, nil
		}
		return nil, err
	}
	return a.checkImportedPackage(checker, pkg)
}

func (a *app) checkPackage(checker *nestif.Checker, pkgname string) ([]nestif.Issue, error) {
	pkg, err := build.Import(pkgname, ".", 0)
	if err != nil {
		if _, nogo := err.(*build.NoGoError); nogo {
			// Don't complain if the failure is due to no Go source files.
			return nil, nil
		}
		return nil, err
	}
	return a.checkImportedPackage(checker, pkg)
}

func (a *app) checkImportedPackage(checker *nestif.Checker, pkg *build.Package) (issues []nestif.Issue, err error) {
	var files []string
	files = append(files, pkg.GoFiles...)
	files = append(files, pkg.CgoFiles...)
	files = append(files, pkg.TestGoFiles...)
	// TODO: Reduce allocation.
	if pkg.Dir != "." {
		for _, f := range files {
			is, err := a.checkFile(checker, filepath.Join(pkg.Dir, f))
			if err != nil {
				a.debug(err)
				continue
			}
			issues = append(issues, is...)
		}
	}
	return
}

func (a *app) write(issues []nestif.Issue) {
	if a.outJSON {
		js, err := json.Marshal(issues)
		if err != nil {
			fmt.Fprintln(a.stderr, err)
			return
		}
		fmt.Fprintln(a.stdout, string(js))
		return
	}
	for i, issue := range issues {
		if i >= a.top {
			return
		}
		fmt.Fprintln(a.stdout, errformat(issue.Pos.Filename, issue.Pos.Line, issue.Pos.Column, issue.Message))
	}
}

func (a *app) debug(err error) {
	if a.verbose {
		fmt.Fprintln(a.stdout, err)
	}
}

func isDir(filename string) bool {
	fi, err := os.Stat(filename)
	return err == nil && fi.IsDir()
}

func exists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

// isGenerated reports whether the source file is generated code
// according the rules from https://golang.org/s/generatedcode.
func isGenerated(src []byte) bool {
	var (
		genHdr = []byte("// Code generated ")
		genFtr = []byte(" DO NOT EDIT.")
	)
	sc := bufio.NewScanner(bytes.NewReader(src))
	for sc.Scan() {
		b := sc.Bytes()
		if bytes.HasPrefix(b, genHdr) && bytes.HasSuffix(b, genFtr) && len(b) >= len(genHdr)+len(genFtr) {
			return true
		}
	}
	return false
}
