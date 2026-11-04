// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package anypb_test

import (
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/altessa-s/go-convert/proto/codec/anypb"
	"github.com/altessa-s/go-convert/proto/codec/tspb"

	converter "github.com/altessa-s/go-convert"
	convcodec "github.com/altessa-s/go-convert/codec"
	pbany "google.golang.org/protobuf/types/known/anypb"
)

// fooer is an interface no protobuf message implements.
type fooer interface{ Foo() }

// copyValue is a next handler that copies the Value field of a struct.
func copyValue(_ string, src, dst reflect.Value) error {
	dst.FieldByName("Value").Set(src.FieldByName("Value"))

	return nil
}

// callsNext runs codec on src and dst and reports whether it called next.
func callsNext(t *testing.T, codec convcodec.CodecE, src, dst reflect.Value) bool {
	t.Helper()

	called := false
	err := codec("v", src, dst, func(string, reflect.Value, reflect.Value) error {
		called = true
		return nil
	})
	require.NoError(t, err)

	return called
}

func TestNew_InterfaceSourceDelegates(t *testing.T) {
	t.Parallel()

	codec := anypb.New(anypb.WithRegistry(newRegistry()))

	for _, payload := range []any{"hello", 42, struct{ N int }{N: 1}, []string{"a"}} {
		holder := struct{ V any }{V: payload}

		var dst *pbany.Any
		require.True(t, callsNext(t, codec, reflect.ValueOf(holder).Field(0), reflect.ValueOf(&dst).Elem()), "%T", payload)
		require.Nil(t, dst)
	}

	// Messages, Anys and registered domain values held by an interface are
	// still packed.
	for _, payload := range []any{wrapperspb.String("x"), Note{Value: "x"}, mustAny(t, wrapperspb.String("x"))} {
		holder := struct{ V any }{V: payload}

		var dst *pbany.Any
		require.NoError(t, codec("v", reflect.ValueOf(holder).Field(0), reflect.ValueOf(&dst).Elem(), copyValue))
		require.True(t, dst.MessageIs(&wrapperspb.StringValue{}), "%T", payload)
	}
}

func TestNew_InterfaceDestinationDelegates(t *testing.T) {
	t.Parallel()

	codec := anypb.New()

	// An unknown type URL into an interface the codec does not own goes to
	// next; into any or proto.Message it is an error.
	unknown := &pbany.Any{TypeUrl: "type.googleapis.com/acme.Missing"}

	var stringer fooer
	require.True(t, callsNext(t, codec, reflect.ValueOf(unknown), reflect.ValueOf(&stringer).Elem()))

	var empty any

	err := codec("v", reflect.ValueOf(unknown), reflect.ValueOf(&empty).Elem(), failNext(t))
	require.Equal(t, anypb.UnknownTypeError{Path: "v", TypeURL: unknown.GetTypeUrl()}, err)

	// A resolved type that does not implement the interface is a mismatch.
	err = codec("v", reflect.ValueOf(mustAny(t, durationpb.New(time.Second))), reflect.ValueOf(&stringer).Elem(), failNext(t))
	require.Equal(t, anypb.TypeMismatchError{Path: "v", TypeURL: "type.googleapis.com/google.protobuf.Duration", Want: "anypb_test.fooer"}, err)
}

func TestNew_RegisteredPairEdgeValues(t *testing.T) {
	t.Parallel()

	registry := anypb.NewRegistry().MustRegister(&timestamppb.Timestamp{}, reflect.TypeFor[time.Time]())
	opts := []converter.Option{
		converter.WithErrorCodecs(anypb.New(anypb.WithRegistry(registry))),
		converter.WithCodecs(tspb.New()),
	}

	type domain struct{ At time.Time }

	type message struct{ At *pbany.Any }

	// The Unix epoch is a valid timestamp with an empty wire encoding.
	epoch := time.Unix(0, 0).UTC()

	var packed message

	_, err := converter.ConvertE(domain{At: epoch}, &packed, opts...)
	require.NoError(t, err)
	require.NotNil(t, packed.At)

	var unpacked domain

	_, err = converter.ConvertE(packed, &unpacked, opts...)
	require.NoError(t, err)
	require.True(t, epoch.Equal(unpacked.At))

	// A zero domain value is absent and packs to a nil Any.
	zero := message{At: mustAny(t, timestamppb.New(epoch))}
	_, err = converter.ConvertE(domain{}, &zero, opts...)
	require.NoError(t, err)
	require.Nil(t, zero.At)
}

func TestNew_InterfaceHoldingAnyUnpacks(t *testing.T) {
	t.Parallel()

	var dst struct{ V *wrapperspb.StringValue }

	_, err := converter.ConvertE(struct{ V any }{mustAny(t, wrapperspb.String("x"))}, &dst,
		converter.WithErrorCodecs(anypb.New()))
	require.NoError(t, err)
	require.Equal(t, "x", dst.V.GetValue())
}

func TestRegistry_RegisterAfterNew(t *testing.T) {
	t.Parallel()

	registry := anypb.NewRegistry().MustRegister(&wrapperspb.StringValue{}, reflect.TypeFor[Note]())
	_ = anypb.New(anypb.WithRegistry(registry))

	err := registry.Register(&wrapperspb.BytesValue{}, reflect.TypeFor[[]byte]())
	require.EqualError(t, err,
		"anypb: register google.protobuf.BytesValue as []uint8: the registry was already passed to New; register every pair first")
	require.Panics(t, func() { registry.MustRegister(&wrapperspb.BytesValue{}, reflect.TypeFor[[]byte]()) })
}

func TestRegistry_ConcurrentRegister(t *testing.T) {
	t.Parallel()

	type a struct{ Value string }

	type b struct{ Value string }

	registry := anypb.NewRegistry()

	var wg sync.WaitGroup

	wg.Go(func() { registry.MustRegister(&wrapperspb.StringValue{}, reflect.TypeFor[a]()) })
	wg.Go(func() { registry.MustRegister(&wrapperspb.BytesValue{}, reflect.TypeFor[b]()) })
	wg.Wait()

	codec := anypb.New(anypb.WithRegistry(registry))

	for range 4 {
		wg.Go(func() {
			var dst *pbany.Any
			if err := codec("v", reflect.ValueOf(a{Value: "x"}), reflect.ValueOf(&dst).Elem(), copyValue); err != nil {
				t.Error(err)
			}
		})
	}

	wg.Wait()
}

func TestNew_PayloadLimits(t *testing.T) {
	t.Parallel()

	src := mustAny(t, wrapperspb.String("hello"))

	var dst *wrapperspb.StringValue

	err := anypb.New(anypb.WithMaxBytes(4))("v", reflect.ValueOf(src), reflect.ValueOf(&dst).Elem(), failNext(t))

	var got anypb.Error
	require.ErrorAs(t, err, &got)
	require.ErrorContains(t, err, "exceeds the 4-byte limit")

	require.NoError(t, anypb.New(anypb.WithMaxBytes(len(src.GetValue())))("v", reflect.ValueOf(src), reflect.ValueOf(&dst).Elem(), failNext(t)))
	require.Equal(t, "hello", dst.GetValue())

	deep := structpb.NewStringValue("leaf")
	for range 10 {
		deep = structpb.NewListValue(&structpb.ListValue{Values: []*structpb.Value{deep}})
	}

	var value *structpb.Value

	err = anypb.New(anypb.WithRecursionLimit(5))("v", reflect.ValueOf(mustAny(t, deep)), reflect.ValueOf(&value).Elem(), failNext(t))
	require.ErrorAs(t, err, &got)

	require.NoError(t, anypb.New()("v", reflect.ValueOf(mustAny(t, deep)), reflect.ValueOf(&value).Elem(), failNext(t)))
}
