// Copyright 2020 Ryo Nakao <ryo@nakao.dev>.
//
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package nestif

import (
	"bytes"
	"go/parser"
	"go/token"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheck(t *testing.T) {
	cases := []struct {
		name          string
		filepath      string
		minComplexity int
		want          []Issue
	}{
		{
			name:          "increment for breaks in the linear flow",
			filepath:      "./testdata/a.go",
			minComplexity: 1,
			want: []Issue{
				{
					Pos: token.Position{
						Filename: "./testdata/a.go",
						Offset:   78,
						Line:     9,
						Column:   2,
					},
					Complexity: 1,
					Message:    "`if b1` has complex nested blocks (complexity: 1)",
				},
			},
		},
		{
			name:          "increment for nested flow-break structures",
			filepath:      "./testdata/b.go",
			minComplexity: 1,
			want: []Issue{
				{
					Pos: token.Position{
						Filename: "./testdata/b.go",
						Offset:   55,
						Line:     5,
						Column:   2,
					},
					Complexity: 9,
					Message:    "`if b1` has complex nested blocks (complexity: 9)",
				},
			},
		},
		{
			name:          "no nesting increment is assessed for else and else if",
			filepath:      "./testdata/c.go",
			minComplexity: 1,
			want: []Issue{
				{
					Pos: token.Position{
						Filename: "./testdata/c.go",
						Offset:   56,
						Line:     6,
						Column:   2,
					},
					Complexity: 4,
					Message:    "`if b1` has complex nested blocks (complexity: 4)",
				},
				{
					Pos: token.Position{
						Filename: "./testdata/c.go",
						Offset:   145,
						Line:     14,
						Column:   2,
					},
					Complexity: 4,
					Message:    "`if b1` has complex nested blocks (complexity: 4)",
				},
			},
		},
		{
			name:          "complexity is less than given num",
			filepath:      "./testdata/a.go",
			minComplexity: 2,
			want:          []Issue{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			checker := &Checker{
				MinComplexity: tc.minComplexity,
			}
			src, _ := ioutil.ReadFile(tc.filepath)
			fset := token.NewFileSet()
			f, _ := parser.ParseFile(fset, tc.filepath, src, parser.ParseComments)
			i := checker.Check(f, fset)

			assert.ElementsMatch(t, tc.want, i)
		})
	}
}

func TestDebug(t *testing.T) {
	cases := []struct {
		name       string
		format     string
		values     []interface{}
		checkerGen func(*bytes.Buffer) *Checker
		want       string
	}{
		{
			name:   "not debug mode",
			format: "warning: %s",
			values: []interface{}{"foo"},
			checkerGen: func(b *bytes.Buffer) *Checker {
				return &Checker{}
			},
			want: "",
		},
		{
			name:   "debug mode",
			format: "warning: %s",
			values: []interface{}{"foo"},
			checkerGen: func(b *bytes.Buffer) *Checker {
				c := &Checker{}
				c.DebugMode(b)
				return c
			},
			want: "warning: foo",
		},
		{
			name:   "debug with multiple values",
			format: "warning: %s %d %T",
			values: []interface{}{"foo", 1, true},
			checkerGen: func(b *bytes.Buffer) *Checker {
				c := &Checker{}
				c.DebugMode(b)
				return c
			},
			want: "warning: foo 1 bool",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := new(bytes.Buffer)
			c := tc.checkerGen(b)
			c.debug(tc.format, tc.values...)

			assert.Equal(t, tc.want, b.String())
		})
	}
}
