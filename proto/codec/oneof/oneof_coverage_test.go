// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package oneof_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-convert/proto/codec/oneof"

	convcodec "github.com/altessa-s/go-convert/codec"
)

// Fixtures with differing field types on both sides, so the standard
// conversion has to convert, copy slices and copy struct fields.

type pbContractor struct {
	Name  string
	Tags  []string
	Score int64
	Level int
}

type domContractor struct {
	Name  string
	Tags  []string
	Score int32
	Level bool // not convertible to or from int: left zero
	Extra chan int
}

type isConvMessage_Body interface { //nolint:revive // protoc naming mirrored on purpose
	isConvMessage_Body()
}

type ConvMessage_Contractor struct { //nolint:revive // protoc naming mirrored on purpose
	Contractor *pbContractor
}

func (*ConvMessage_Contractor) isConvMessage_Body() {}

type ConvMessage_IDs struct { //nolint:revive // protoc naming mirrored on purpose
	IDs []int64
}

func (*ConvMessage_IDs) isConvMessage_Body() {}

type ConvMessage_Count struct { //nolint:revive // protoc naming mirrored on purpose
	Count int64
}

func (*ConvMessage_Count) isConvMessage_Body() {}

// ConvMessage_Other has no field named after its registry key.
type ConvMessage_Other struct { //nolint:revive // protoc naming mirrored on purpose
	Unrelated *string
}

func (*ConvMessage_Other) isConvMessage_Body() {}

type convMessage struct {
	Body isConvMessage_Body
}

type convPayload struct {
	Contractor *domContractor
	IDs        *[]int32
	Count      *int64
	unexported *int //nolint:unused // pins that unexported fields are skipped
}

var convRegistry = map[string]any{
	"Contractor": &ConvMessage_Contractor{},
	"IDs":        &ConvMessage_IDs{},
	"Count":      &ConvMessage_Count{},
}

// run invokes codec and reports whether it fell through to next.
func run(codec convcodec.Codec, src, dst reflect.Value) bool {
	passed := false

	codec("Body", src, dst, func(string, reflect.Value, reflect.Value) { passed = true })

	return passed
}

func bodyField(m *convMessage) reflect.Value { return reflect.ValueOf(m).Elem().FieldByName("Body") }

func TestStructToOneof_ConvertsStructFields(t *testing.T) {
	t.Parallel()

	codec := oneof.New(oneof.WithWrapperRegistry(convRegistry))
	src := convPayload{Contractor: &domContractor{Name: "acme", Tags: []string{"a", "b"}, Score: 7, Level: true}}

	var msg convMessage
	require.False(t, run(codec, reflect.ValueOf(src), bodyField(&msg)))

	w, ok := msg.Body.(*ConvMessage_Contractor)
	require.True(t, ok, "%T", msg.Body)
	require.Equal(t, &pbContractor{Name: "acme", Tags: []string{"a", "b"}, Score: 7}, w.Contractor)
}

func TestStructToOneof_ConvertsPrimitiveSlice(t *testing.T) {
	t.Parallel()

	codec := oneof.New(oneof.WithWrapperRegistry(convRegistry))
	ids := []int32{1, 2, 3}

	var msg convMessage
	require.False(t, run(codec, reflect.ValueOf(&convPayload{IDs: &ids}), bodyField(&msg)))
	require.Equal(t, &ConvMessage_IDs{IDs: []int64{1, 2, 3}}, msg.Body)

	empty := []int32{}
	msg = convMessage{}
	require.False(t, run(codec, reflect.ValueOf(convPayload{IDs: &empty}), bodyField(&msg)))
	require.Equal(t, &ConvMessage_IDs{}, msg.Body)
}

func TestStructToOneof_StrictValidationRejectsSeveralActiveFields(t *testing.T) {
	t.Parallel()

	ids := []int32{1}
	src := reflect.ValueOf(convPayload{Contractor: &domContractor{}, IDs: &ids})

	var msg convMessage
	require.True(t, run(oneof.New(oneof.WithWrapperRegistry(convRegistry), oneof.WithStrictValidation()), src, bodyField(&msg)))
	require.Nil(t, msg.Body)

	// Without strict validation the first active field wins.
	require.False(t, run(oneof.New(oneof.WithWrapperRegistry(convRegistry)), src, bodyField(&msg)))
	require.IsType(t, &ConvMessage_Contractor{}, msg.Body)

	// With exactly one active field strict validation converts as usual.
	msg = convMessage{}
	require.False(t, run(oneof.New(oneof.WithWrapperRegistry(convRegistry), oneof.WithStrictValidation()),
		reflect.ValueOf(convPayload{IDs: &ids}), bodyField(&msg)))
	require.Equal(t, &ConvMessage_IDs{IDs: []int64{1}}, msg.Body)
}

func TestStructToOneof_AllNil(t *testing.T) {
	t.Parallel()

	src := reflect.ValueOf(convPayload{})

	var msg convMessage
	require.True(t, run(oneof.New(oneof.WithWrapperRegistry(convRegistry)), src, bodyField(&msg)), "falls through")
	require.Nil(t, msg.Body)

	require.False(t, run(oneof.New(oneof.WithWrapperRegistry(convRegistry), oneof.WithIgnoreNilFields()), src, bodyField(&msg)),
		"swallowed with WithIgnoreNilFields")
	require.Nil(t, msg.Body)

	require.False(t, run(oneof.New(oneof.WithWrapperRegistry(convRegistry), oneof.WithFallbackField("contractor")), src, bodyField(&msg)))
	require.Equal(t, &ConvMessage_Contractor{}, msg.Body, "the fallback variant is set, empty")

	msg = convMessage{}
	require.True(t, run(oneof.New(oneof.WithWrapperRegistry(convRegistry), oneof.WithFallbackField("missing")), src, bodyField(&msg)),
		"an unknown fallback field falls through")
	require.Nil(t, msg.Body)
}

func TestStructToOneof_UnresolvableWrapperFallsThrough(t *testing.T) {
	t.Parallel()

	src := reflect.ValueOf(convPayload{Contractor: &domContractor{Name: "x"}})

	var msg convMessage
	require.True(t, run(oneof.New(), src, bodyField(&msg)), "no registry")
	require.True(t, run(oneof.New(oneof.WithWrapperRegistry(map[string]any{"IDs": &ConvMessage_IDs{}})), src, bodyField(&msg)),
		"field not registered")
	require.True(t, run(oneof.New(oneof.WithWrapperRegistry(map[string]any{"Contractor": ConvMessage_Other{}})), src, bodyField(&msg)),
		"registered wrapper lacks the field")
	require.Nil(t, msg.Body)
}

func TestFieldConverter_BothDirections(t *testing.T) {
	t.Parallel()

	toPB := func(src reflect.Value) (reflect.Value, error) {
		d := src.Interface().(*domContractor)
		return reflect.ValueOf(&pbContractor{Name: "pb:" + d.Name}), nil
	}
	codec := oneof.New(oneof.WithWrapperRegistry(convRegistry), oneof.WithFieldConverter("CONTRACTOR", toPB))

	var msg convMessage
	require.False(t, run(codec, reflect.ValueOf(convPayload{Contractor: &domContractor{Name: "a"}}), bodyField(&msg)))
	require.Equal(t, &ConvMessage_Contractor{Contractor: &pbContractor{Name: "pb:a"}}, msg.Body)

	toDom := func(src reflect.Value) (reflect.Value, error) {
		p := src.Interface().(*pbContractor)
		return reflect.ValueOf(&domContractor{Name: "dom:" + p.Name}), nil
	}
	back := oneof.New(oneof.WithFieldConverter("contractor", toDom))

	var dst convPayload
	require.False(t, run(back, bodyField(&msg), reflect.ValueOf(&dst).Elem()))
	require.Equal(t, &domContractor{Name: "dom:pb:a"}, dst.Contractor)
}

func TestFieldConverter_ErrorFallsThrough(t *testing.T) {
	t.Parallel()

	fail := func(reflect.Value) (reflect.Value, error) { return reflect.Value{}, errors.New("nope") }
	codec := oneof.New(oneof.WithWrapperRegistry(convRegistry), oneof.WithFieldConverter("contractor", fail))

	var msg convMessage
	require.True(t, run(codec, reflect.ValueOf(convPayload{Contractor: &domContractor{}}), bodyField(&msg)))
	require.Nil(t, msg.Body)

	msg.Body = &ConvMessage_Contractor{Contractor: &pbContractor{Name: "x"}}

	var dst convPayload
	require.True(t, run(codec, bodyField(&msg), reflect.ValueOf(&dst).Elem()))
	require.Nil(t, dst.Contractor)
}

func TestOneofToStruct_ConvertsWrapperValues(t *testing.T) {
	t.Parallel()

	codec := oneof.New()

	msg := convMessage{Body: &ConvMessage_Contractor{Contractor: &pbContractor{Name: "n", Tags: []string{"t"}, Score: 3}}}

	var dst convPayload
	require.False(t, run(codec, bodyField(&msg), reflect.ValueOf(&dst).Elem()))
	require.Equal(t, &domContractor{Name: "n", Tags: []string{"t"}, Score: 3}, dst.Contractor)

	msg = convMessage{Body: &ConvMessage_Count{Count: 4}}
	require.False(t, run(codec, bodyField(&msg), reflect.ValueOf(&dst).Elem()), "a non-pointer wrapper field is unpacked when set")
	require.NotNil(t, dst.Count)
	require.Equal(t, int64(4), *dst.Count)

	msg = convMessage{Body: &ConvMessage_IDs{IDs: []int64{5, 6}}}
	require.False(t, run(codec, bodyField(&msg), reflect.ValueOf(&dst).Elem()))
	require.Equal(t, []int32{5, 6}, *dst.IDs)
}

func TestOneofToStruct_PointerDestinationIsAllocated(t *testing.T) {
	t.Parallel()

	msg := convMessage{Body: &ConvMessage_Count{Count: 9}}

	var dst *convPayload
	require.False(t, run(oneof.New(), bodyField(&msg), reflect.ValueOf(&dst).Elem()))
	require.NotNil(t, dst)
	require.Equal(t, int64(9), *dst.Count)
}

func TestOneofToStruct_FallsThrough(t *testing.T) {
	t.Parallel()

	codec := oneof.New()

	t.Run("empty wrapper", func(t *testing.T) {
		t.Parallel()

		msg := convMessage{Body: &ConvMessage_Count{}}

		var dst convPayload
		require.True(t, run(codec, bodyField(&msg), reflect.ValueOf(&dst).Elem()))
		require.False(t, run(oneof.New(oneof.WithIgnoreNilFields()), bodyField(&msg), reflect.ValueOf(&dst).Elem()))
		require.Nil(t, dst.Count)
	})
	t.Run("nil wrapper pointer in the interface", func(t *testing.T) {
		t.Parallel()

		msg := convMessage{Body: (*ConvMessage_Count)(nil)}

		var dst convPayload
		require.True(t, run(codec, bodyField(&msg), reflect.ValueOf(&dst).Elem()))
	})
	t.Run("destination lacks the field", func(t *testing.T) {
		t.Parallel()

		msg := convMessage{Body: &ConvMessage_Other{Unrelated: new("x")}}

		var dst convPayload
		require.True(t, run(codec, bodyField(&msg), reflect.ValueOf(&dst).Elem()))
	})
	t.Run("destination is not a struct", func(t *testing.T) {
		t.Parallel()

		msg := convMessage{Body: &ConvMessage_Count{Count: 1}}

		var dst string
		require.True(t, run(codec, bodyField(&msg), reflect.ValueOf(&dst).Elem()))
	})
	t.Run("interface that is not a oneof", func(t *testing.T) {
		t.Parallel()

		holder := struct{ V any }{V: &ConvMessage_Count{Count: 1}}

		var dst convPayload
		require.True(t, run(codec, reflect.ValueOf(holder).Field(0), reflect.ValueOf(&dst).Elem()))
	})
	t.Run("oneof holding a non-struct", func(t *testing.T) {
		t.Parallel()

		holder := struct{ V isNumBody }{V: numBody(3)}

		var dst convPayload
		require.True(t, run(codec, reflect.ValueOf(holder).Field(0), reflect.ValueOf(&dst).Elem()))
	})
}

type isNumBody interface { //nolint:revive // protoc naming mirrored on purpose
	isNum_Body()
}

type numBody int

func (numBody) isNum_Body() {}

func TestNewForField_AppliesToTargetFieldCaseInsensitively(t *testing.T) {
	t.Parallel()

	codec := oneof.NewForField("body", oneof.WithWrapperRegistry(convRegistry))

	var msg convMessage
	require.False(t, run(codec, reflect.ValueOf(convPayload{Count: new(int64(2))}), bodyField(&msg)))
	require.Equal(t, &ConvMessage_Count{Count: 2}, msg.Body)
}

// Fixtures for conversions where reflect.ConvertibleTo is true but a plain
// reflect.Value.Convert would change the value's meaning or panic.

type isConvText_Body interface { //nolint:revive // protoc naming mirrored on purpose
	isConvText_Body()
}

type ConvText_Label struct { //nolint:revive // protoc naming mirrored on purpose
	Label string
}

func (*ConvText_Label) isConvText_Body() {}

type ConvText_Labels struct { //nolint:revive // protoc naming mirrored on purpose
	Labels []string
}

func (*ConvText_Labels) isConvText_Body() {}

type ConvText_Raw struct { //nolint:revive // protoc naming mirrored on purpose
	Raw []byte
}

func (*ConvText_Raw) isConvText_Body() {}

type ConvText_Chars struct { //nolint:revive // protoc naming mirrored on purpose
	Chars []rune
}

func (*ConvText_Chars) isConvText_Body() {}

type ConvText_Pair struct { //nolint:revive // protoc naming mirrored on purpose
	Pair [2]int64
}

func (*ConvText_Pair) isConvText_Body() {}

type convText struct {
	Body isConvText_Body
}

type textPayload struct {
	Label  *int64
	Labels *[]uint16
	Raw    *string
	Chars  *string
	Pair   *[]int64
}

var textRegistry = map[string]any{
	"Label":  &ConvText_Label{},
	"Labels": &ConvText_Labels{},
	"Raw":    &ConvText_Raw{},
	"Chars":  &ConvText_Chars{},
	"Pair":   &ConvText_Pair{},
}

func textBody(m *convText) reflect.Value { return reflect.ValueOf(m).Elem().FieldByName("Body") }

func TestStructToOneof_ConversionSemantics(t *testing.T) {
	t.Parallel()

	codec := oneof.New(oneof.WithWrapperRegistry(textRegistry))

	tests := []struct {
		name string
		src  textPayload
		want isConvText_Body
	}{
		{"integer to string is decimal, not a rune", textPayload{Label: new(int64(65))}, &ConvText_Label{Label: "65"}},
		{"negative integer to string", textPayload{Label: new(int64(-7))}, &ConvText_Label{Label: "-7"}},
		{"integer slice to string slice is decimal", textPayload{Labels: &[]uint16{65, 1000}}, &ConvText_Labels{Labels: []string{"65", "1000"}}},
		{"string to bytes keeps UTF-8 bytes", textPayload{Raw: new("héllo")}, &ConvText_Raw{Raw: []byte("héllo")}},
		{"string to runes decodes code points", textPayload{Chars: new("héllo")}, &ConvText_Chars{Chars: []rune("héllo")}},
		{"slice to array of matching length", textPayload{Pair: &[]int64{1, 2}}, &ConvText_Pair{Pair: [2]int64{1, 2}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var msg convText
			require.False(t, run(codec, reflect.ValueOf(tc.src), textBody(&msg)))
			require.Equal(t, tc.want, msg.Body)
		})
	}
}

func TestStructToOneof_ShortSliceToArrayLeavesZero(t *testing.T) {
	t.Parallel()

	// []T -> [N]T is ConvertibleTo, but Convert panics when len < N.
	var msg convText

	require.NotPanics(t, func() {
		run(oneof.New(oneof.WithWrapperRegistry(textRegistry)), reflect.ValueOf(textPayload{Pair: &[]int64{1}}), textBody(&msg))
	})
	require.Equal(t, &ConvText_Pair{}, msg.Body)
}

func TestOneofToStruct_ConversionSemantics(t *testing.T) {
	t.Parallel()

	type textDst struct {
		Label  *int64
		Labels *[]string
		Raw    *string
		Chars  *string
	}

	codec := oneof.New()

	var dst textDst

	msg := convText{Body: &ConvText_Raw{Raw: []byte("héllo")}}
	require.False(t, run(codec, textBody(&msg), reflect.ValueOf(&dst).Elem()))
	require.Equal(t, "héllo", *dst.Raw)

	msg = convText{Body: &ConvText_Chars{Chars: []rune("héllo")}}
	require.False(t, run(codec, textBody(&msg), reflect.ValueOf(&dst).Elem()))
	require.Equal(t, "héllo", *dst.Chars)
}

func TestNew_InvalidValuesFallThroughWithoutPanic(t *testing.T) {
	t.Parallel()

	codec := oneof.New(oneof.WithWrapperRegistry(convRegistry))

	var msg convMessage

	tests := []struct {
		name     string
		src, dst reflect.Value
	}{
		{"zero source", reflect.Value{}, bodyField(&msg)},
		{"nil struct pointer source", reflect.ValueOf((*convPayload)(nil)), bodyField(&msg)},
		{"zero destination", reflect.ValueOf(convPayload{}), reflect.Value{}},
		{"oneof source, zero destination", bodyField(&convMessage{Body: &ConvMessage_Count{Count: 1}}), reflect.Value{}},
		{"both zero", reflect.Value{}, reflect.Value{}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var passed bool

			require.NotPanics(t, func() { passed = run(codec, tc.src, tc.dst) })
			require.True(t, passed, "an invalid value is not a oneof conversion and goes to next")
		})
	}

	require.Nil(t, msg.Body)
}
