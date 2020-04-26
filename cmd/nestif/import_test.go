// Copyright 2020 Ryo Nakao <ryo@nakao.dev>.
//
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAllPackagesInFS(t *testing.T) {
	cases := []struct {
		name    string
		pattern string
		want    []string
		log     string
	}{
		{
			name:    "... glob operator",
			pattern: "./...",
			want:    []string{"./."},
		},
		{
			name:    "parent directly",
			pattern: "../../...",
			want:    []string{"../..", "../../cmd/nestif"},
		},
		{
			name:    "... glob operator",
			pattern: "../../testdata/nogo/...",
			want:    nil,
			log:     "warning: \"../../testdata/nogo/...\" matched no packages\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := new(bytes.Buffer)
			s := allPackagesInFS(tc.pattern, b)
			assert.ElementsMatch(t, tc.want, s)
			assert.Equal(t, tc.log, b.String())
		})
	}
}
