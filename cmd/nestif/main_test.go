// Copyright 2020 Ryo Nakao <nakabonne@gmail.com>.
//
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRun(t *testing.T) {
	cases := []struct {
		name          string
		args          []string
		verbose       bool
		outJSON       bool
		minComplexity int
		top           int
		excludeDirs   []string
		want          string
		code          int
	}{
		{
			name:          "increment for breaks in the linear flow",
			args:          []string{"../../testdata/a.go"},
			minComplexity: 1,
			top:           10,
			want:          "../../testdata/a.go:9:2: `if b1` is deeply nested (complexity: 1)\n",
			code:          0,
		},
		{
			name:          "show only top 2",
			args:          []string{"../../testdata/d.go"},
			minComplexity: 1,
			top:           2,
			want:          "../../testdata/d.go:16:2: `if b1` is deeply nested (complexity: 3)\n../../testdata/d.go:6:2: `if b1` is deeply nested (complexity: 1)\n",
			code:          0,
		},
		{
			name:          "show only those with complexity of 2 or more",
			args:          []string{"../../testdata/d.go"},
			minComplexity: 2,
			top:           10,
			want:          "../../testdata/d.go:16:2: `if b1` is deeply nested (complexity: 3)\n",
			code:          0,
		},
		{
			name:          "ignore generated file",
			args:          []string{"../../testdata/generated.go"},
			minComplexity: 1,
			top:           10,
			want:          "",
			code:          0,
		},
		{
			name:          "directory given",
			args:          []string{"../../testdata/a"},
			minComplexity: 1,
			top:           10,
			want:          "../../testdata/a/a.go:8:2: `if b1` is deeply nested (complexity: 1)\n",
			code:          0,
		},
		{
			name:          "Check files recursively",
			args:          []string{"../../testdata/a/..."},
			minComplexity: 1,
			top:           10,
			want:          "../../testdata/a/a.go:8:2: `if b1` is deeply nested (complexity: 1)\n../../testdata/a/b/a.go:8:2: `if b1` is deeply nested (complexity: 1)\n",
			code:          0,
		},
		{
			name:          "Check all files recursively",
			verbose:       true,
			args:          []string{"./..."},
			minComplexity: 1,
			top:           10,
			want:          "",
			code:          0,
		},
		{
			name:          "no args given",
			verbose:       true,
			args:          []string{},
			minComplexity: 1,
			top:           10,
			want:          "",
			code:          0,
		},
		{
			name:          "package name given",
			args:          []string{"github.com/nakabonne/nestif/testdata/a"},
			minComplexity: 1,
			top:           10,
			want: func() string {
				path, _ := filepath.Abs("../../testdata/a/a.go")
				return path + ":8:2: `if b1` is deeply nested (complexity: 1)\n"
			}(),
			code: 0,
		},
		{
			name:          "json output",
			outJSON:       true,
			args:          []string{"../../testdata/a.go"},
			minComplexity: 1,
			top:           10,
			want:          "[{\"Pos\":{\"Filename\":\"../../testdata/a.go\",\"Offset\":78,\"Line\":9,\"Column\":2},\"Complexity\":1,\"Message\":\"`if b1` is deeply nested (complexity: 1)\"}]\n",
			code:          0,
		},
		{
			name:          "exclude-dirs given",
			args:          []string{"../../testdata"},
			minComplexity: 1,
			top:           10,
			excludeDirs:   []string{"../../testdata"},
			want:          "",
			code:          0,
		},
		{
			name:          "wrong exclude-dirs given",
			args:          []string{"../../testdata"},
			minComplexity: 1,
			top:           10,
			excludeDirs:   []string{"(^|/../../testdata"},
			want:          "failed to parse exclude dir pattern: error parsing regexp: missing closing ): `(^|/../../testdata`\n",
			code:          1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := new(bytes.Buffer)
			a := app{
				verbose:       tc.verbose,
				outJSON:       tc.outJSON,
				minComplexity: tc.minComplexity,
				top:           tc.top,
				excludeDirs:   tc.excludeDirs,
				stdout:        b,
				stderr:        b,
			}
			c := a.run(tc.args)
			assert.Equal(t, tc.code, c)
			assert.Equal(t, tc.want, b.String())
		})
	}
}
