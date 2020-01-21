// Copyright 2020 Ryo Nakao <nakabonne@gmail.com>.
//
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package nestif

import (
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
						Offset:   77,
						Line:     8,
						Column:   2,
					},
					Complexity: 1,
					Message:    "./testdata/a.go:8:2: `if b1` is nested (complexity: 1)",
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
					Message:    "./testdata/b.go:5:2: `if b1` is nested (complexity: 9)",
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
						Offset:   69,
						Line:     6,
						Column:   2,
					},
					Complexity: 3,
					Message:    "./testdata/c.go:6:2: `if b1` is nested (complexity: 3)",
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
