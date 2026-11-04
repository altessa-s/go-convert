// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package convert_test

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	converter "github.com/altessa-s/go-convert"
)

func TestOverflowCheckPanicsOnSignedOverflow(t *testing.T) {
	type Src struct{ Age int64 }

	type Dst struct{ Age int8 }

	src := &Src{Age: 1000}
	dst := &Dst{}

	defer func() {
		r := recover()
		require.NotNil(t, r, "expected panic for int64(1000) → int8 with overflow check")
		oe, ok := r.(converter.OverflowError)
		require.True(t, ok, "expected OverflowError, got %T: %v", r, r)
		require.Equal(t, reflect.Int64, oe.From)
		require.Equal(t, reflect.Int8, oe.To)
		require.NotEmpty(t, oe.Error())
	}()

	converter.Convert(src, dst, converter.WithOverflowCheck())
}

func TestOverflowCheckPanicsOnUnsignedOverflow(t *testing.T) {
	type Src struct{ Count uint64 }

	type Dst struct{ Count uint8 }

	src := &Src{Count: 300}
	dst := &Dst{}

	defer func() {
		r := recover()
		require.NotNil(t, r, "expected panic for uint64(300) → uint8 with overflow check")
		oe, ok := r.(converter.OverflowError)
		require.True(t, ok, "expected OverflowError, got %T: %v", r, r)
		require.Equal(t, reflect.Uint64, oe.From)
		require.Equal(t, reflect.Uint8, oe.To)
	}()

	converter.Convert(src, dst, converter.WithOverflowCheck())
}

func TestOverflowCheckAllowsInRangeValues(t *testing.T) {
	type Src struct{ Age int64 }

	type Dst struct{ Age int8 }

	src := &Src{Age: 42}
	dst := &Dst{}

	converter.Convert(src, dst, converter.WithOverflowCheck())

	require.Equal(t, int8(42), dst.Age)
}

func TestOverflowWithoutCheckTruncates(t *testing.T) {
	type Src struct{ Age int64 }

	type Dst struct{ Age int8 }

	src := &Src{Age: 1000}
	dst := &Dst{}

	// Without WithOverflowCheck — should silently truncate (backwards compat)
	converter.Convert(src, dst)

	// Value is truncated, not 1000
	require.True(t, dst.Age != 0 || src.Age == 0, "conversion should have happened even with truncation")
}

type overflowSource struct {
	N int64
}

type overflowDest struct {
	N int8
}

func TestConvertEReturnsOverflowError(t *testing.T) {
	t.Parallel()

	var dst overflowDest

	_, err := converter.ConvertE(overflowSource{N: 300}, &dst, converter.WithOverflowCheck())
	require.Error(t, err)

	var overflow converter.OverflowError
	require.ErrorAs(t, err, &overflow)
}

func TestOverflowCheckMapKeysAndValues(t *testing.T) {
	t.Parallel()
	t.Run("map key", func(t *testing.T) {
		t.Parallel()

		type source struct{ Values map[int64]string }

		type destination struct{ Values map[uint8]string }

		var dst destination

		_, err := converter.ConvertE(source{Values: map[int64]string{257: "value"}}, &dst, converter.WithOverflowCheck())

		var overflow converter.OverflowError
		require.ErrorAs(t, err, &overflow)
		require.Equal(t, reflect.Int64, overflow.From)
		require.Equal(t, reflect.Uint8, overflow.To)
		require.Equal(t, "values.key", overflow.Field)
	})
	t.Run("map value", func(t *testing.T) {
		t.Parallel()

		type source struct{ Values map[string]int64 }

		type destination struct{ Values map[string]uint8 }

		var dst destination

		_, err := converter.ConvertE(source{Values: map[string]int64{"key": 257}}, &dst, converter.WithOverflowCheck())

		var overflow converter.OverflowError
		require.ErrorAs(t, err, &overflow)
		require.Equal(t, reflect.Int64, overflow.From)
		require.Equal(t, reflect.Uint8, overflow.To)
		require.Equal(t, "values.[key]", overflow.Field)
	})
}

func TestOverflowCheckUintptr(t *testing.T) {
	t.Parallel()

	type source struct{ Value uintptr }

	type destination struct{ Value uint8 }

	var dst destination

	_, err := converter.ConvertE(source{Value: uintptr(300)}, &dst, converter.WithOverflowCheck())

	var overflow converter.OverflowError
	require.ErrorAs(t, err, &overflow)
	require.Equal(t, reflect.Uintptr, overflow.From)
	require.Equal(t, reflect.Uint8, overflow.To)

	dst = destination{}
	converter.Convert(source{Value: uintptr(42)}, &dst, converter.WithOverflowCheck())
	require.Equal(t, uint8(42), dst.Value)
}
