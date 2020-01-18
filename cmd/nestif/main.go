// Copyright 2020 Ryo Nakao <nakabonne@gmail.com>.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/build"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/nakabonne/nestif"
)

var (
	flagSet = flag.NewFlagSet("nestif", flag.ContinueOnError)

	verbose       = flagSet.Bool("v", false, "verbose output")
	outJSON       = flagSet.Bool("json", false, "emit json format")
	minComplexity = flagSet.Int("min", 1, "minimum complexity to show")
	top           = flagSet.Int("top", 10, "show only the top N most complex if statements")
	sortIssue     = flagSet.Bool("sort", false, "sort in descending order of complexity")
	//iferr         = flagSet.Bool("iferr", false, `include the simple "if err != nil" in the calculation`)

	usage = func() {
		fmt.Fprintln(os.Stderr, "usage: nestif [<flag> ...] <Go files or directories or packages> ...")
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

	issues, err := check()
	if err != nil {
		flagSet.Usage()
	}

	if *sortIssue {
		sort.Slice(issues, func(i, j int) bool {
			return issues[i].Complexity > issues[j].Complexity
		})
	}
	// TODO: Implement top.

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
		fmt.Println(i.Message)
	}
}

func check() (issues []nestif.Issue, err error) {
	checker := &nestif.Checker{
		MinComplexity: *minComplexity,
		//IfErr:         *iferr,
	}
	if *verbose {
		checker.DebugMode()
	}

	// TODO: Improve performance.
	var files, dirs, pkgs []string
	for _, arg := range flagSet.Args() {
		if strings.HasSuffix(arg, "/...") && isDir(arg[:len(arg)-len("/...")]) {
			dirs = append(dirs, allPackagesInFS(arg)...)
		} else if isDir(arg) {
			dirs = append(dirs, arg)
		} else if exists(arg) {
			files = append(files, arg)
		} else {
			pkgs = append(pkgs, arg)
		}

	}
	if len(files) == 0 && len(dirs) == 0 && len(pkgs) == 0 {
		err = errors.New("")
		return
	}
	for _, f := range files {
		is, err := checkFile(checker, f)
		if err != nil {
			if *verbose {
				fmt.Println(err)
			}
			continue
		}
		issues = append(issues, is...)
	}
	for _, d := range dirs {
		is, err := checkDir(checker, d)
		if err != nil {
			if *verbose {
				fmt.Println(err)
			}
			continue
		}
		issues = append(issues, is...)
	}
	for _, p := range pkgs {
		is, err := checkPackage(checker, p)
		if err != nil {
			if *verbose {
				fmt.Println(err)
			}
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
		return nil, fmt.Errorf("%s is a generated file\n", filepath)
	}

	return checker.Check(f, fset), nil
}

// Copyright (c) 2013 The Go Authors. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file or at
// https://developers.google.com/open-source/licenses/bsd.

func checkDir(checker *nestif.Checker, dirname string) ([]nestif.Issue, error) {
	pkg, err := build.ImportDir(dirname, 0)
	if err != nil {
		if _, nogo := err.(*build.NoGoError); nogo {
			// Don't complain if the failure is due to no Go source files.
			return nil, nil
		}
		return nil, err
	}
	return checkImportedPackage(checker, pkg)
}

func checkPackage(checker *nestif.Checker, pkgname string) ([]nestif.Issue, error) {
	pkg, err := build.Import(pkgname, ".", 0)
	if err != nil {
		if _, nogo := err.(*build.NoGoError); nogo {
			// Don't complain if the failure is due to no Go source files.
			return nil, nil
		}
		return nil, err
	}
	return checkImportedPackage(checker, pkg)
}

func checkImportedPackage(checker *nestif.Checker, pkg *build.Package) (issues []nestif.Issue, err error) {
	var files []string
	files = append(files, pkg.GoFiles...)
	files = append(files, pkg.CgoFiles...)
	files = append(files, pkg.TestGoFiles...)
	// TODO: Improve performance.
	if pkg.Dir != "." {
		for _, f := range files {
			is, err := checkFile(checker, filepath.Join(pkg.Dir, f))
			if err != nil {
				if *verbose {
					fmt.Println(err)
				}
				continue
			}
			issues = append(issues, is...)
		}
	}
	return
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
