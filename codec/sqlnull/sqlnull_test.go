// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package sqlnull_test

import (
	"database/sql"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-convert/codec/sqlnull"
	"github.com/altessa-s/go-convert/codec/unixtime"

	converter "github.com/altessa-s/go-convert"
	convcodec "github.com/altessa-s/go-convert/codec"
)

var errInner = errors.New("inner failure")

// run invokes codec on a source value and a pointer to a destination,
// failing the test when the codec delegates.
func run(t *testing.T, codec convcodec.CodecE, src, dstPtr any) error {
	t.Helper()

	return codec("Field", reflect.ValueOf(src), reflect.ValueOf(dstPtr).Elem(), func(string, reflect.Value, reflect.Value) error {
		require.Fail(t, "next should not be called")
		return nil
	})
}

func TestIsNullType(t *testing.T) {
	t.Parallel()

	type Null[T any] struct {
		V     T
		Valid bool
	}

	tests := []struct {
		typ  reflect.Type
		want bool
	}{
		{reflect.TypeFor[sql.Null[int]](), true},
		{reflect.TypeFor[sql.Null[time.Time]](), true},
		{reflect.TypeFor[sql.NullString](), true},
		{reflect.TypeFor[sql.NullInt64](), true},
		{reflect.TypeFor[sql.NullInt32](), true},
		{reflect.TypeFor[sql.NullInt16](), true},
		{reflect.TypeFor[sql.NullByte](), true},
		{reflect.TypeFor[sql.NullFloat64](), true},
		{reflect.TypeFor[sql.NullBool](), true},
		{reflect.TypeFor[sql.NullTime](), true},
		{reflect.TypeFor[*sql.NullString](), false},
		{reflect.TypeFor[Null[int]](), false},
		{reflect.TypeFor[sql.RawBytes](), false},
		{reflect.TypeFor[sql.TxOptions](), false},
		{reflect.TypeFor[string](), false},
	}
	for _, tt := range tests {
		t.Run(tt.typ.String(), func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, sqlnull.IsNullType(tt.typ))
		})
	}
}

func TestFromNull(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	tests := []struct {
		name   string
		src    any
		dst    any // pointer to a destination pre-filled with a stale value
		expect any
	}{
		{name: "NullString to string", src: sql.NullString{String: "a", Valid: true}, dst: new("stale"), expect: "a"},
		{name: "invalid NullString to string", src: sql.NullString{String: "a"}, dst: new("stale"), expect: ""},
		{name: "NullString to *string", src: sql.NullString{String: "a", Valid: true}, dst: new(new("stale")), expect: new("a")},
		{name: "invalid NullString to *string", src: sql.NullString{}, dst: new(new("stale")), expect: (*string)(nil)},
		{name: "NullInt64 to int64", src: sql.NullInt64{Int64: 7, Valid: true}, dst: new(int64(1)), expect: int64(7)},
		{name: "NullInt32 to *int32", src: sql.NullInt32{Int32: 7, Valid: true}, dst: new((*int32)(nil)), expect: new(int32(7))},
		{name: "NullInt16 to int16", src: sql.NullInt16{Int16: 7, Valid: true}, dst: new(int16(0)), expect: int16(7)},
		{name: "NullByte to byte", src: sql.NullByte{Byte: 7, Valid: true}, dst: new(byte(0)), expect: byte(7)},
		{name: "NullFloat64 to *float64", src: sql.NullFloat64{Float64: 1.5, Valid: true}, dst: new((*float64)(nil)), expect: new(1.5)},
		{name: "NullBool to bool", src: sql.NullBool{Bool: true, Valid: true}, dst: new(false), expect: true},
		{name: "NullTime to time", src: sql.NullTime{Time: now, Valid: true}, dst: new(time.Time{}), expect: now},
		{name: "invalid NullTime to *time", src: sql.NullTime{}, dst: new(new(now)), expect: (*time.Time)(nil)},
		{name: "Null[int] to int", src: sql.Null[int]{V: 3, Valid: true}, dst: new(0), expect: 3},
		{name: "invalid Null[int] to int", src: sql.Null[int]{V: 3}, dst: new(9), expect: 0},
		{name: "Null[[]byte] to *[]byte", src: sql.Null[[]byte]{V: []byte("x"), Valid: true}, dst: new((*[]byte)(nil)), expect: new([]byte("x"))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.NoError(t, run(t, sqlnull.New(), tt.src, tt.dst))
			require.Equal(t, tt.expect, reflect.ValueOf(tt.dst).Elem().Interface())
		})
	}
}

func TestToNull(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		opts   []sqlnull.Option
		src    any
		expect any
	}{
		{name: "string", src: "a", expect: sql.NullString{String: "a", Valid: true}},
		{name: "empty string", src: "", expect: sql.NullString{Valid: true}},
		{name: "empty string as null", opts: []sqlnull.Option{sqlnull.WithZeroAsNull()}, src: "", expect: sql.NullString{}},
		{name: "*string", src: new("a"), expect: sql.NullString{String: "a", Valid: true}},
		{name: "*string to empty", opts: []sqlnull.Option{sqlnull.WithZeroAsNull()}, src: new(""), expect: sql.NullString{Valid: true}},
		{name: "nil *string", src: (*string)(nil), expect: sql.NullString{}},
		{name: "int64", src: int64(4), expect: sql.NullInt64{Int64: 4, Valid: true}},
		{name: "*float64", src: new(2.5), expect: sql.NullFloat64{Float64: 2.5, Valid: true}},
		{name: "bool", src: false, expect: sql.NullBool{Valid: true}},
		{name: "int to Null[int]", src: 5, expect: sql.Null[int]{V: 5, Valid: true}},
		{name: "nil *int to Null[int]", src: (*int)(nil), expect: sql.Null[int]{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dst := reflect.New(reflect.TypeOf(tt.expect))
			dst.Elem().Field(1).SetBool(true) // stale Valid flag must be cleared
			require.NoError(t, run(t, sqlnull.New(tt.opts...), tt.src, dst.Interface()))
			require.Equal(t, tt.expect, dst.Elem().Interface())
		})
	}
}

// convertInt is a stand-in chain that converts between integer kinds; a nil
// pointer source leaves the zero destination.
func convertInt(_ string, src, dst reflect.Value) error {
	if v := reflect.Indirect(src); v.IsValid() {
		dst.SetInt(v.Int())
	}

	return nil
}

func TestZeroAsNullAppliesOnlyWithOption(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		opts   []sqlnull.Option
		src    any
		expect sql.NullInt64
	}{
		{name: "zero int", src: 0, expect: sql.NullInt64{Valid: true}},
		{name: "zero int as null", opts: []sqlnull.Option{sqlnull.WithZeroAsNull()}, src: 0, expect: sql.NullInt64{}},
		{name: "int", opts: []sqlnull.Option{sqlnull.WithZeroAsNull()}, src: 3, expect: sql.NullInt64{Int64: 3, Valid: true}},
		{name: "pointer to zero int", opts: []sqlnull.Option{sqlnull.WithZeroAsNull()}, src: new(0), expect: sql.NullInt64{Valid: true}},
		{name: "nil int pointer", src: (*int)(nil), expect: sql.NullInt64{}},
		{name: "zero NullInt32", src: sql.NullInt32{Valid: true}, expect: sql.NullInt64{Valid: true}},
		{name: "zero NullInt32 as null", opts: []sqlnull.Option{sqlnull.WithZeroAsNull()}, src: sql.NullInt32{Valid: true}, expect: sql.NullInt64{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dst := sql.NullInt64{Int64: 9, Valid: true}
			require.NoError(t, sqlnull.New(tt.opts...)("Field", reflect.ValueOf(tt.src), reflect.ValueOf(&dst).Elem(), convertInt))
			require.Equal(t, tt.expect, dst)
		})
	}
}

func TestNilInterfaceSourceIsNull(t *testing.T) {
	t.Parallel()

	var src struct{ V any }

	dst := sql.NullString{String: "stale", Valid: true}
	require.NoError(t, sqlnull.New()("Field", reflect.ValueOf(&src).Elem().Field(0), reflect.ValueOf(&dst).Elem(), func(string, reflect.Value, reflect.Value) error {
		require.Fail(t, "next should not be called")
		return nil
	}))
	require.Equal(t, sql.NullString{}, dst)

	src.V = (*int)(nil)
	stale := sql.NullInt64{Int64: 7, Valid: true}
	require.NoError(t, sqlnull.New()("Field", reflect.ValueOf(&src).Elem().Field(0), reflect.ValueOf(&stale).Elem(), func(string, reflect.Value, reflect.Value) error {
		require.Fail(t, "next should not be called")
		return nil
	}))
	require.Equal(t, sql.NullInt64{}, stale, "an interface holding a nil pointer is NULL")
}

type counter struct{ Hits int }

type counterRow struct{ Hits sql.NullInt64 }

func TestConverterZeroIntIsValid(t *testing.T) {
	t.Parallel()

	row, err := converter.ConvertE(counter{}, &counterRow{}, converter.WithErrorCodecs(sqlnull.New()))
	require.NoError(t, err)
	require.Equal(t, &counterRow{Hits: sql.NullInt64{Valid: true}}, row, "int(0) behaves as int64(0)")

	row, err = converter.ConvertE(counter{}, &counterRow{}, converter.WithErrorCodecs(sqlnull.New(sqlnull.WithZeroAsNull())))
	require.NoError(t, err)
	require.Equal(t, &counterRow{}, row)
}

func TestComposition(t *testing.T) {
	t.Parallel()

	codec := sqlnull.New()
	inner := func(_ string, src, dst reflect.Value) error {
		// Stand-in for a downstream codec: int64 seconds <-> time.Time.
		if at, ok := reflect.TypeAssert[time.Time](src); ok {
			dst.SetInt(at.Unix())
			return nil
		}

		switch {
		case src.Kind() == reflect.Pointer && src.IsNil():
			dst.SetZero()
		default:
			dst.Set(reflect.ValueOf(time.Unix(reflect.Indirect(src).Int(), 0).UTC()))
		}

		return nil
	}

	var seconds int64
	require.NoError(t, codec("At", reflect.ValueOf(sql.NullTime{Time: time.Unix(60, 0), Valid: true}), reflect.ValueOf(&seconds).Elem(), inner))
	require.Equal(t, int64(60), seconds)

	seconds = 9
	require.NoError(t, codec("At", reflect.ValueOf(sql.NullTime{}), reflect.ValueOf(&seconds).Elem(), inner))
	require.Zero(t, seconds, "an invalid source zeroes the destination without calling next")

	var at sql.NullTime
	require.NoError(t, codec("At", reflect.ValueOf(int64(60)), reflect.ValueOf(&at).Elem(), inner))
	require.Equal(t, sql.NullTime{Time: time.Unix(60, 0).UTC(), Valid: true}, at)

	require.NoError(t, codec("At", reflect.ValueOf((*int64)(nil)), reflect.ValueOf(&at).Elem(), inner))
	require.Equal(t, sql.NullTime{}, at, "a zero inner value maps to NULL")

	var wide sql.Null[int64]
	require.NoError(t, codec("N", reflect.ValueOf(sql.NullInt32{Int32: 3, Valid: true}), reflect.ValueOf(&wide).Elem(),
		func(_ string, src, dst reflect.Value) error {
			dst.SetInt(src.Int())
			return nil
		}))
	require.Equal(t, sql.Null[int64]{V: 3, Valid: true}, wide)

	wide.Valid = true
	require.NoError(t, run(t, codec, sql.NullInt32{}, &wide))
	require.Equal(t, sql.Null[int64]{}, wide)
}

func TestErrorsFromChain(t *testing.T) {
	t.Parallel()

	fail := func(string, reflect.Value, reflect.Value) error { return errInner }
	codec := sqlnull.New()

	var at sql.NullTime
	require.ErrorIs(t, codec("At", reflect.ValueOf(int64(1)), reflect.ValueOf(&at).Elem(), fail), errInner)
	require.Equal(t, sql.NullTime{}, at)

	var n int64
	require.ErrorIs(t, codec("At", reflect.ValueOf(sql.NullTime{Valid: true}), reflect.ValueOf(&n).Elem(), fail), errInner)

	var wide sql.Null[int64]
	require.ErrorIs(t, codec("N", reflect.ValueOf(sql.NullInt32{Valid: true}), reflect.ValueOf(&wide).Elem(), fail), errInner)
	require.Equal(t, sql.Null[int64]{}, wide)
}

func TestDelegates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		src, dst any
	}{
		{name: "plain pair", src: "a", dst: new(string)},
		{name: "same null type", src: sql.NullString{String: "a", Valid: true}, dst: new(sql.NullString)},
		{name: "pointer to null type", src: &sql.NullString{}, dst: new(string)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			called := false
			err := sqlnull.New()("Field", reflect.ValueOf(tt.src), reflect.ValueOf(tt.dst).Elem(), func(string, reflect.Value, reflect.Value) error {
				called = true
				return nil
			})
			require.NoError(t, err)
			require.True(t, called)
		})
	}
}

type row struct {
	ID        int64
	Name      sql.NullString
	Age       sql.Null[int]
	Score     sql.NullFloat64
	CreatedAt sql.NullTime
	DeletedAt sql.NullTime
}

type entity struct {
	ID        int64
	Name      *string
	Age       int
	Score     *float64
	CreatedAt int64
	DeletedAt *int64
}

func TestConverterRoundTrip(t *testing.T) {
	t.Parallel()

	opts := []converter.Option{
		converter.WithErrorCodecs(sqlnull.New()),
		converter.WithCodecs(unixtime.New()),
	}
	src := row{
		ID:        1,
		Name:      sql.NullString{String: "Ann", Valid: true},
		Age:       sql.Null[int]{V: 30, Valid: true},
		CreatedAt: sql.NullTime{Time: time.Unix(1700000000, 0), Valid: true},
	}

	got, err := converter.ConvertE(src, &entity{}, opts...)
	require.NoError(t, err)
	require.Equal(t, &entity{ID: 1, Name: new("Ann"), Age: 30, CreatedAt: 1700000000}, got)

	back, err := converter.ConvertE(*got, &row{}, opts...)
	require.NoError(t, err)
	require.Equal(t, src.Name, back.Name)
	require.Equal(t, src.Age, back.Age)
	require.Equal(t, sql.NullFloat64{}, back.Score)
	require.True(t, back.CreatedAt.Valid)
	require.True(t, src.CreatedAt.Time.Equal(back.CreatedAt.Time))
	require.Equal(t, sql.NullTime{}, back.DeletedAt)
}

func TestConverterReturnsErrorUnchanged(t *testing.T) {
	t.Parallel()

	failing := func(path string, src, dst reflect.Value, next convcodec.CodecHandlerE) error {
		if src.Type() == reflect.TypeFor[time.Time]() {
			return errInner
		}

		return next(path, src, dst)
	}
	_, err := converter.ConvertE(row{CreatedAt: sql.NullTime{Valid: true}}, &entity{}, converter.WithErrorCodecs(sqlnull.New(), failing))
	require.Equal(t, errInner, err)
}
