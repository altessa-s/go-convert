// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package jsonpb_test

import (
	"encoding/json"
	"math"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/altessa-s/go-convert/proto/codec/jsonpb"

	converter "github.com/altessa-s/go-convert"
	convcodec "github.com/altessa-s/go-convert/codec"
)

type status string

type domainDoc struct {
	Payload any
	Tags    []any
	Meta    map[string]any
	Raw     json.RawMessage
}

type protoDoc struct {
	Payload *structpb.Value
	Tags    *structpb.ListValue
	Meta    *structpb.Struct
	Raw     *structpb.Struct
}

func mustValue(t *testing.T, x any) *structpb.Value {
	t.Helper()

	v, err := structpb.NewValue(x)
	require.NoError(t, err)

	return v
}

func failNext(t *testing.T) convcodec.CodecHandlerE {
	t.Helper()

	return func(string, reflect.Value, reflect.Value) error {
		require.Fail(t, "next should not be called")
		return nil
	}
}

// runE runs the codec on src into a fresh destination of type D.
func runE[D any](t *testing.T, codec convcodec.CodecE, src any) (D, error) {
	t.Helper()

	var dst D

	err := codec("field", reflect.ValueOf(src), reflect.ValueOf(&dst).Elem(), failNext(t))

	return dst, err
}

func TestNewE_ToProto(t *testing.T) {
	t.Parallel()

	nested := map[string]any{"name": "ann", "tags": []any{"a", 1.5, nil, true}}
	for _, tc := range []struct {
		name string
		src  any
		want *structpb.Value
	}{
		{name: "bool", src: true, want: structpb.NewBoolValue(true)},
		{name: "float64", src: 1.5, want: structpb.NewNumberValue(1.5)},
		{name: "float32", src: float32(2.5), want: structpb.NewNumberValue(2.5)},
		{name: "int", src: -7, want: structpb.NewNumberValue(-7)},
		{name: "int8", src: int8(-8), want: structpb.NewNumberValue(-8)},
		{name: "uint64", src: uint64(9), want: structpb.NewNumberValue(9)},
		{name: "uint8", src: uint8(10), want: structpb.NewNumberValue(10)},
		{name: "string", src: "x", want: structpb.NewStringValue("x")},
		{name: "named string", src: status("on"), want: structpb.NewStringValue("on")},
		{name: "list", src: []any{"a", 1, nil}, want: mustValue(t, []any{"a", 1, nil})},
		{name: "map", src: nested, want: mustValue(t, nested)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := runE[*structpb.Value](t, jsonpb.NewE(), tc.src)
			require.NoError(t, err)
			require.True(t, proto.Equal(tc.want, got), "got %v", got)
		})
	}
}

func TestNewE_FromProto(t *testing.T) {
	t.Parallel()

	nested := map[string]any{"name": "ann", "tags": []any{"a", 1.5, nil, true}}
	for _, tc := range []struct {
		name string
		src  *structpb.Value
		want any
	}{
		{name: "null", src: structpb.NewNullValue(), want: nil},
		{name: "no kind", src: &structpb.Value{}, want: nil},
		{name: "bool", src: structpb.NewBoolValue(true), want: true},
		{name: "number", src: structpb.NewNumberValue(1.5), want: 1.5},
		{name: "string", src: structpb.NewStringValue("x"), want: "x"},
		{name: "list", src: mustValue(t, []any{"a", 1.0, nil}), want: []any{"a", 1.0, nil}},
		{name: "map", src: mustValue(t, nested), want: nested},
		{name: "empty list", src: structpb.NewListValue(nil), want: []any{}},
		{name: "empty map", src: structpb.NewStructValue(nil), want: map[string]any{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := runE[any](t, jsonpb.NewE(), tc.src)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestNewE_ListAndStruct(t *testing.T) {
	t.Parallel()

	codec := jsonpb.NewE()
	list := []any{"a", 2.0, map[string]any{"k": false}}
	m := map[string]any{"a": "b", "n": []any{1.0}}

	pbList, err := runE[*structpb.ListValue](t, codec, list)
	require.NoError(t, err)
	gotList, err := runE[[]any](t, codec, pbList)
	require.NoError(t, err)
	require.Equal(t, list, gotList)

	pbStruct, err := runE[*structpb.Struct](t, codec, m)
	require.NoError(t, err)
	gotMap, err := runE[map[string]any](t, codec, pbStruct)
	require.NoError(t, err)
	require.Equal(t, m, gotMap)

	// Struct and ListValue sources also fill an any destination.
	gotAny, err := runE[any](t, codec, pbStruct)
	require.NoError(t, err)
	require.Equal(t, m, gotAny)
	gotAny, err = runE[any](t, codec, pbList)
	require.NoError(t, err)
	require.Equal(t, list, gotAny)

	// Non-pointer destinations are supported too.
	var value structpb.ListValue
	require.NoError(t, codec("field", reflect.ValueOf(list), reflect.ValueOf(&value).Elem(), failNext(t)))
	require.True(t, proto.Equal(pbList, &value))
}

func TestNewE_AnyFieldsByDynamicValue(t *testing.T) {
	t.Parallel()

	codec := jsonpb.NewE()

	for _, tc := range []struct {
		name string
		src  any
		dst  any
		want proto.Message
	}{
		{name: "list", src: []any{"a"}, dst: new(*structpb.ListValue), want: mustValue(t, []any{"a"}).GetListValue()},
		{name: "struct", src: map[string]any{"a": 1.0}, dst: new(*structpb.Struct), want: mustValue(t, map[string]any{"a": 1.0}).GetStructValue()},
		{name: "value", src: "a", dst: new(*structpb.Value), want: structpb.NewStringValue("a")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			holder := struct{ V any }{V: tc.src}
			src := reflect.ValueOf(&holder).Elem().Field(0)
			dst := reflect.ValueOf(tc.dst).Elem()
			require.NoError(t, codec("v", src, dst, failNext(t)))
			got, ok := reflect.TypeAssert[proto.Message](dst)
			require.True(t, ok)
			require.True(t, proto.Equal(tc.want, got), "got %v", got)
		})
	}
}

func TestNewE_ConvertRoundTrip(t *testing.T) {
	t.Parallel()

	opts := converter.WithErrorCodecs(jsonpb.NewE())
	src := domainDoc{
		Payload: map[string]any{"id": 7.0, "ok": true},
		Tags:    []any{"a", nil, 2.5},
		Meta:    map[string]any{"owner": "ann", "nested": map[string]any{"x": []any{}}},
		Raw:     json.RawMessage(`{"k":"v"}`),
	}

	var message protoDoc

	_, err := converter.ConvertE(src, &message, opts)
	require.NoError(t, err)
	require.Equal(t, "ann", message.Meta.GetFields()["owner"].GetStringValue())
	require.Equal(t, 7.0, message.Payload.GetStructValue().GetFields()["id"].GetNumberValue())
	require.Equal(t, structpb.NullValue_NULL_VALUE, message.Tags.GetValues()[1].GetNullValue())
	require.Equal(t, "v", message.Raw.GetFields()["k"].GetStringValue())

	var back domainDoc

	_, err = converter.ConvertE(message, &back, opts)
	require.NoError(t, err)
	require.Equal(t, src.Payload, back.Payload)
	require.Equal(t, src.Tags, back.Tags)
	require.Equal(t, src.Meta, back.Meta)
	require.JSONEq(t, string(src.Raw), string(back.Raw))
}

func TestNewE_Nil(t *testing.T) {
	t.Parallel()

	prefilled := func() protoDoc {
		return protoDoc{
			Payload: structpb.NewStringValue("old"),
			Tags:    &structpb.ListValue{},
			Meta:    &structpb.Struct{},
			Raw:     &structpb.Struct{},
		}
	}

	cleared := prefilled()
	_, err := converter.ConvertE(domainDoc{}, &cleared, converter.WithErrorCodecs(jsonpb.NewE()))
	require.NoError(t, err)
	require.Equal(t, protoDoc{}, cleared)

	kept := prefilled()
	_, err = converter.ConvertE(domainDoc{}, &kept, converter.WithErrorCodecs(jsonpb.NewE(jsonpb.WithIgnoreNil())))
	require.NoError(t, err)
	require.Equal(t, "old", kept.Payload.GetStringValue())
	require.NotNil(t, kept.Tags)
	require.NotNil(t, kept.Meta)
	require.NotNil(t, kept.Raw)

	// The converter skips nil pointer fields before codecs run; called
	// directly, the codec clears the destination for a nil message.
	for _, src := range []any{(*structpb.Value)(nil), (*structpb.ListValue)(nil), (*structpb.Struct)(nil)} {
		var dst any = "old"
		require.NoError(t, jsonpb.NewE()("field", reflect.ValueOf(src), reflect.ValueOf(&dst).Elem(), failNext(t)))
		require.Nil(t, dst)

		dst = "old"
		require.NoError(t, jsonpb.NewE(jsonpb.WithIgnoreNil())("field", reflect.ValueOf(src), reflect.ValueOf(&dst).Elem(), failNext(t)))
		require.Equal(t, "old", dst)
	}
}

func TestNewE_Errors(t *testing.T) {
	t.Parallel()

	invalidUTF8 := string([]byte{0xff})
	for _, tc := range []struct {
		name string
		src  any
		dst  any
		want error
	}{
		{
			name: "NaN in",
			src:  domainDoc{Payload: math.NaN()},
			dst:  &protoDoc{},
			want: jsonpb.ValueError{Path: "payload", Reason: "non-finite numbers have no JSON representation"},
		},
		{
			name: "infinity nested in list",
			src:  domainDoc{Tags: []any{"a", map[string]any{"x": math.Inf(1)}}},
			dst:  &protoDoc{},
			want: jsonpb.ValueError{Path: "tags[1].x", Value: math.Inf(1), Reason: "non-finite numbers have no JSON representation"},
		},
		{
			name: "unsupported type",
			src:  domainDoc{Meta: map[string]any{"ch": struct{}{}}},
			dst:  &protoDoc{},
			want: jsonpb.ValueError{Path: "meta.ch", Value: struct{}{}, Reason: "unsupported type"},
		},
		{
			name: "invalid UTF-8",
			src:  domainDoc{Payload: invalidUTF8},
			dst:  &protoDoc{},
			want: jsonpb.ValueError{Path: "payload", Value: invalidUTF8, Reason: "invalid UTF-8"},
		},
		{
			name: "list from a map",
			src:  struct{ Tags any }{Tags: map[string]any{}},
			dst:  &protoDoc{},
			want: jsonpb.ValueError{Path: "tags", Value: map[string]any{}, Reason: "unsupported type"},
		},
		{
			name: "infinity out",
			src:  protoDoc{Tags: &structpb.ListValue{Values: []*structpb.Value{structpb.NewNumberValue(math.Inf(-1))}}},
			dst:  &domainDoc{},
			want: jsonpb.ValueError{Path: "tags[0]", Value: math.Inf(-1), Reason: "non-finite numbers have no JSON representation"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := converter.ConvertE(tc.src, tc.dst, converter.WithErrorCodecs(jsonpb.NewE()))
			if want, ok := tc.want.(jsonpb.ValueError); ok && want.Value == nil {
				// NaN never compares equal; check the rest of the error.
				var got jsonpb.ValueError
				require.ErrorAs(t, err, &got)
				f, isFloat := got.Value.(float64)
				require.True(t, isFloat && math.IsNaN(f))

				got.Value = nil
				require.Equal(t, want, got)

				return
			}

			require.Equal(t, tc.want, err)
		})
	}
}

func TestNewE_InvalidRawMessage(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{`{invalid`, `[1,2]`, `"text"`} {
		_, err := converter.ConvertE(domainDoc{Raw: json.RawMessage(raw)}, &protoDoc{}, converter.WithErrorCodecs(jsonpb.NewE()))

		var got jsonpb.Error
		require.ErrorAs(t, err, &got, raw)
		require.Equal(t, "raw", got.Path)
		require.Error(t, got.Unwrap())
		require.Equal(t, got, err, "the codec error is returned unchanged")
	}
}

func TestNewE_PassThrough(t *testing.T) {
	t.Parallel()

	codec := jsonpb.NewE()

	for _, tc := range []struct {
		name     string
		src, dst any
	}{
		{name: "strings", src: "a", dst: new(string)},
		{name: "value to value", src: structpb.NewNullValue(), dst: new(*structpb.Value)},
		{name: "struct to slice", src: &structpb.Struct{}, dst: new([]any)},
		{name: "channel to value", src: make(chan int), dst: new(*structpb.Value)},
		{name: "typed slice to list", src: []string{"a"}, dst: new(*structpb.ListValue)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			called := false
			err := codec("field", reflect.ValueOf(tc.src), reflect.ValueOf(tc.dst).Elem(), func(string, reflect.Value, reflect.Value) error {
				called = true
				return nil
			})
			require.NoError(t, err)
			require.True(t, called)
		})
	}
}

func TestNewE_LegacyNewUnchanged(t *testing.T) {
	t.Parallel()

	// The legacy codec still passes invalid JSON to next instead of failing.
	raw := json.RawMessage(`{invalid`)

	var dst *structpb.Struct

	called := false

	jsonpb.New()("Raw", reflect.ValueOf(&raw), reflect.ValueOf(&dst).Elem(), func(string, reflect.Value, reflect.Value) { called = true })
	require.True(t, called)
	require.Nil(t, dst)
}

func TestNewE_NestedTypedNilsBecomeNull(t *testing.T) {
	t.Parallel()

	src := []any{map[string]any(nil), []any(nil), map[string]any{"k": []any(nil)}}

	var dst *structpb.ListValue

	codec := jsonpb.NewE()
	err := codec("v", reflect.ValueOf(src), reflect.ValueOf(&dst).Elem(),
		func(string, reflect.Value, reflect.Value) error { return nil })
	require.NoError(t, err)
	require.Len(t, dst.GetValues(), 3)
	require.IsType(t, &structpb.Value_NullValue{}, dst.GetValues()[0].GetKind())
	require.IsType(t, &structpb.Value_NullValue{}, dst.GetValues()[1].GetKind())
	require.IsType(t, &structpb.Value_NullValue{}, dst.GetValues()[2].GetStructValue().GetFields()["k"].GetKind())
}
