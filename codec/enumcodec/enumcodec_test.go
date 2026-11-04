// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package enumcodec_test

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-convert/codec/enumcodec"

	converter "github.com/altessa-s/go-convert"
	convcodec "github.com/altessa-s/go-convert/codec"
)

type color int

const (
	colorUnspecified color = iota
	colorRed
	colorGreen
	colorBlue
)

type label string

var colorNames = map[color]string{colorRed: "red", colorGreen: "green", colorBlue: "blue"}

var colorNumbers = map[color]int64{colorRed: 10, colorGreen: 20, colorBlue: 300}

// run invokes codec on a source value and a pointer to a destination,
// failing the test when the codec delegates.
func run(t *testing.T, codec convcodec.CodecE, src, dstPtr any) error {
	t.Helper()

	return codec("Field", reflect.ValueOf(src), reflect.ValueOf(dstPtr).Elem(), func(string, reflect.Value, reflect.Value) error {
		require.Fail(t, "next should not be called")
		return nil
	})
}

func TestNew_Format(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		opts []enumcodec.Option
		src  any
		want string
		err  bool
	}{
		{name: "known", src: colorGreen, want: "green"},
		{name: "pointer source", src: new(colorBlue), want: "blue"},
		{name: "unknown", src: color(42), err: true},
		{name: "zero without option", src: colorUnspecified, err: true},
		{name: "zero with unspecified", opts: []enumcodec.Option{enumcodec.WithUnspecified()}, src: colorUnspecified, want: ""},
		{name: "unknown with unspecified", opts: []enumcodec.Option{enumcodec.WithUnspecified()}, src: color(42), err: true},
		{name: "unknown as zero", opts: []enumcodec.Option{enumcodec.WithUnknownAsZero()}, src: color(42), want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dst := "unchanged"

			err := run(t, enumcodec.MustNew(colorNames, tt.opts...), tt.src, &dst)
			if tt.err {
				var unknown enumcodec.UnknownValueError
				require.ErrorAs(t, err, &unknown)
				require.Equal(t, "Field", unknown.Path)
				require.Equal(t, reflect.TypeFor[color](), unknown.Enum)
				require.Equal(t, "unchanged", dst)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, dst)
		})
	}
}

func TestNew_Parse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		opts []enumcodec.Option
		src  any
		want color
		err  bool
	}{
		{name: "known", src: "green", want: colorGreen},
		{name: "pointer source", src: new("red"), want: colorRed},
		{name: "unknown", src: "purple", err: true},
		{name: "wrong case", src: "RED", err: true},
		{name: "case insensitive", opts: []enumcodec.Option{enumcodec.WithCaseInsensitive()}, src: "ReD", want: colorRed},
		{name: "empty without option", src: "", err: true},
		{name: "empty with unspecified", opts: []enumcodec.Option{enumcodec.WithUnspecified()}, src: "", want: colorUnspecified},
		{name: "unknown as zero", opts: []enumcodec.Option{enumcodec.WithUnknownAsZero()}, src: "purple", want: colorUnspecified},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dst := colorBlue

			err := run(t, enumcodec.MustNew(colorNames, tt.opts...), tt.src, &dst)
			if tt.err {
				var unknown enumcodec.UnknownValueError
				require.ErrorAs(t, err, &unknown)
				require.Equal(t, reflect.Indirect(reflect.ValueOf(tt.src)).Interface(), unknown.Value)
				require.Equal(t, colorBlue, dst)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, dst)
		})
	}
}

func TestNew_Pointers(t *testing.T) {
	t.Parallel()

	codec := enumcodec.MustNew(colorNames)

	var name *string
	require.NoError(t, run(t, codec, colorRed, &name))
	require.Equal(t, "red", *name)

	var value *color
	require.NoError(t, run(t, codec, new("green"), &value))
	require.Equal(t, colorGreen, *value)

	name = new("stale")
	require.NoError(t, run(t, codec, (*color)(nil), &name))
	require.Nil(t, name, "a nil source must clear a pointer destination")

	plain := colorBlue
	require.NoError(t, run(t, codec, (*string)(nil), &plain))
	require.Equal(t, colorUnspecified, plain, "a nil source must zero a value destination")
}

func TestNewInt(t *testing.T) {
	t.Parallel()

	codec := enumcodec.MustNewInt(colorNumbers)

	var i32 int32
	require.NoError(t, run(t, codec, colorGreen, &i32))
	require.Equal(t, int32(20), i32)

	var u16 uint16
	require.NoError(t, run(t, codec, colorBlue, &u16))
	require.Equal(t, uint16(300), u16)

	var c color
	require.NoError(t, run(t, codec, uint8(10), &c))
	require.Equal(t, colorRed, c)
	require.NoError(t, run(t, codec, new(int64(300)), &c))
	require.Equal(t, colorBlue, c)

	var unknown enumcodec.UnknownValueError
	require.ErrorAs(t, run(t, codec, 11, &c), &unknown)
	require.Equal(t, 11, unknown.Value)
	require.ErrorAs(t, run(t, codec, uint64(1<<63), &c), &unknown)

	var (
		u8       uint8
		rangeErr enumcodec.RangeError
	)
	require.ErrorAs(t, run(t, codec, colorBlue, &u8), &rangeErr)
	require.Equal(t, enumcodec.RangeError{Path: "Field", Value: 300, Kind: reflect.Uint8}, rangeErr)
	require.Zero(t, u8)

	unspecified := enumcodec.MustNewInt(colorNumbers, enumcodec.WithUnspecified())

	var n int
	require.NoError(t, run(t, unspecified, colorUnspecified, &n))
	require.Zero(t, n)

	c = colorRed
	require.NoError(t, run(t, unspecified, 0, &c))
	require.Equal(t, colorUnspecified, c)
}

// shade is a second int32 enum whose values differ from paint's table.
type shade int32

// paint is an int32 enum with a NewInt table.
type paint int32

type canvas struct {
	Paint paint
	Shade *shade
}

type canvasDTO struct {
	Paint shade
	Shade *paint
}

func TestNewInt_EnumPairDelegates(t *testing.T) {
	t.Parallel()

	codec := enumcodec.MustNewInt(map[paint]int64{1: 100, 2: 200})

	called := false
	err := codec("Field", reflect.ValueOf(paint(1)), reflect.ValueOf(new(shade)).Elem(), func(string, reflect.Value, reflect.Value) error {
		called = true
		return nil
	})
	require.NoError(t, err)
	require.True(t, called, "an enum to enum pair must be delegated")

	dto, err := converter.ConvertE(canvas{Paint: 2, Shade: new(shade(1))}, &canvasDTO{}, converter.WithErrorCodecs(codec))
	require.NoError(t, err)
	require.Equal(t, &canvasDTO{Paint: 2, Shade: new(paint(1))}, dto, "the engine converts enum pairs by value")
}

func TestNewInt_CaseInsensitive(t *testing.T) {
	t.Parallel()

	codec, err := enumcodec.NewInt(colorNumbers, enumcodec.WithCaseInsensitive())
	require.Nil(t, codec)

	var cfg enumcodec.ConfigError
	require.ErrorAs(t, err, &cfg)
	require.Equal(t, reflect.TypeFor[color](), cfg.Enum)
	require.Panics(t, func() { enumcodec.MustNewInt(colorNumbers, enumcodec.WithCaseInsensitive()) })
}

func TestNegativeIntoUnsigned(t *testing.T) {
	t.Parallel()

	codec := enumcodec.MustNewInt(map[color]int64{colorRed: -1})

	var (
		u        uint
		rangeErr enumcodec.RangeError
	)
	require.ErrorAs(t, run(t, codec, colorRed, &u), &rangeErr)
}

func TestDelegates(t *testing.T) {
	t.Parallel()

	type other int

	codec := enumcodec.MustNew(colorNames)

	tests := []struct {
		name     string
		src, dst any
	}{
		{name: "enum to enum", src: colorRed, dst: new(color)},
		{name: "plain string pair", src: "a", dst: new(string)},
		{name: "enum to integer", src: colorRed, dst: new(int)},
		{name: "other type to string", src: other(1), dst: new(string)},
		{name: "double pointer", src: new(new(colorRed)), dst: new(string)},
		{name: "string to other", src: "red", dst: new(other)},
		{name: "named string to enum", src: label("blue"), dst: new(color)},
		{name: "enum to named string", src: colorBlue, dst: new(label)},
		{name: "enum to named string pointer", src: colorBlue, dst: new(*label)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			called := false
			err := codec("Field", reflect.ValueOf(tt.src), reflect.ValueOf(tt.dst).Elem(), func(string, reflect.Value, reflect.Value) error {
				called = true
				return nil
			})
			require.NoError(t, err)
			require.True(t, called, "next should be called")
		})
	}
}

func TestStringKindEnum(t *testing.T) {
	t.Parallel()

	type status string

	codec := enumcodec.MustNew(map[status]string{"ACTIVE": "active"})

	var dst string
	require.NoError(t, run(t, codec, status("ACTIVE"), &dst))
	require.Equal(t, "active", dst)

	called := false

	var same status

	err := codec("Field", reflect.ValueOf(status("ACTIVE")), reflect.ValueOf(&same).Elem(), func(string, reflect.Value, reflect.Value) error {
		called = true
		return nil
	})
	require.NoError(t, err)
	require.True(t, called, "an E to E pair must be delegated")
}

func TestDuplicates(t *testing.T) {
	t.Parallel()

	var cfg enumcodec.ConfigError

	_, err := enumcodec.New(map[color]string{colorRed: "x", colorBlue: "x"})
	require.ErrorAs(t, err, &cfg)
	require.ErrorContains(t, err, "mapped more than once")

	_, err = enumcodec.NewInt(map[color]int64{colorRed: 1, colorBlue: 1})
	require.ErrorAs(t, err, &cfg)

	_, err = enumcodec.New(map[color]string{colorRed: "x", colorBlue: "X"}, enumcodec.WithCaseInsensitive())
	require.ErrorAs(t, err, &cfg)
	require.ErrorContains(t, err, "collide ignoring case")

	_, err = enumcodec.New(map[color]string{colorRed: "x", colorBlue: "X"})
	require.NoError(t, err)

	require.Panics(t, func() { enumcodec.MustNew(map[color]string{colorRed: "x", colorBlue: "x"}) })
	require.Panics(t, func() { enumcodec.MustNewInt(map[color]int64{colorRed: 1, colorBlue: 1}) })
}

func TestErrorMessages(t *testing.T) {
	t.Parallel()

	err := enumcodec.UnknownValueError{Path: "a.b", Enum: reflect.TypeFor[color](), Value: "x"}
	require.Equal(t, `enumcodec: field "a.b": unknown enumcodec_test.color value "x"`, err.Error())

	rangeErr := enumcodec.RangeError{Path: "a", Value: 300, Kind: reflect.Uint8}
	require.Equal(t, `enumcodec: field "a": value 300 overflows uint8`, rangeErr.Error())
}

type order struct {
	ID     int
	Color  color
	Shade  *color
	Number color
}

type orderDTO struct {
	ID     int
	Color  string
	Shade  *string
	Number int32
}

func TestConverterRoundTrip(t *testing.T) {
	t.Parallel()

	opts := []converter.Option{converter.WithErrorCodecs(
		enumcodec.MustNew(colorNames),
		enumcodec.MustNewInt(colorNumbers),
	)}
	// Color and Shade are claimed by the string codec, Number by the integer one.
	src := order{ID: 1, Color: colorGreen, Shade: new(colorBlue), Number: colorRed}

	dto, err := converter.ConvertE(src, &orderDTO{}, opts...)
	require.NoError(t, err)
	require.Equal(t, &orderDTO{ID: 1, Color: "green", Shade: new("blue"), Number: 10}, dto)

	back, err := converter.ConvertE(*dto, &order{}, opts...)
	require.NoError(t, err)
	require.Equal(t, &src, back)
}

func TestConverterReturnsErrorUnchanged(t *testing.T) {
	t.Parallel()

	_, err := converter.ConvertE(orderDTO{Color: "purple"}, &order{}, converter.WithErrorCodecs(enumcodec.MustNew(colorNames)))
	require.Equal(t, enumcodec.UnknownValueError{Path: "color", Enum: reflect.TypeFor[color](), Value: "purple"}, err)

	_, err = converter.ConvertE(order{Color: colorRed, Number: color(7)}, &orderDTO{}, converter.WithErrorCodecs(enumcodec.MustNewInt(colorNumbers)))
	require.Equal(t, enumcodec.UnknownValueError{Path: "number", Enum: reflect.TypeFor[color](), Value: color(7)}, err,
		"Color pairs color with string, which NewInt does not claim")
}
