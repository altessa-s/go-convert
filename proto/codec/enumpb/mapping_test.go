// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package enumpb_test

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-convert/proto/codec/enumpb"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/typepb"

	converter "github.com/altessa-s/go-convert"
)

// cardinality is a hand-written Go enum mirrored by typepb.Field_Cardinality.
type cardinality uint8

const (
	cardinalityNone cardinality = iota
	cardinalityOptional
	cardinalityRequired
	cardinalityRepeated
	cardinalityMissing
)

var cardinalities = map[typepb.Field_Cardinality]cardinality{
	typepb.Field_CARDINALITY_OPTIONAL: cardinalityOptional,
	typepb.Field_CARDINALITY_REQUIRED: cardinalityRequired,
	typepb.Field_CARDINALITY_REPEATED: cardinalityRepeated,
}

func TestNewMapping_Conversions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		opts []enumpb.Option
		src  any
		dst  any
		want any
	}{
		{name: "enum to Go", src: typepb.Field_CARDINALITY_REQUIRED, dst: new(cardinality), want: cardinalityRequired},
		{name: "Go to enum", src: cardinalityRepeated, dst: new(typepb.Field_Cardinality), want: typepb.Field_CARDINALITY_REPEATED},
		{name: "enum pointer to Go pointer", src: typepb.Field_CARDINALITY_OPTIONAL.Enum(), dst: new(*cardinality), want: new(cardinalityOptional)},
		{name: "Go pointer to enum pointer", src: new(cardinalityOptional), dst: new(*typepb.Field_Cardinality), want: typepb.Field_CARDINALITY_OPTIONAL.Enum()},
		{name: "unspecified to zero Go value", opts: []enumpb.Option{enumpb.WithUnspecifiedAsEmpty()},
			src: typepb.Field_CARDINALITY_UNKNOWN, dst: new(cardinality), want: cardinalityNone},
		{name: "unspecified to nil Go pointer", opts: []enumpb.Option{enumpb.WithUnspecifiedAsEmpty()},
			src: typepb.Field_CARDINALITY_UNKNOWN, dst: new(new(cardinalityRequired)), want: (*cardinality)(nil)},
		{name: "zero Go value to unspecified", opts: []enumpb.Option{enumpb.WithUnspecifiedAsEmpty()},
			src: cardinalityNone, dst: new(typepb.Field_CARDINALITY_REPEATED), want: typepb.Field_CARDINALITY_UNKNOWN},
		{name: "zero Go value to nil enum pointer", opts: []enumpb.Option{enumpb.WithUnspecifiedAsEmpty()},
			src: cardinalityNone, dst: new(typepb.Field_CARDINALITY_REPEATED.Enum()), want: (*typepb.Field_Cardinality)(nil)},
		{name: "nil enum pointer zeroes the destination", src: (*typepb.Field_Cardinality)(nil), dst: new(cardinalityRequired), want: cardinalityNone},
		{name: "nil enum pointer clears a Go pointer", src: (*typepb.Field_Cardinality)(nil), dst: new(new(cardinalityRequired)), want: (*cardinality)(nil)},
		{name: "nil Go pointer zeroes the destination", src: (*cardinality)(nil),
			dst: new(typepb.Field_CARDINALITY_REQUIRED), want: typepb.Field_CARDINALITY_UNKNOWN},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dst := reflect.ValueOf(tt.dst).Elem()
			err := enumpb.NewMapping(cardinalities, tt.opts...)("f", reflect.ValueOf(tt.src), dst, failNext(t))
			require.NoError(t, err)
			require.Equal(t, tt.want, dst.Interface())
		})
	}
}

func TestNewMapping_TableTakesPrecedence(t *testing.T) {
	t.Parallel()

	table := map[typepb.Field_Cardinality]cardinality{
		typepb.Field_CARDINALITY_UNKNOWN:  cardinalityMissing,
		typepb.Field_CARDINALITY_OPTIONAL: cardinalityNone,
	}
	codec := enumpb.NewMapping(table, enumpb.WithUnspecifiedAsEmpty())

	var goValue cardinality
	require.NoError(t, codec("f", reflect.ValueOf(typepb.Field_CARDINALITY_UNKNOWN), reflect.ValueOf(&goValue).Elem(), failNext(t)))
	require.Equal(t, cardinalityMissing, goValue)

	var enum typepb.Field_Cardinality
	require.NoError(t, codec("f", reflect.ValueOf(cardinalityNone), reflect.ValueOf(&enum).Elem(), failNext(t)))
	require.Equal(t, typepb.Field_CARDINALITY_OPTIONAL, enum)
}

func TestNewMapping_UnknownValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		src  any
		dst  any
		want enumpb.UnknownValueError
	}{
		{name: "enum outside the table", src: typepb.Field_CARDINALITY_UNKNOWN, dst: new(cardinality),
			want: enumpb.UnknownValueError{Enum: "google.protobuf.Field.Cardinality", Path: "x.y", Value: protoreflect.EnumNumber(0)}},
		{name: "undeclared enum number", src: typepb.Field_Cardinality(9), dst: new(cardinality),
			want: enumpb.UnknownValueError{Enum: "google.protobuf.Field.Cardinality", Path: "x.y", Value: protoreflect.EnumNumber(9)}},
		{name: "Go value outside the table", src: cardinalityMissing, dst: new(typepb.Field_Cardinality),
			want: enumpb.UnknownValueError{Enum: "google.protobuf.Field.Cardinality", Path: "x.y", Value: cardinalityMissing}},
		{name: "zero Go value without the option", src: cardinalityNone, dst: new(typepb.Field_Cardinality),
			want: enumpb.UnknownValueError{Enum: "google.protobuf.Field.Cardinality", Path: "x.y", Value: cardinalityNone}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := enumpb.NewMapping(cardinalities)("x.y", reflect.ValueOf(tt.src), reflect.ValueOf(tt.dst).Elem(), failNext(t))
			require.Equal(t, error(tt.want), err)
		})
	}
}

func TestNewMapping_Delegates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		src  any
		dst  any
	}{
		{name: "other enum", src: typepb.Field_TYPE_BOOL, dst: new(cardinality)},
		{name: "Go value to string", src: cardinalityOptional, dst: new(string)},
		{name: "enum to string", src: typepb.Field_CARDINALITY_OPTIONAL, dst: new(string)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			called := false
			err := enumpb.NewMapping(cardinalities)("f", reflect.ValueOf(tt.src), reflect.ValueOf(tt.dst).Elem(), func(string, reflect.Value, reflect.Value) error {
				called = true
				return nil
			})
			require.NoError(t, err)
			require.True(t, called)
		})
	}
}

func TestNewMapping_PanicsOnDuplicateGoValue(t *testing.T) {
	t.Parallel()

	table := map[typepb.Field_Cardinality]cardinality{
		typepb.Field_CARDINALITY_OPTIONAL: cardinalityOptional,
		typepb.Field_CARDINALITY_REQUIRED: cardinalityOptional,
	}

	require.Panics(t, func() { enumpb.NewMapping(table) })
}

type columnDTO struct {
	Name        string
	Cardinality cardinality
}

func TestConverter_Mapping(t *testing.T) {
	t.Parallel()

	opts := []converter.Option{converter.WithErrorCodecs(enumpb.NewMapping(cardinalities))}

	var dto columnDTO

	_, err := converter.ConvertE(&typepb.Field{Name: "tags", Cardinality: typepb.Field_CARDINALITY_REPEATED}, &dto, opts...)
	require.NoError(t, err)
	require.Equal(t, columnDTO{Name: "tags", Cardinality: cardinalityRepeated}, dto)

	var field typepb.Field

	_, err = converter.ConvertE(&columnDTO{Name: "id", Cardinality: cardinalityMissing}, &field, opts...)
	require.Equal(t, error(enumpb.UnknownValueError{Enum: "google.protobuf.Field.Cardinality", Path: "cardinality", Value: cardinalityMissing}), err)
}
