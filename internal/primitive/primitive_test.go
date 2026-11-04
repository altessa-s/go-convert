// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package primitive

import (
	"math"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConvertWidensAndNarrows(t *testing.T) {
	t.Parallel()

	src := int64(42)

	var dst int8

	handled, overflow := Convert(reflect.ValueOf(&src).Elem(), reflect.ValueOf(&dst).Elem(), true)
	require.True(t, handled)
	require.False(t, overflow)
	require.Equal(t, int8(42), dst)
}

func TestConvertReportsOverflowWithoutWriting(t *testing.T) {
	t.Parallel()

	src := int64(math.MaxInt16)
	dst := int8(7)
	handled, overflow := Convert(reflect.ValueOf(&src).Elem(), reflect.ValueOf(&dst).Elem(), true)
	require.False(t, handled)
	require.True(t, overflow)
	require.Equal(t, int8(7), dst)
}

func TestConvertAllocatesNilPointerForNonZeroSource(t *testing.T) {
	t.Parallel()

	src := 3.5

	var dst *float32

	handled, _ := Convert(reflect.ValueOf(&src).Elem(), reflect.ValueOf(&dst).Elem(), false)
	require.True(t, handled)
	require.InDelta(t, 3.5, *dst, 0)

	zero := 0.0

	var untouched *float32

	handled, _ = Convert(reflect.ValueOf(&zero).Elem(), reflect.ValueOf(&untouched).Elem(), false)
	require.False(t, handled)
	require.Nil(t, untouched)
}

func TestConvertRejectsUnregisteredPairs(t *testing.T) {
	t.Parallel()

	src := "text"

	var dst int

	handled, _ := Convert(reflect.ValueOf(&src).Elem(), reflect.ValueOf(&dst).Elem(), false)
	require.False(t, handled)

	_, ok := Lookup(reflect.Struct, reflect.Int)
	require.False(t, ok)
	_, ok = Lookup(reflect.Kind(255), reflect.Int)
	require.False(t, ok)
}

func TestWouldOverflow(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		src  any
		dst  any
		want bool
	}{
		{"int fits", int64(127), int8(0), false},
		{"int overflows", int64(128), int8(0), true},
		{"negative to unsigned", -1, uint(0), true},
		{"unsigned beyond signed max", uint64(math.MaxUint64), int64(0), true},
		{"fractional float to int", 1.5, 0, true},
		{"negative float to unsigned", -1.0, uint(0), true},
		{"float64 beyond float32", math.MaxFloat64, float32(0), true},
		{"int to float64", math.MaxInt64, 0.0, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := WouldOverflow(reflect.ValueOf(tc.src), reflect.New(reflect.TypeOf(tc.dst)).Elem())
			require.Equal(t, tc.want, got)
		})
	}
}
