// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package convert_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	converter "github.com/altessa-s/go-convert"
)

func TestConvertMapPointerValues(t *testing.T) {
	t.Parallel()

	t.Run("int to *int", func(t *testing.T) {
		t.Parallel()

		type A struct{ M map[string]int }

		type B struct{ M map[string]*int }

		var dst B

		require.NotPanics(t, func() { converter.Convert(A{M: map[string]int{"a": 1}}, &dst) })
		require.NotNil(t, dst.M["a"])
		require.Equal(t, 1, *dst.M["a"])
	})

	t.Run("*int to *int64", func(t *testing.T) {
		t.Parallel()

		type A struct{ M map[string]*int }

		type B struct{ M map[string]*int64 }

		one := 1

		var dst B

		require.NotPanics(t, func() { converter.Convert(A{M: map[string]*int{"a": &one}}, &dst) })
		require.NotNil(t, dst.M["a"])
		require.Equal(t, int64(1), *dst.M["a"])
	})

	t.Run("*int to *int keeps value", func(t *testing.T) {
		t.Parallel()

		type A struct{ M map[string]*int }

		type B struct{ M map[string]*int }

		one := 1

		var dst B
		converter.Convert(A{M: map[string]*int{"a": &one}}, &dst)
		require.Equal(t, 1, *dst.M["a"])
	})

	t.Run("*int to int", func(t *testing.T) {
		t.Parallel()

		type A struct{ M map[string]*int }

		type B struct{ M map[string]int }

		one := 1

		var dst B
		converter.Convert(A{M: map[string]*int{"a": &one}}, &dst)
		require.Equal(t, map[string]int{"a": 1}, dst.M)
	})

	t.Run("int to int64", func(t *testing.T) {
		t.Parallel()

		type A struct{ M map[string]int }

		type B struct{ M map[string]int64 }

		var dst B
		converter.Convert(A{M: map[string]int{"a": 1}}, &dst)
		require.Equal(t, map[string]int64{"a": 1}, dst.M)
	})
}

type mapSliceItem struct {
	Name string
}

type mapToSliceSource struct {
	Items map[string]mapSliceItem
}

type mapToSliceDest struct {
	Items map[string][]mapSliceItem
}

func TestConvertMapStructValueToSliceValue(t *testing.T) {
	t.Parallel()

	src := mapToSliceSource{Items: map[string]mapSliceItem{"k": {Name: "n"}}}

	var dst mapToSliceDest
	converter.Convert(src, &dst)
	require.Len(t, dst.Items["k"], 1)
	require.Equal(t, "n", dst.Items["k"][0].Name)
}
