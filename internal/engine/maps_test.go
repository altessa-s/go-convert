// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package engine

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

type mapValue struct{ X int32 }

type mapValueDst struct{ X int64 }

type mapKey struct{ K int }

type mapKeyDst struct{ K int }

type mapSrc struct {
	Scalars map[int32]float64
	Strings map[string]string
	Named   map[namedString]uint8
	Structs map[string]mapValue
	Ptrs    map[int]*mapValue
	Keys    map[mapKey]int
	Formats map[int]int
}

type mapDst struct {
	Scalars map[int64]float32
	Strings map[namedString]namedString
	Named   map[string]int64
	Structs map[string]mapValueDst
	Ptrs    map[int64]*mapValueDst
	Keys    map[mapKeyDst]int
	Formats map[string]string
}

// TestMapConversions covers the key and value buffers of the map walker:
// every stored entry must be the converted value of its own source entry.
func TestMapConversions(t *testing.T) {
	t.Parallel()

	big := 1e40
	in := &mapSrc{
		Scalars: map[int32]float64{1: 1.5, -2: big},
		Strings: map[string]string{"a": "b"},
		Named:   map[namedString]uint8{"n": 7},
		Structs: map[string]mapValue{"s": {X: 3}, "t": {X: -4}},
		Ptrs:    map[int]*mapValue{1: {X: 5}, 2: nil},
		Keys:    map[mapKey]int{{K: 1}: 1},
		Formats: map[int]int{65: 66},
	}

	for _, cfg := range []Config{{}, {IgnoreNilValues: true}, {CopySlices: true}} {
		var out mapDst
		require.NoError(t, New(cfg).Convert(in, &out))
		require.Equal(t, map[int64]float32{1: 1.5, -2: float32(big)}, out.Scalars)
		require.Equal(t, map[namedString]namedString{"a": "b"}, out.Strings)
		require.Equal(t, map[string]int64{"n": 7}, out.Named)
		require.Equal(t, map[string]mapValueDst{"s": {X: 3}, "t": {X: -4}}, out.Structs)
		require.Equal(t, map[int64]*mapValueDst{1: {X: 5}, 2: nil}, out.Ptrs)
		require.Equal(t, map[mapKeyDst]int{{K: 1}: 1}, out.Keys)
		require.Equal(t, map[string]string{"65": "66"}, out.Formats)
	}

	type keys struct{ M map[int64]int }

	type narrowKeys struct{ M map[int8]int }

	var narrow narrowKeys
	require.Equal(t, OverflowError{Field: "m.key", From: reflect.Int64, To: reflect.Int8, Value: int64(300)},
		New(Config{OverflowCheck: true}).Convert(&keys{M: map[int64]int{300: 1}}, &narrow))

	type values struct{ M map[string]int64 }

	type narrowValues struct{ M map[string]int8 }

	var narrowV narrowValues
	require.Equal(t, OverflowError{Field: "m.[k]", From: reflect.Int64, To: reflect.Int8, Value: int64(300)},
		New(Config{OverflowCheck: true}).Convert(&values{M: map[string]int64{"k": 300}}, &narrowV))
	require.NoError(t, New(Config{}).Convert(&values{M: map[string]int64{"k": 300}}, &narrowV))
	require.Equal(t, map[string]int8{"k": 44}, narrowV.M)
}
