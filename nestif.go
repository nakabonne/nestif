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
	Depth int
	// Ignore the checks "if err != nil"
	IgnoreIfErr bool

	// For debug mode.
	logWriter io.Writer
}

// Check detects deeply nested if statements.
func (c *Checker) Check(f *ast.File, fset *token.FileSet) []Issue {
	var issues []Issue
	for _, decl := range f.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			// TODO: Check if nested if statement exists.
			fmt.Println(fn)
		}
	}
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
