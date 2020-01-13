// Copyright 2020 Ryo Nakao <nakabonne@gmail.com>.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package nestif provides an API to detect deeply nested if statements.
package nestif

import (
	"fmt"
	"go/ast"
	"go/token"
	"io"
	"os"
)

// Issue represents an issue of root if statement that has nested ifs.
type Issue struct {
	Pos        token.Position
	Complexity int
	// Condition string such as "if a == b".
	Condition string
}

// Message makes a message with its own source position.
func (i *Issue) Message() string {
	msg := fmt.Sprintf("%s has nested if statements (complexity: %d)", i.Condition, i.Complexity)
	return errformat(i.Pos.Filename, i.Pos.Line, i.Pos.Column, msg)
}

// Checker represents a checker that finds nested if statements.
type Checker struct {
	// Minimum complexity to report.
	MinComplexity int
	// Ignore to check "if err != nil".
	IgnoreIfErr bool

	// For debug mode.
	logWriter io.Writer
}

// Check detects deeply nested if statements.
func (c *Checker) Check(f *ast.File, fset *token.FileSet) []Issue {
	var issues []Issue
	ast.Inspect(f, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			return true
		}
		for _, stmt := range fn.Body.List {
			is := c.checkFunc(&stmt, fset)
			if len(is) > 0 {
				issues = append(issues, is...)
			}
		}
		return true
	})
	c.debug("%d issues found\n", len(issues))

	return issues
}

// checkFunc inspects a function and return a list of Issue.
func (c *Checker) checkFunc(stmt *ast.Stmt, fset *token.FileSet) (issues []Issue) {
	ast.Inspect(*stmt, func(n ast.Node) bool {
		ifStmt, ok := n.(*ast.IfStmt)
		if !ok {
			return true
		}

		i := c.checkIf(ifStmt, fset)
		if i != nil {
			issues = append(issues, *i)
		}
		return false
	})
	return
}

// checkIf inspects a if statement and return an Issue.
func (c *Checker) checkIf(stmt *ast.IfStmt, fset *token.FileSet) *Issue {
	v := &visitor{}
	ast.Walk(v, stmt)
	if v.complexity < c.MinComplexity {
		return nil
	}

	return &Issue{
		Pos:        fset.Position(stmt.Pos()),
		Condition:  "if statement", // TODO: Use condition such as "if a == b".
		Complexity: v.complexity,
	}
}

type visitor struct {
	complexity int
	nesting    int
}

func (v *visitor) Visit(n ast.Node) ast.Visitor {
	ifStmt, ok := n.(*ast.IfStmt)
	if !ok {
		return v
	}

	v.complexity += v.nesting
	v.nesting++
	// TODO: Ignore "if err != nil"
	ast.Walk(v, ifStmt.Body)
	if ifStmt.Else != nil {
		ast.Walk(v, ifStmt.Else)
	}
	v.nesting--

	return nil
}

// DebugMode makes it possible to emit debug logs.
func (c *Checker) DebugMode() {
	c.logWriter = os.Stderr
}

func (c *Checker) debug(format string, a ...interface{}) {
	if c.logWriter != nil {
		fmt.Fprintf(c.logWriter, format, a...)
	}
}

func errformat(file string, line, col int, msg string) string {
	return fmt.Sprintf("%s:%d:%d: %s", file, line, col, msg)
}
