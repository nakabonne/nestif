package nestif

import (
	"fmt"
	"go/ast"
	"go/token"
	"io"
	"os"
)

// Issue represents nested if statement.
type Issue struct {
	Pos   token.Position
	Depth int
}

func (i *Issue) Message() string {
	var msg string
	switch i.Depth {
	case 1:
		msg = fmt.Sprintf("a nested if statement found (depth: %d)", i.Depth)
	default:
		msg = fmt.Sprintf("a deeply nested if statement found (depth: %d)", i.Depth)
	}
	return fmt.Sprintf("%s:%d:%d: %s", i.Pos.Filename, i.Pos.Line, i.Pos.Column, msg)
}

type Checker struct {
	// Lower limit of nesting depth to check.
	MinDepth int
	// Ignore to check "if err != nil".
	IgnoreIfErr bool

	// For debug mode.
	logWriter io.Writer
}

// Check detects deeply nested if statements.
func (c *Checker) Check(f *ast.File, fset *token.FileSet) []Issue {
	var issues []Issue
	ast.Inspect(f, func(n ast.Node) bool {
		dec, ok := n.(*ast.FuncDecl)
		if !ok || dec.Body == nil {
			return true
		}

		for _, stmt := range dec.Body.List {
			ast.Inspect(stmt, func(n ast.Node) bool {
				if n, ok := n.(*ast.IfStmt); ok {
					count := 0
					ast.Inspect(n.Body, func(n ast.Node) bool {
						if _, ok := n.(*ast.IfStmt); ok {
							count++
						}
						return true
					})
					if count > 0 {
						issues = append(issues, Issue{
							Pos:   fset.Position(n.Pos()),
							Depth: count,
						})
					}
					return false
				}
				return true
			})
		}

		return true
	})
	return issues
}

func (c *Checker) DebugMode() {
	c.logWriter = os.Stderr
}

func (c *Checker) debug(format string, a ...interface{}) {
	if c.logWriter != nil {
		fmt.Fprintf(c.logWriter, format, a...)
	}
}
