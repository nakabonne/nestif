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
	"flag"
	"fmt"
	"go/build"
	"go/parser"
	"go/token"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/nakabonne/nestif"
)

var (
	flagSet = flag.NewFlagSet("nestif", flag.ContinueOnError)
	usage   = func() {
		fmt.Fprintln(os.Stderr, "usage: nestif [<flag> ...] <Go files or directories or packages> ...")
		flagSet.PrintDefaults()
	}
)

type app struct {
	verbose       bool
	outJSON       bool
	minComplexity int
	top           int
	stdout        io.Writer
	stderr        io.Writer
}

func main() {
	a := &app{
		stdout: os.Stdout,
		stderr: os.Stderr,
	}
	flagSet.BoolVar(&a.verbose, "v", false, "verbose output")
	flagSet.BoolVar(&a.outJSON, "json", false, "emit json format")
	flagSet.IntVar(&a.minComplexity, "min", 1, "minimum complexity to show")
	flagSet.IntVar(&a.top, "top", 10, "show only the top N most complex if statements")
	flagSet.Usage = usage
	if err := flagSet.Parse(os.Args[1:]); err != nil {
		if err != flag.ErrHelp {
			fmt.Fprintln(a.stderr, err)
		}
		return
	}

	a.run(flagSet.Args())
}

func (a *app) run(args []string) {
	issues := a.check(args)

	sort.Slice(issues, func(i, j int) bool {
		return issues[i].Complexity > issues[j].Complexity
	})

	a.write(issues)
}

func (a *app) check(args []string) (issues []nestif.Issue) {
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

	for _, f := range files {
		is, err := checkFile(checker, f)
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
	return
}

func isDir(filename string) bool {
	fi, err := os.Stat(filename)
	return err == nil && fi.IsDir()
}

func exists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

func checkFile(checker *nestif.Checker, filepath string) ([]nestif.Issue, error) {
	src, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filepath, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	if len(f.Comments) > 0 && isGenerated(src) {
		return nil, fmt.Errorf("%s is a generated file", filepath)
	}

	return checker.Check(f, fset), nil
}

// Copyright (c) 2013 The Go Authors. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file or at
// https://developers.google.com/open-source/licenses/bsd.
func (a *app) checkDir(checker *nestif.Checker, dirname string) ([]nestif.Issue, error) {
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
			is, err := checkFile(checker, filepath.Join(pkg.Dir, f))
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
		fmt.Fprintln(a.stdout, issue.Message)
	}
}

func (a *app) debug(err error) {
	if a.verbose {
		fmt.Fprintln(a.stdout, err)
	}
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
