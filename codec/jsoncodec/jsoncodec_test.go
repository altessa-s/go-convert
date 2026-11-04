// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package jsoncodec_test

import (
	stdjson "encoding/json"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-convert/codec/jsoncodec"

	converter "github.com/altessa-s/go-convert"
	convcodec "github.com/altessa-s/go-convert/codec"
)

type point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// run invokes codec on a source value and a pointer to a destination,
// failing the test when the codec delegates.
func run(t *testing.T, codec convcodec.CodecE, src, dstPtr any) error {
	t.Helper()
	return codec("Field", reflect.ValueOf(src), reflect.ValueOf(dstPtr).Elem(), func(string, reflect.Value, reflect.Value) error {
		require.Fail(t, "next should not be called")
		return nil
	})
}

func TestUnmarshal(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		opts   []jsoncodec.Option
		src    any
		dst    any // pointer to a destination
		expect any
	}{
		{name: "RawMessage to struct", src: stdjson.RawMessage(`{"x":1,"y":2}`), dst: new(point), expect: point{X: 1, Y: 2}},
		{name: "jsontext.Value to *struct", src: jsontext.Value(`{"x":3}`), dst: new(*point), expect: &point{X: 3}},
		{name: "*RawMessage to map", src: new(stdjson.RawMessage(`{"a":1}`)), dst: new(map[string]int), expect: map[string]int{"a": 1}},
		{name: "to slice", src: jsontext.Value(`[1,2]`), dst: new([]int), expect: []int{1, 2}},
		{name: "to any", src: jsontext.Value(`{"a":[true]}`), dst: new(any), expect: map[string]any{"a": []any{true}}},
		{name: "replaces instead of merging", src: jsontext.Value(`{"b":2}`), dst: &map[string]int{"a": 1}, expect: map[string]int{"b": 2}},
		{name: "null", src: jsontext.Value(`null`), dst: &map[string]int{"a": 1}, expect: map[string]int(nil)},
		{name: "empty text", src: jsontext.Value(nil), dst: &point{X: 9}, expect: point{}},
		{name: "nil source", src: (*jsontext.Value)(nil), dst: new(&point{X: 9}), expect: (*point)(nil)},
		{name: "bytes", opts: []jsoncodec.Option{jsoncodec.WithBytes()}, src: []byte(`{"x":5}`), dst: new(point), expect: point{X: 5}},
		{name: "string", opts: []jsoncodec.Option{jsoncodec.WithStrings()}, src: `{"y":6}`, dst: new(point), expect: point{Y: 6}},
		{name: "*string", opts: []jsoncodec.Option{jsoncodec.WithStrings()}, src: new(`[1]`), dst: new([]int), expect: []int{1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.NoError(t, run(t, jsoncodec.New(tt.opts...), tt.src, tt.dst))
			require.Equal(t, tt.expect, reflect.ValueOf(tt.dst).Elem().Interface())
		})
	}
}

func TestMarshal(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		opts   []jsoncodec.Option
		src    any
		dst    any // pointer to a destination
		expect any
	}{
		{name: "struct to RawMessage", src: point{X: 1, Y: 2}, dst: new(stdjson.RawMessage), expect: stdjson.RawMessage(`{"x":1,"y":2}`)},
		{name: "*struct to *jsontext.Value", src: &point{X: 1}, dst: new(*jsontext.Value), expect: new(jsontext.Value(`{"x":1,"y":0}`))},
		{name: "map", src: map[string]int{"b": 2, "a": 1}, dst: new(jsontext.Value), expect: jsontext.Value(`{"a":1,"b":2}`)},
		{name: "slice", src: []string{"a"}, dst: new(jsontext.Value), expect: jsontext.Value(`["a"]`)},
		{name: "nil slice", src: []string(nil), dst: new(jsontext.Value), expect: jsontext.Value(`[]`)},
		{
			name: "nil slice as null", opts: []jsoncodec.Option{jsoncodec.WithJSONOptions(json.FormatNilSliceAsNull(true))},
			src: []string(nil), dst: new(jsontext.Value), expect: jsontext.Value(`null`),
		},
		{name: "nil pointer", src: (*point)(nil), dst: new(jsontext.Value(`{}`)), expect: jsontext.Value(nil)},
		{name: "bytes", opts: []jsoncodec.Option{jsoncodec.WithBytes()}, src: point{}, dst: new([]byte), expect: []byte(`{"x":0,"y":0}`)},
		{name: "string", opts: []jsoncodec.Option{jsoncodec.WithStrings()}, src: []int{1}, dst: new(""), expect: `[1]`},
		{name: "*string", opts: []jsoncodec.Option{jsoncodec.WithStrings()}, src: []int{1}, dst: new((*string)(nil)), expect: new(`[1]`)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			opts := append([]jsoncodec.Option{jsoncodec.WithJSONOptions(json.Deterministic(true))}, tt.opts...)
			require.NoError(t, run(t, jsoncodec.New(opts...), tt.src, tt.dst))
			require.Equal(t, tt.expect, reflect.ValueOf(tt.dst).Elem().Interface())
		})
	}
}

func TestMarshalAny(t *testing.T) {
	t.Parallel()
	type holder struct{ V any }
	var text jsontext.Value
	src := reflect.ValueOf(holder{V: map[string]int{"a": 1}}).Field(0)
	require.NoError(t, jsoncodec.New()("V", src, reflect.ValueOf(&text).Elem(), nil))
	require.JSONEq(t, `{"a":1}`, string(text))

	text = jsontext.Value(`1`)
	src = reflect.ValueOf(holder{}).Field(0)
	require.NoError(t, jsoncodec.New()("V", src, reflect.ValueOf(&text).Elem(), nil))
	require.Nil(t, text, "a nil interface zeroes the destination")
}

type failingMarshaler struct{}

func (failingMarshaler) MarshalJSON() ([]byte, error) { return nil, errors.New("boom") }

func TestErrors(t *testing.T) {
	t.Parallel()

	dst := point{X: 9}
	err := run(t, jsoncodec.New(), jsontext.Value(`{"x":`), &dst)
	var codecErr jsoncodec.Error
	require.ErrorAs(t, err, &codecErr)
	require.Equal(t, "Field", codecErr.Path)
	require.Equal(t, jsoncodec.OpUnmarshal, codecErr.Op)
	var syntactic *jsontext.SyntacticError
	require.ErrorAs(t, err, &syntactic)
	require.Equal(t, point{X: 9}, dst, "the destination is unchanged on error")

	err = run(t, jsoncodec.New(jsoncodec.WithJSONOptions(json.RejectUnknownMembers(true))), jsontext.Value(`{"z":1}`), &dst)
	var semantic *json.SemanticError
	require.ErrorAs(t, err, &semantic)

	var text jsontext.Value
	err = run(t, jsoncodec.New(), []failingMarshaler{{}}, &text)
	require.ErrorAs(t, err, &codecErr)
	require.Equal(t, jsoncodec.OpMarshal, codecErr.Op)
	require.Contains(t, err.Error(), `jsoncodec: field "Field": marshal: `)
	require.Nil(t, text)
}

func TestDelegates(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		opts     []jsoncodec.Option
		src, dst any
	}{
		{name: "string to struct without option", src: `{}`, dst: new(point)},
		{name: "bytes to struct without option", src: []byte(`{}`), dst: new(point)},
		{name: "struct to string without option", src: point{}, dst: new(string)},
		{name: "text to text", opts: []jsoncodec.Option{jsoncodec.WithStrings()}, src: jsontext.Value(`1`), dst: new(string)},
		{name: "text to bytes", src: jsontext.Value(`1`), dst: new([]byte)},
		{name: "text to int", src: jsontext.Value(`1`), dst: new(int)},
		{name: "struct to struct", src: point{}, dst: new(point)},
		{name: "text to non-empty interface", src: jsontext.Value(`1`), dst: new(fmt.Stringer)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			called := false
			err := jsoncodec.New(tt.opts...)("Field", reflect.ValueOf(tt.src), reflect.ValueOf(tt.dst).Elem(),
				func(string, reflect.Value, reflect.Value) error {
					called = true
					return nil
				})
			require.NoError(t, err)
			require.True(t, called)
		})
	}
}

type event struct {
	ID      int
	Payload jsontext.Value
	Meta    stdjson.RawMessage
	Tags    *jsontext.Value
}

type eventModel struct {
	ID      int
	Payload point
	Meta    map[string]any
	Tags    []string
}

func TestConverterRoundTrip(t *testing.T) {
	t.Parallel()
	opts := []converter.Option{converter.WithErrorCodecs(jsoncodec.New(jsoncodec.WithJSONOptions(json.Deterministic(true))))}
	src := event{ID: 1, Payload: jsontext.Value(`{"x":1,"y":2}`), Meta: stdjson.RawMessage(`{"k":"v"}`), Tags: new(jsontext.Value(`["a","b"]`))}

	model, err := converter.ConvertE(src, &eventModel{}, opts...)
	require.NoError(t, err)
	require.Equal(t, &eventModel{ID: 1, Payload: point{X: 1, Y: 2}, Meta: map[string]any{"k": "v"}, Tags: []string{"a", "b"}}, model)

	back, err := converter.ConvertE(*model, &event{}, opts...)
	require.NoError(t, err)
	require.Equal(t, &src, back)
}

func TestConverterReturnsErrorUnchanged(t *testing.T) {
	t.Parallel()
	_, err := converter.ConvertE(event{Payload: jsontext.Value(`{"x":"one"}`)}, &eventModel{}, converter.WithErrorCodecs(jsoncodec.New()))
	var codecErr jsoncodec.Error
	require.ErrorAs(t, err, &codecErr)
	require.Equal(t, "payload", codecErr.Path)
	require.Equal(t, jsoncodec.OpUnmarshal, codecErr.Op)
	require.IsType(t, jsoncodec.Error{}, err, "the codec error is returned as is")
}

func ExampleNew() {
	type Order struct {
		ID    int
		Items stdjson.RawMessage
	}
	type Item struct {
		SKU string `json:"sku"`
		Qty int    `json:"qty"`
	}
	type OrderModel struct {
		ID    int
		Items []Item
	}
	codec := jsoncodec.New()

	model, err := converter.ConvertE(Order{ID: 7, Items: stdjson.RawMessage(`[{"sku":"A1","qty":2}]`)}, &OrderModel{}, converter.WithErrorCodecs(codec))
	fmt.Println(model.Items, err)

	order, err := converter.ConvertE(*model, &Order{}, converter.WithErrorCodecs(codec))
	fmt.Println(string(order.Items), err)

	_, err = converter.ConvertE(Order{Items: stdjson.RawMessage(`{"sku":"A1"}`)}, &OrderModel{}, converter.WithErrorCodecs(codec))
	fmt.Println(err != nil)
	// Output:
	// [{A1 2}] <nil>
	// [{"sku":"A1","qty":2}] <nil>
	// true
}
