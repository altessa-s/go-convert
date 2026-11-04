// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package anypb_test

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-convert/proto/codec/anypb"
	"github.com/altessa-s/go-convert/proto/codec/tspb"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	converter "github.com/altessa-s/go-convert"
	convcodec "github.com/altessa-s/go-convert/codec"
	pbany "google.golang.org/protobuf/types/known/anypb"
)

// Note is a domain type mirroring wrapperspb.StringValue field by field.
type Note struct{ Value string }

type domainEnvelope struct {
	Body    *wrapperspb.StringValue
	At      time.Time
	Note    *Note
	Payload any
}

type protoEnvelope struct {
	Body    *pbany.Any
	At      *pbany.Any
	Note    *pbany.Any
	Payload *pbany.Any
}

func newRegistry() *anypb.Registry {
	return anypb.NewRegistry().
		Register(&timestamppb.Timestamp{}, reflect.TypeFor[time.Time]()).
		Register(&wrapperspb.StringValue{}, reflect.TypeFor[*Note]())
}

func envelopeOptions(opts ...anypb.Option) []converter.Option {
	opts = append([]anypb.Option{anypb.WithRegistry(newRegistry())}, opts...)

	return []converter.Option{
		converter.WithErrorCodecs(anypb.New(opts...)),
		converter.WithCodecs(tspb.New()),
	}
}

func mustAny(t testing.TB, m proto.Message) *pbany.Any {
	t.Helper()

	a, err := pbany.New(m)
	require.NoError(t, err)

	return a
}

func failNext(t *testing.T) convcodec.CodecHandlerE {
	t.Helper()

	return func(string, reflect.Value, reflect.Value) error {
		require.Fail(t, "next should not be called")
		return nil
	}
}

func TestNew_Messages(t *testing.T) {
	t.Parallel()

	instant := time.Date(2026, time.March, 14, 9, 26, 53, 589793000, time.UTC)
	for _, tc := range []struct {
		name    string
		message proto.Message
	}{
		{name: "string", message: wrapperspb.String("hello")},
		{name: "int64", message: wrapperspb.Int64(-42)},
		{name: "timestamp", message: timestamppb.New(instant)},
		{name: "duration", message: durationpb.New(90 * time.Second)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			codec := anypb.New()
			// Message -> Any.
			var packed *pbany.Any
			require.NoError(t, codec("body", reflect.ValueOf(tc.message), reflect.ValueOf(&packed).Elem(), failNext(t)))
			require.True(t, packed.MessageIs(tc.message))

			// Any -> pointer message.
			unpacked := reflect.New(reflect.TypeOf(tc.message))
			require.NoError(t, codec("body", reflect.ValueOf(packed), unpacked.Elem(), failNext(t)))
			got, ok := reflect.TypeAssert[proto.Message](unpacked.Elem())
			require.True(t, ok)
			require.True(t, proto.Equal(tc.message, got), "got %v", got)

			// Any -> non-pointer message.
			value := reflect.New(reflect.TypeOf(tc.message).Elem())
			require.NoError(t, codec("body", reflect.ValueOf(packed), value.Elem(), failNext(t)))
			got, ok = reflect.TypeAssert[proto.Message](value)
			require.True(t, ok)
			require.True(t, proto.Equal(tc.message, got), "got %v", got)
		})
	}
}

func TestNew_RegistryRoundTrip(t *testing.T) {
	t.Parallel()

	instant := time.Date(2026, time.March, 14, 9, 26, 53, 0, time.UTC)
	for _, tc := range []struct {
		name    string
		payload any
	}{
		{name: "registered domain value", payload: Note{Value: "inline"}},
		{name: "registered domain pointer", payload: &Note{Value: "pointer"}},
		{name: "registered time", payload: instant},
		{name: "unregistered message", payload: durationpb.New(time.Minute)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			src := domainEnvelope{Body: wrapperspb.String("body"), At: instant, Note: &Note{Value: "note"}, Payload: tc.payload}

			var message protoEnvelope

			_, err := converter.ConvertE(src, &message, envelopeOptions()...)
			require.NoError(t, err)
			require.True(t, proto.Equal(mustAny(t, wrapperspb.String("body")), message.Body))
			require.True(t, proto.Equal(mustAny(t, timestamppb.New(instant)), message.At))
			require.True(t, proto.Equal(mustAny(t, wrapperspb.String("note")), message.Note))

			var back domainEnvelope

			_, err = converter.ConvertE(message, &back, envelopeOptions()...)
			require.NoError(t, err)
			require.True(t, proto.Equal(src.Body, back.Body))
			require.True(t, instant.Equal(back.At))
			require.Equal(t, src.Note, back.Note)

			// An interface destination receives the registered domain value,
			// or the resolved message for unregistered types.
			switch want := tc.payload.(type) {
			case *Note:
				require.Equal(t, *want, back.Payload)
			case time.Time:
				got, ok := back.Payload.(time.Time)
				require.True(t, ok)
				require.True(t, want.Equal(got))
			case proto.Message:
				got, ok := back.Payload.(proto.Message)
				require.True(t, ok)
				require.True(t, proto.Equal(want, got))
			default:
				require.Equal(t, want, back.Payload)
			}
		})
	}
}

func TestNew_Resolver(t *testing.T) {
	t.Parallel()

	types := new(protoregistry.Types)
	require.NoError(t, types.RegisterMessage((&durationpb.Duration{}).ProtoReflect().Type()))
	codec := anypb.New(anypb.WithResolver(types))

	var dst proto.Message

	src := mustAny(t, durationpb.New(time.Second))
	require.NoError(t, codec("payload", reflect.ValueOf(src), reflect.ValueOf(&dst).Elem(), failNext(t)))
	require.True(t, proto.Equal(durationpb.New(time.Second), dst))

	// A type the custom resolver does not know is unknown, even though it is
	// in protoregistry.GlobalTypes.
	err := codec("payload", reflect.ValueOf(mustAny(t, wrapperspb.String("x"))), reflect.ValueOf(&dst).Elem(), failNext(t))
	require.Equal(t, anypb.UnknownTypeError{Path: "payload", TypeURL: "type.googleapis.com/google.protobuf.StringValue"}, err)
}

func TestNew_Nil(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		src  any
		dst  func() any
	}{
		{name: "nil Any into message", src: (*pbany.Any)(nil), dst: func() any { d := wrapperspb.String("old"); return &d }},
		{name: "nil Any into interface", src: (*pbany.Any)(nil), dst: func() any { var d any = "old"; return &d }},
		{name: "nil message into Any", src: (*wrapperspb.StringValue)(nil), dst: func() any { d := mustAny(t, wrapperspb.String("old")); return &d }},
		{name: "nil interface into Any", src: struct{ V any }{}, dst: func() any { d := mustAny(t, wrapperspb.String("old")); return &d }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			src := reflect.ValueOf(tc.src)
			if src.Kind() == reflect.Struct {
				src = src.Field(0)
			}

			cleared := reflect.ValueOf(tc.dst()).Elem()
			require.NoError(t, anypb.New()("v", src, cleared, failNext(t)))
			require.True(t, cleared.IsZero())

			kept := reflect.ValueOf(tc.dst()).Elem()
			require.NoError(t, anypb.New(anypb.WithIgnoreNil())("v", src, kept, failNext(t)))
			require.False(t, kept.IsZero())
		})
	}
}

// errFromNext is returned by a codec registered after anypb.
var errFromNext = errors.New("next failed")

func TestNew_Errors(t *testing.T) {
	t.Parallel()

	const stringURL = "type.googleapis.com/google.protobuf.StringValue"

	type stringerDst struct{ Payload fmt.Stringer }

	type anyDst struct{ Payload any }

	type anySrc struct{ Payload *pbany.Any }

	failing := func(path string, src, dst reflect.Value, next convcodec.CodecHandlerE) error {
		if path == "note" {
			return errFromNext
		}

		return next(path, src, dst)
	}

	for _, tc := range []struct {
		name string
		src  any
		dst  any
		opts []converter.Option
		want error
	}{
		{
			name: "message type mismatch",
			src:  anySrc{Payload: mustAny(t, wrapperspb.String("x"))},
			dst:  &struct{ Payload *durationpb.Duration }{},
			want: anypb.TypeMismatchError{Path: "payload", TypeURL: stringURL, Want: "google.protobuf.Duration"},
		},
		{
			name: "domain type mismatch",
			src:  anySrc{Payload: mustAny(t, durationpb.New(time.Second))},
			dst:  &struct{ Payload time.Time }{},
			want: anypb.TypeMismatchError{
				Path: "payload", TypeURL: "type.googleapis.com/google.protobuf.Duration", Want: "google.protobuf.Timestamp",
			},
		},
		{
			name: "interface not implemented",
			src:  anySrc{Payload: mustAny(t, wrapperspb.String("x"))},
			dst:  &stringerDst{},
			want: anypb.TypeMismatchError{Path: "payload", TypeURL: stringURL, Want: "fmt.Stringer"},
		},
		{
			name: "unknown type URL",
			src:  anySrc{Payload: &pbany.Any{TypeUrl: "type.googleapis.com/acme.Missing"}},
			dst:  &anyDst{},
			want: anypb.UnknownTypeError{Path: "payload", TypeURL: "type.googleapis.com/acme.Missing"},
		},
		{
			name: "empty type URL",
			src:  anySrc{Payload: &pbany.Any{}},
			dst:  &anyDst{},
			want: anypb.UnknownTypeError{Path: "payload"},
		},
		{
			name: "error from next",
			src:  domainEnvelope{Note: &Note{Value: "x"}},
			dst:  &protoEnvelope{},
			opts: []converter.Option{converter.WithErrorCodecs(failing)},
			want: errFromNext,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			opts := append(envelopeOptions(), tc.opts...)
			_, err := converter.ConvertE(tc.src, tc.dst, opts...)
			require.Equal(t, tc.want, err, "the codec error is returned unchanged")
		})
	}
}

func TestNew_InvalidPayload(t *testing.T) {
	t.Parallel()

	src := &pbany.Any{TypeUrl: "type.googleapis.com/google.protobuf.StringValue", Value: []byte{0xff}}
	for _, dst := range []any{new(*wrapperspb.StringValue), new(*Note), new(any)} {
		err := anypb.New(anypb.WithRegistry(newRegistry()))("body", reflect.ValueOf(src), reflect.ValueOf(dst).Elem(), failNext(t))

		var got anypb.Error
		require.ErrorAs(t, err, &got)
		require.Equal(t, "body", got.Path)
		require.Error(t, got.Unwrap())
	}
}

func TestNew_PassThrough(t *testing.T) {
	t.Parallel()

	codec := anypb.New()

	for _, tc := range []struct {
		name     string
		src, dst any
	}{
		{name: "strings", src: "a", dst: new(string)},
		{name: "any to any", src: mustAny(t, wrapperspb.String("x")), dst: new(*pbany.Any)},
		{name: "unregistered struct from Any", src: mustAny(t, wrapperspb.String("x")), dst: new(Note)},
		{name: "unregistered struct to Any", src: Note{}, dst: new(*pbany.Any)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			called := false
			err := codec("v", reflect.ValueOf(tc.src), reflect.ValueOf(tc.dst).Elem(), func(string, reflect.Value, reflect.Value) error {
				called = true
				return nil
			})
			require.NoError(t, err)
			require.True(t, called)
		})
	}
}

func TestNew_InterfaceHoldingAny(t *testing.T) {
	t.Parallel()

	packed := mustAny(t, wrapperspb.String("x"))
	holder := struct{ V any }{V: packed}

	var dst *pbany.Any
	require.NoError(t, anypb.New()("v", reflect.ValueOf(holder).Field(0), reflect.ValueOf(&dst).Elem(), failNext(t)))
	require.Same(t, packed, dst)
}

func TestRegistry_RegisterPanics(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		register func(r *anypb.Registry)
	}{
		{name: "nil message", register: func(r *anypb.Registry) { r.Register(nil, reflect.TypeFor[Note]()) }},
		{name: "nil domain", register: func(r *anypb.Registry) { r.Register(&wrapperspb.StringValue{}, nil) }},
		{name: "interface domain", register: func(r *anypb.Registry) { r.Register(&wrapperspb.StringValue{}, reflect.TypeFor[fmt.Stringer]()) }},
		{name: "duplicate message", register: func(r *anypb.Registry) { r.Register(&wrapperspb.StringValue{}, reflect.TypeFor[string]()) }},
		{name: "duplicate domain", register: func(r *anypb.Registry) { r.Register(&wrapperspb.BytesValue{}, reflect.TypeFor[Note]()) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			registry := anypb.NewRegistry().Register(&wrapperspb.StringValue{}, reflect.TypeFor[Note]())

			require.Panics(t, func() { tc.register(registry) })
		})
	}
}
