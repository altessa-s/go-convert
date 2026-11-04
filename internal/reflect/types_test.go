// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package reflect_test

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	reflectutils "github.com/altessa-s/go-convert/internal/reflect"
)

type testStruct struct{}

func TestIndirectType(t *testing.T) {
	tests := []struct {
		name string
		typ  reflect.Type
		want reflect.Type
	}{
		{"non-pointer", reflect.TypeFor[testStruct](), reflect.TypeFor[testStruct]()},
		{"single pointer", reflect.TypeFor[*testStruct](), reflect.TypeFor[testStruct]()},
		{"double pointer", reflect.TypeFor[**testStruct](), reflect.TypeFor[testStruct]()},
		{"int", reflect.TypeFor[int](), reflect.TypeFor[int]()},
		{"*int", reflect.TypeFor[*int](), reflect.TypeFor[int]()},
		{"string", reflect.TypeFor[string](), reflect.TypeFor[string]()},
		{"*string", reflect.TypeFor[*string](), reflect.TypeFor[string]()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reflectutils.IndirectType(tt.typ)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestIndirectValueAndMakeDstPointerDepth(t *testing.T) {
	t.Parallel()

	value := 7
	valuePtr := &value
	got := reflectutils.IndirectValue(reflect.ValueOf(&valuePtr))
	require.Equal(t, int64(7), got.Int())

	var nilInner *int

	got = reflectutils.IndirectValue(reflect.ValueOf(&nilInner))
	require.False(t, got.IsValid())

	var destination **int

	dst := reflect.ValueOf(&destination).Elem()
	reflectutils.MakeDst(&dst, reflect.TypeFor[int]())
	require.NotNil(t, destination)
	require.NotNil(t, *destination)
	require.Equal(t, 0, **destination)
}

func TestMakeDst_Pointer(t *testing.T) {
	var dst *testStruct

	v := reflect.ValueOf(&dst).Elem()
	typ := reflect.TypeFor[testStruct]()

	reflectutils.MakeDst(&v, typ)

	require.Equal(t, reflect.Struct, v.Kind())
}

func TestMakeDst_NonPointer(t *testing.T) {
	var dst testStruct

	v := reflect.ValueOf(&dst).Elem()
	typ := reflect.TypeFor[testStruct]()

	reflectutils.MakeDst(&v, typ)

	require.Equal(t, reflect.Struct, v.Kind())
}

func TestIsPrimitive(t *testing.T) {
	tests := []struct {
		name string
		kind reflect.Kind
		want bool
	}{
		{"int", reflect.Int, true},
		{"int8", reflect.Int8, true},
		{"int16", reflect.Int16, true},
		{"int32", reflect.Int32, true},
		{"int64", reflect.Int64, true},
		{"uint", reflect.Uint, true},
		{"uint8", reflect.Uint8, true},
		{"uint16", reflect.Uint16, true},
		{"uint32", reflect.Uint32, true},
		{"uint64", reflect.Uint64, true},
		{"float32", reflect.Float32, true},
		{"float64", reflect.Float64, true},
		{"bool", reflect.Bool, true},
		{"string", reflect.String, true},
		{"struct", reflect.Struct, false},
		{"slice", reflect.Slice, false},
		{"map", reflect.Map, false},
		{"pointer", reflect.Pointer, false},
		{"interface", reflect.Interface, false},
		{"chan", reflect.Chan, false},
		{"func", reflect.Func, false},
		{"array", reflect.Array, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reflectutils.IsPrimitive(tt.kind)
			require.Equal(t, tt.want, got)
		})
	}
}

type label string

func TestConvertValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		src    any
		dst    reflect.Type
		want   any
		wantOK bool
	}{
		{"int to string is decimal", 65, reflect.TypeFor[string](), "65", true},
		{"negative int8 to string", int8(-7), reflect.TypeFor[string](), "-7", true},
		{"uint to named string", uint32(1000), reflect.TypeFor[label](), label("1000"), true},
		{"int widening", int32(5), reflect.TypeFor[int64](), int64(5), true},
		{"string to bytes", "héllo", reflect.TypeFor[[]byte](), []byte("héllo"), true},
		{"string to runes", "héllo", reflect.TypeFor[[]rune](), []rune("héllo"), true},
		{"bytes to string", []byte("héllo"), reflect.TypeFor[string](), "héllo", true},
		{"runes to string", []rune("héllo"), reflect.TypeFor[string](), "héllo", true},
		{"slice to array of matching length", []int{1, 2}, reflect.TypeFor[[2]int](), [2]int{1, 2}, true},
		{"short slice to array is refused", []int{1}, reflect.TypeFor[[2]int](), nil, false},
		{"nil slice to array is refused", []int(nil), reflect.TypeFor[[2]int](), nil, false},
		{"inconvertible types", "x", reflect.TypeFor[int](), nil, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, ok := reflectutils.ConvertValue(reflect.ValueOf(tc.src), tc.dst)
			require.Equal(t, tc.wantOK, ok)

			if tc.wantOK {
				require.Equal(t, tc.want, got.Interface())
			}
		})
	}
}

func TestConvertValueRoundsIntegersToFloat32Once(t *testing.T) {
	t.Parallel()

	for _, v := range []any{int64(1<<62 + 1<<38 + 1), uint64(1<<62 + 1<<38 + 1)} {
		got, ok := reflectutils.ConvertValue(reflect.ValueOf(v), reflect.TypeFor[float32]())
		if !ok {
			t.Fatalf("%T not converted", v)
		}

		want := float32(int64(1<<62 + 1<<38 + 1))
		if got.Interface() != want {
			t.Fatalf("%T: got %v, want %v", v, got.Interface(), want)
		}
	}
}
