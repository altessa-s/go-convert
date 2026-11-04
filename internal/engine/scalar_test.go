// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package engine

import (
	"math"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	reflectutils "github.com/altessa-s/go-convert/internal/reflect"
)

type (
	namedString string
	namedBool   bool
	namedFloat  float32
)

// scalarSamples holds edge values of every scalar kind.
var scalarSamples = []any{
	int(math.MinInt64), int8(-128), int16(-1), int32(math.MaxInt32), int64(1<<60 + 1<<36 + 1),
	uint(math.MaxUint64), uint8(255), uint16(65535), uint32(1 << 31), uint64(1<<63 + 1), uintptr(7),
	float32(-1.5), float32(math.MaxFloat32), float64(1e20), float64(-3.99), math.NaN(), math.Inf(-1), float64(1 << 63),
	true, "text", namedString("named"), namedBool(true), namedFloat(2.5),
}

// scalarTargets holds one zero value of every scalar kind.
var scalarTargets = []any{
	int(0), int8(0), int16(0), int32(0), int64(0), uint(0), uint8(0), uint16(0), uint32(0), uint64(0), uintptr(0),
	float32(0), float64(0), false, "", namedString(""), namedBool(false), namedFloat(0),
}

func TestSetScalarMatchesConvert(t *testing.T) {
	t.Parallel()

	for _, sample := range scalarSamples {
		src := reflect.ValueOf(sample)

		for _, target := range scalarTargets {
			dstType := reflect.TypeOf(target)
			if !isScalarPair(src.Kind(), dstType.Kind()) {
				continue
			}

			require.True(t, src.CanConvert(dstType), "%T -> %s", sample, dstType)

			got := reflect.New(dstType).Elem()
			setScalar(src, got)

			want, ok := reflectutils.ConvertValue(src, dstType)
			require.True(t, ok, "%T -> %s", sample, dstType)

			if want.Kind() == reflect.Float32 || want.Kind() == reflect.Float64 {
				if math.IsNaN(want.Float()) {
					require.True(t, math.IsNaN(got.Float()), "%T -> %s", sample, dstType)
					continue
				}
			}

			require.Equal(t, want.Interface(), got.Interface(), "%v (%T) -> %s", sample, sample, dstType)
		}
	}
}

func TestIsScalarPair(t *testing.T) {
	t.Parallel()

	require.True(t, isScalarPair(reflect.Int, reflect.Float32))
	require.True(t, isScalarPair(reflect.Float64, reflect.Float32))
	require.True(t, isScalarPair(reflect.String, reflect.String))
	require.False(t, isScalarPair(reflect.Int, reflect.String), "integers are formatted, not converted")
	require.False(t, isScalarPair(reflect.Float32, reflect.Float32), "float32 copies keep NaN payloads")
	require.False(t, isScalarPair(reflect.Bool, reflect.Int))
	require.False(t, isScalarPair(reflect.String, reflect.Slice))
	require.False(t, isScalarPair(reflect.Struct, reflect.Struct))
}

func TestConvertConvertibleKeepsSetFailures(t *testing.T) {
	t.Parallel()

	type hidden struct{ v int8 }

	w := New(Config{}).walker()
	defer w.release()

	var out int64

	field := reflect.ValueOf(hidden{v: 3}).Field(0)

	require.PanicsWithValue(t, "reflect: reflect.Value.Set using unaddressable value", func() {
		w.convertConvertible("v", field, reflect.ValueOf(out))
	}, "an unsettable destination fails as before")
	require.PanicsWithValue(t, "reflect: reflect.Value.Set using value obtained using unexported field", func() {
		w.convertConvertible("v", field, reflect.ValueOf(&out).Elem())
	}, "an unexported source fails as before")
}

func TestPlannedPrimitiveKeepsPathSemantics(t *testing.T) {
	t.Parallel()

	type wide struct{ V int64 }

	type narrow struct{ V float32 }

	x := int64(1<<60 + 1<<36 + 1)
	e := New(Config{})

	var byValue, byPointer narrow
	require.NoError(t, e.Convert(wide{V: x}, &byValue))
	require.NoError(t, e.Convert(&wide{V: x}, &byPointer))

	require.Equal(t, float32(x), byValue.V, "an unaddressable source rounds once")
	require.Equal(t, float32(x), byPointer.V, "an addressable source rounds once")
}

func TestPrimitiveSliceElements(t *testing.T) {
	t.Parallel()

	type wide struct{ V []int64 }

	type narrow struct{ V []int8 }

	w := New(Config{}).walker()
	defer w.release()

	require.NotNil(t, w.primitiveElemFunc(reflect.TypeFor[int64](), reflect.TypeFor[int8]()))
	require.Nil(t, w.primitiveElemFunc(reflect.TypeFor[int64](), reflect.TypeFor[int64]()), "assignable elements")
	require.Nil(t, w.primitiveElemFunc(reflect.TypeFor[int](), reflect.TypeFor[uint]()), "no table entry")
	require.Nil(t, w.primitiveElemFunc(reflect.TypeFor[*int](), reflect.TypeFor[*int8]()), "pointers")

	var out narrow
	require.NoError(t, New(Config{}).Convert(&wide{V: []int64{1, 300, -2}}, &out))
	require.Equal(t, []int8{1, 44, -2}, out.V)

	out = narrow{}
	require.NoError(t, New(Config{IgnoreZeroValues: true}).Convert(&wide{V: []int64{0, 5}}, &out))
	require.Equal(t, []int8{0, 5}, out.V, "skipped zero elements stay zero")

	out = narrow{}
	err := New(Config{OverflowCheck: true}).Convert(&wide{V: []int64{1, 300}}, &out)
	require.Equal(t, OverflowError{Field: "v.1", From: reflect.Int64, To: reflect.Int8, Value: int64(300)}, err)
	require.Equal(t, []int8{1}, out.V, "the converted prefix is kept")

	err = New(Config{MaxDepth: 1}).Convert(&wide{V: []int64{1}}, &out)
	require.Equal(t, DepthError{Limit: 1}, err)
}

func TestCachedTypeRelationsMatchReflect(t *testing.T) {
	t.Parallel()

	types := []reflect.Type{
		reflect.TypeFor[any](), reflect.TypeFor[error](), reflect.TypeFor[int](), reflect.TypeFor[int64](),
		reflect.TypeFor[float32](), reflect.TypeFor[namedFloat](), reflect.TypeFor[string](), reflect.TypeFor[namedString](),
		reflect.TypeFor[bool](), reflect.TypeFor[namedBool](), reflect.TypeFor[[]byte](), reflect.TypeFor[[]int](),
		reflect.TypeFor[[2]int](), reflect.TypeFor[*int](), reflect.TypeFor[map[string]int](), reflect.TypeFor[planInner](),
		reflect.TypeFor[struct{ X int }](), reflect.TypeFor[*planInner](), reflect.TypeFor[reflect.Kind](),
	}

	w := New(Config{}).walker()
	defer w.release()

	for _, src := range types {
		for _, dst := range types {
			require.Equal(t, src.AssignableTo(dst), w.isTypeCachedAssignable(src, dst), "%s -> %s", src, dst)
			require.Equal(t, src.ConvertibleTo(dst), w.isTypeCachedConvertible(src, dst), "%s -> %s", src, dst)
		}
	}
}
