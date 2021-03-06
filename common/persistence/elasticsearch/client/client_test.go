// The MIT License
//
// Copyright (c) 2020 Temporal Technologies Inc.  All rights reserved.
//
// Copyright (c) 2020 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package client

import (
	"fmt"
	"testing"

	"github.com/olivere/elastic/v7"
	"github.com/stretchr/testify/require"
)

func Test_BuildPutMappingBody(t *testing.T) {
	tests := []struct {
		root     string
		expected string
	}{
		{
			root:     "Attr",
			expected: "map[properties:map[Attr:map[properties:map[testKey:map[type:text]]]]]",
		},
		{
			root:     "",
			expected: "map[properties:map[testKey:map[type:text]]]",
		},
	}
	k := "testKey"
	v := "text"

	var client clientV6
	for _, test := range tests {
		require.Equal(t, test.expected, fmt.Sprintf("%v", client.buildPutMappingBody(test.root, k, v)))
	}
}
func Test_ConvertV7Sorters(t *testing.T) {
	sortersV7 := make([]elastic.Sorter, 2)
	sortersV7[0] = elastic.NewFieldSort("test")
	sortersV7[1] = elastic.NewFieldSort("test2")

	sortersV6 := convertV7SortersToV6(sortersV7)
	require.NotNil(t, sortersV6[0])
	source0, err0 := sortersV6[0].Source()
	require.NoError(t, err0)
	require.NotNil(t, source0)
}
