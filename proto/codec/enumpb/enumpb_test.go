// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package enumpb_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-convert/proto/codec/enumpb"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/typepb"

	converter "github.com/altessa-s/go-convert"
)

// label is a named string type, to check that any string kind is supported.
type label string

func failNext(t *testing.T) func(string, reflect.Value, reflect.Value) error {
	t.Helper()

	return func(string, reflect.Value, reflect.Value) error {
		require.Fail(t, "next should not be called")
		return nil
	}
}

func TestNew_EnumToString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		opts []enumpb.Option
		src  any
		want string
	}{
		{name: "declared name", src: typepb.Field_CARDINALITY_REPEATED, want: "CARDINALITY_REPEATED"},
		{name: "zero is a declared name", src: typepb.Field_CARDINALITY_UNKNOWN, want: "CARDINALITY_UNKNOWN"},
		{name: "explicit prefix", opts: []enumpb.Option{enumpb.WithTrimPrefix("CARDINALITY_")}, src: typepb.Field_CARDINALITY_OPTIONAL, want: "OPTIONAL"},
		{name: "enum prefix", opts: []enumpb.Option{enumpb.WithTrimEnumPrefix()}, src: typepb.Field_CARDINALITY_REQUIRED, want: "REQUIRED"},
		{name: "enum prefix of a nested enum", opts: []enumpb.Option{enumpb.WithTrimEnumPrefix()}, src: descriptorpb.FeatureSet_LEGACY_REQUIRED, want: "LEGACY_REQUIRED"},
		{name: "enum prefix absent keeps the name", opts: []enumpb.Option{enumpb.WithTrimEnumPrefix()}, src: typepb.Field_TYPE_STRING, want: "TYPE_STRING"},
		{name: "enum prefix wins over explicit prefix",
			opts: []enumpb.Option{enumpb.WithTrimPrefix("CARD"), enumpb.WithTrimEnumPrefix()}, src: typepb.Field_CARDINALITY_REQUIRED, want: "REQUIRED"},
		{name: "lower case", opts: []enumpb.Option{enumpb.WithTrimPrefix("TYPE_"), enumpb.WithCase(enumpb.CaseLower)}, src: typepb.Field_TYPE_SFIXED64, want: "sfixed64"},
		{name: "upper case", opts: []enumpb.Option{enumpb.WithCase(enumpb.CaseUpper)}, src: typepb.Field_TYPE_BOOL, want: "TYPE_BOOL"},
		{name: "unspecified as empty", opts: []enumpb.Option{enumpb.WithUnspecifiedAsEmpty()}, src: typepb.Field_CARDINALITY_UNKNOWN, want: ""},
		{name: "unspecified as empty keeps other values", opts: []enumpb.Option{enumpb.WithUnspecifiedAsEmpty()}, src: typepb.Field_TYPE_INT32, want: "TYPE_INT32"},
		{name: "pointer source", src: new(descriptorpb.FieldDescriptorProto_TYPE_BYTES), want: "TYPE_BYTES"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dst := "unchanged"
			err := enumpb.New(tt.opts...)("Field.Kind", reflect.ValueOf(tt.src), reflect.ValueOf(&dst).Elem(), failNext(t))
			require.NoError(t, err)
			require.Equal(t, tt.want, dst)
		})
	}
}

func TestNew_StringToEnum(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		opts []enumpb.Option
		src  string
		want typepb.Field_Cardinality
	}{
		{name: "declared name", src: "CARDINALITY_REPEATED", want: typepb.Field_CARDINALITY_REPEATED},
		{name: "explicit prefix", opts: []enumpb.Option{enumpb.WithTrimPrefix("CARDINALITY_")}, src: "OPTIONAL", want: typepb.Field_CARDINALITY_OPTIONAL},
		{name: "enum prefix", opts: []enumpb.Option{enumpb.WithTrimEnumPrefix()}, src: "REQUIRED", want: typepb.Field_CARDINALITY_REQUIRED},
		{name: "lower case parses any case",
			opts: []enumpb.Option{enumpb.WithTrimEnumPrefix(), enumpb.WithCase(enumpb.CaseLower)}, src: "Repeated", want: typepb.Field_CARDINALITY_REPEATED},
		{name: "upper case parses any case", opts: []enumpb.Option{enumpb.WithCase(enumpb.CaseUpper)}, src: "cardinality_optional", want: typepb.Field_CARDINALITY_OPTIONAL},
		{name: "empty as unspecified", opts: []enumpb.Option{enumpb.WithUnspecifiedAsEmpty()}, src: "", want: typepb.Field_CARDINALITY_UNKNOWN},
		{name: "unspecified name still parses", opts: []enumpb.Option{enumpb.WithUnspecifiedAsEmpty()}, src: "CARDINALITY_UNKNOWN", want: typepb.Field_CARDINALITY_UNKNOWN},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dst := typepb.Field_CARDINALITY_REPEATED + 100
			err := enumpb.New(tt.opts...)("Field.Cardinality", reflect.ValueOf(tt.src), reflect.ValueOf(&dst).Elem(), failNext(t))
			require.NoError(t, err)
			require.Equal(t, tt.want, dst)
		})
	}
}

func TestNew_Pointers(t *testing.T) {
	t.Parallel()

	codec := enumpb.New(enumpb.WithTrimEnumPrefix(), enumpb.WithCase(enumpb.CaseLower), enumpb.WithUnspecifiedAsEmpty())

	t.Run("enum to string pointer", func(t *testing.T) {
		t.Parallel()

		var dst *label
		require.NoError(t, codec("f", reflect.ValueOf(descriptorpb.FieldDescriptorProto_TYPE_GROUP), reflect.ValueOf(&dst).Elem(), failNext(t)))
		require.NotNil(t, dst)
		require.Equal(t, label("group"), *dst)
	})
	t.Run("unspecified to nil string pointer", func(t *testing.T) {
		t.Parallel()

		dst := new(label("stale"))
		require.NoError(t, codec("f", reflect.ValueOf(typepb.Field_CARDINALITY_UNKNOWN), reflect.ValueOf(&dst).Elem(), failNext(t)))
		require.Nil(t, dst)
	})
	t.Run("string to enum pointer", func(t *testing.T) {
		t.Parallel()

		var dst *descriptorpb.FieldDescriptorProto_Label
		require.NoError(t, codec("f", reflect.ValueOf(new("repeated")), reflect.ValueOf(&dst).Elem(), failNext(t)))
		require.NotNil(t, dst)
		require.Equal(t, descriptorpb.FieldDescriptorProto_LABEL_REPEATED, *dst)
	})
	t.Run("empty string to nil enum pointer", func(t *testing.T) {
		t.Parallel()

		dst := descriptorpb.FeatureSet_EXPLICIT.Enum()
		require.NoError(t, codec("f", reflect.ValueOf(""), reflect.ValueOf(&dst).Elem(), failNext(t)))
		require.Nil(t, dst)
	})
}

func TestNew_NilSourceZeroesDestination(t *testing.T) {
	t.Parallel()

	codec := enumpb.New(enumpb.WithUnspecifiedAsEmpty())

	var enumSrc *descriptorpb.FieldDescriptorProto_Type

	dst := "stale"
	require.NoError(t, codec("f", reflect.ValueOf(enumSrc), reflect.ValueOf(&dst).Elem(), failNext(t)))
	require.Empty(t, dst)

	ptrDst := new("stale")
	require.NoError(t, codec("f", reflect.ValueOf(enumSrc), reflect.ValueOf(&ptrDst).Elem(), failNext(t)))
	require.Nil(t, ptrDst)

	var stringSrc *string

	enumDst := typepb.Field_TYPE_BOOL
	require.NoError(t, codec("f", reflect.ValueOf(stringSrc), reflect.ValueOf(&enumDst).Elem(), failNext(t)))
	require.Equal(t, typepb.Field_TYPE_UNKNOWN, enumDst)

	enumPtrDst := typepb.Field_TYPE_BOOL.Enum()
	require.NoError(t, codec("f", reflect.ValueOf(stringSrc), reflect.ValueOf(&enumPtrDst).Elem(), failNext(t)))
	require.Nil(t, enumPtrDst)
}

func TestNew_InvalidCasePanics(t *testing.T) {
	t.Parallel()

	for _, c := range []enumpb.Case{enumpb.CaseUpper + 1, -1} {
		require.PanicsWithValue(t, fmt.Sprintf("enumpb: New: invalid Case %d; use CaseKeep, CaseLower or CaseUpper", int(c)), func() {
			enumpb.New(enumpb.WithCase(c))
		})
		require.Panics(t, func() {
			enumpb.NewMapping(map[typepb.Field_Cardinality]string{}, enumpb.WithCase(c))
		})
	}

	require.NotPanics(t, func() { enumpb.New(enumpb.WithCase(enumpb.CaseKeep)) })
}

func TestNew_UnknownValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		opts    []enumpb.Option
		src     any
		dst     any
		want    enumpb.UnknownValueError
		message string
	}{
		{
			name:    "unknown number",
			src:     typepb.Field_Cardinality(42),
			dst:     new(string),
			want:    enumpb.UnknownValueError{Enum: "google.protobuf.Field.Cardinality", Path: "a.b", Value: protoreflect.EnumNumber(42)},
			message: `enumpb: field "a.b": google.protobuf.Field.Cardinality has no value for 42`,
		},
		{
			name: "zero number without a zero value",
			src:  descriptorpb.FieldDescriptorProto_Type(0),
			dst:  new(string),
			want: enumpb.UnknownValueError{Enum: "google.protobuf.FieldDescriptorProto.Type", Path: "a.b", Value: protoreflect.EnumNumber(0)},
		},
		{
			name:    "unknown name",
			src:     "CARDINALITY_SOMETIMES",
			dst:     new(typepb.Field_Cardinality),
			want:    enumpb.UnknownValueError{Enum: "google.protobuf.Field.Cardinality", Path: "a.b", Value: "CARDINALITY_SOMETIMES"},
			message: `enumpb: field "a.b": google.protobuf.Field.Cardinality has no value for "CARDINALITY_SOMETIMES"`,
		},
		{
			name: "untrimmed name with a trimmed prefix",
			opts: []enumpb.Option{enumpb.WithTrimEnumPrefix()},
			src:  "CARDINALITY_OPTIONAL",
			dst:  new(typepb.Field_Cardinality),
			want: enumpb.UnknownValueError{Enum: "google.protobuf.Field.Cardinality", Path: "a.b", Value: "CARDINALITY_OPTIONAL"},
		},
		{
			name: "case-sensitive by default",
			src:  "cardinality_optional",
			dst:  new(typepb.Field_Cardinality),
			want: enumpb.UnknownValueError{Enum: "google.protobuf.Field.Cardinality", Path: "a.b", Value: "cardinality_optional"},
		},
		{
			name: "empty name without the option",
			src:  "",
			dst:  new(typepb.Field_Cardinality),
			want: enumpb.UnknownValueError{Enum: "google.protobuf.Field.Cardinality", Path: "a.b", Value: ""},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := enumpb.New(tt.opts...)("a.b", reflect.ValueOf(tt.src), reflect.ValueOf(tt.dst).Elem(), failNext(t))

			var unknown enumpb.UnknownValueError
			require.ErrorAs(t, err, &unknown)
			require.Equal(t, tt.want, unknown)

			if tt.message != "" {
				require.EqualError(t, err, tt.message)
			}
		})
	}
}

func TestNew_Delegates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		src  any
		dst  any
	}{
		{name: "string to string", src: "a", dst: new(string)},
		{name: "plain int32 to string", src: int32(1), dst: new(string)},
		{name: "string to plain int32", src: "1", dst: new(int32)},
		{name: "enum to enum", src: typepb.Field_TYPE_BOOL, dst: new(typepb.Field_Kind)},
		{name: "enum to int", src: typepb.Field_TYPE_BOOL, dst: new(int)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			called := false
			err := enumpb.New()("f", reflect.ValueOf(tt.src), reflect.ValueOf(tt.dst).Elem(), func(string, reflect.Value, reflect.Value) error {
				called = true
				return nil
			})
			require.NoError(t, err)
			require.True(t, called)
		})
	}
}

type fieldDTO struct {
	Name        string
	Kind        string
	Cardinality *string
}

func TestConverter_BothDirections(t *testing.T) {
	t.Parallel()

	opts := []converter.Option{converter.WithErrorCodecs(enumpb.New(
		enumpb.WithTrimEnumPrefix(), enumpb.WithCase(enumpb.CaseLower), enumpb.WithUnspecifiedAsEmpty(),
	))}

	field := &typepb.Field{Name: "id", Kind: typepb.Field_TYPE_INT64, Cardinality: typepb.Field_CARDINALITY_UNKNOWN}

	var dto fieldDTO

	_, err := converter.ConvertE(field, &dto, opts...)
	require.NoError(t, err)
	require.Equal(t, fieldDTO{Name: "id", Kind: "type_int64"}, dto)

	dto.Cardinality = new("repeated")

	var back typepb.Field

	_, err = converter.ConvertE(&dto, &back, opts...)
	require.NoError(t, err)
	require.True(t, proto.Equal(&typepb.Field{Name: "id", Kind: typepb.Field_TYPE_INT64, Cardinality: typepb.Field_CARDINALITY_REPEATED}, &back))
}

func TestConverter_ErrorReturnedUnchanged(t *testing.T) {
	t.Parallel()

	dto := fieldDTO{Name: "id", Kind: "type_decimal"}

	var field typepb.Field

	_, err := converter.ConvertE(&dto, &field, converter.WithErrorCodecs(enumpb.New(enumpb.WithCase(enumpb.CaseLower))))

	var unknown enumpb.UnknownValueError
	require.ErrorAs(t, err, &unknown)
	require.Equal(t, enumpb.UnknownValueError{Enum: "google.protobuf.Field.Kind", Path: "kind", Value: "type_decimal"}, unknown)
	require.IsType(t, enumpb.UnknownValueError{}, err, "the codec error must reach the caller unwrapped")
}
