// Copyright 2020 Ryo Nakao <nakabonne@gmail.com>.
//
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package nestif provides an API to detect deeply nested if statements.
package nestif

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"io"
)

// Issue represents an issue of root if statement that has nested ifs.
type Issue struct {
	Pos        token.Position
	Complexity int
	Message    string
}

// Checker represents a checker that finds nested if statements.
type Checker struct {
	// Minimum complexity to report.
	MinComplexity int
	// Include the simple "if err != nil" in the calculation.
	//IfErr bool

	// For debug mode.
	debugWriter io.Writer
	issues      []Issue
}

// Check inspects a single file and returns found issues.
func (c *Checker) Check(f *ast.File, fset *token.FileSet) []Issue {
	c.issues = []Issue{} // refresh
	ast.Inspect(f, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			return true
		}
		for _, stmt := range fn.Body.List {
			c.checkFunc(&stmt, fset)
		}
		return true
	})

	return c.issues
}

// checkFunc inspects a function and sets a list of issues if there are.
func (c *Checker) checkFunc(stmt *ast.Stmt, fset *token.FileSet) {
	ast.Inspect(*stmt, func(n ast.Node) bool {
		ifStmt, ok := n.(*ast.IfStmt)
		if !ok {
			return true
		}

		c.checkIf(ifStmt, fset)
		return false
	})
}

// checkIf inspects a if statement and sets an issue if there is.
func (c *Checker) checkIf(stmt *ast.IfStmt, fset *token.FileSet) {
	v := newVisitor()
	ast.Walk(v, stmt)
	if v.complexity < c.MinComplexity {
		return
	}
	pos := fset.Position(stmt.Pos())
	c.issues = append(c.issues, Issue{
		Pos:        pos,
		Complexity: v.complexity,
		Message:    c.makeMessage(pos.Filename, pos.Line, pos.Column, v.complexity, stmt.Cond, fset),
	})
}

type visitor struct {
	complexity int
	nesting    int
	// To avoid adding complexity including nesting level to `else if`.
	elseifs map[ast.Node]bool
	// Include the simple "if err != nil" in the calculation.
	//ifErr bool
}

func newVisitor() *visitor {
	return &visitor{
		elseifs: make(map[ast.Node]bool),
		//ifErr: c.IfErr,
	}
}

// Visit traverses an AST in depth-first order by calling itself
// recursively, and calculates the complexities of if statements.
func (v *visitor) Visit(n ast.Node) ast.Visitor {
	ifStmt, ok := n.(*ast.IfStmt)
	if !ok {
		return v
	}

	// Ignore the simple "if err != nil"
	//if !v.ifErr && ifErr(ifStmt.Cond) {
	//	return nil
	//}

	v.incComplexity(ifStmt)
	v.nesting++
	ast.Walk(v, ifStmt.Body)
	v.nesting--

	if _, ok := ifStmt.Else.(*ast.BlockStmt); ok {
		v.complexity++
		v.nesting++
		ast.Walk(v, ifStmt.Else)
		v.nesting--
	} else if _, ok := ifStmt.Else.(*ast.IfStmt); ok {
		v.elseifs[ifStmt.Else] = true
		ast.Walk(v, ifStmt.Else)
	}

	return nil
}

func (v *visitor) incComplexity(n *ast.IfStmt) {
	// In case of `else if`, increase by 1.
	if v.elseifs[n] {
		v.complexity++
	} else {
		v.complexity += v.nesting
	}
}

func (c *Checker) makeMessage(file string, line, col, complexity int, cond ast.Expr, fset *token.FileSet) string {
	p := &printer.Config{}
	b := new(bytes.Buffer)
	if err := p.Fprint(b, fset, cond); err != nil {
		c.debug("failed to convert condition into string: %v", err)
	}
	msg := fmt.Sprintf("`if %s` is nested (complexity: %d)", b.String(), complexity)
	return errformat(file, line, col, msg)
}

func errformat(file string, line, col int, msg string) string {
	return fmt.Sprintf("%s:%d:%d: %s", file, line, col, msg)
}

// DebugMode makes it possible to emit debug logs.
func (c *Checker) DebugMode(w io.Writer) {
	c.debugWriter = w
}

func (c *Checker) debug(format string, a ...interface{}) {
	if c.debugWriter != nil {
		fmt.Fprintf(c.debugWriter, format, a...)
	}
}

/*
// ifErr checks if the given condition is "if err != nil"
func ifErr(cond ast.Expr) bool {
	expr, ok := cond.(*ast.BinaryExpr)
	if !ok {
		return false
	}
	// TODO: Check if the type of X is error
	y, ok := expr.Y.(*ast.Ident)
	if !ok {
		return false
	}
	if y.String() != "nil" {
		return false
	}
	// TODO: Check if operator is "!="
	return true
}

var errorType = types.Universe.Lookup("error").Type().Underlying().(*types.Interface)

func isErrorType(t types.Type) bool {
	return types.Implements(t, errorType)
}
*/
