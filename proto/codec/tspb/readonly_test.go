// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package tspb_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/altessa-s/go-convert/proto/codec/tspb"
)

// The codec declines destinations reflect does not allow to write, so its
// unsafe writes never modify read-only memory, and leaves mistyped
// destinations to the next handler.
func TestNewDeclinesUnwritableDestinations(t *testing.T) {
	t.Parallel()

	type stamp time.Time

	type holder struct {
		hiddenTime time.Time
		hiddenTS   timestamppb.Timestamp
		Named      stamp
	}

	h := &holder{}
	codec := tspb.New()
	ts := timestamppb.New(time.Unix(1700000000, 5))
	at := time.Unix(1700000000, 5)

	tests := []struct {
		name string
		src  reflect.Value
		dst  reflect.Value
	}{
		{name: "unexported time field", src: reflect.ValueOf(ts), dst: reflect.ValueOf(h).Elem().Field(0)},
		{name: "unexported timestamp field", src: reflect.ValueOf(at), dst: reflect.ValueOf(h).Elem().Field(1)},
		{name: "unexported timestamp field from int64", src: reflect.ValueOf(int64(1)), dst: reflect.ValueOf(h).Elem().Field(1)},
		{name: "time not addressable", src: reflect.ValueOf(ts), dst: reflect.ValueOf(time.Time{})},
		{name: "named time type", src: reflect.ValueOf(ts), dst: reflect.ValueOf(h).Elem().Field(2)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			called := false

			codec("At", tt.src, tt.dst, func(string, reflect.Value, reflect.Value) { called = true })
			require.True(t, called, "the codec must delegate")
		})
	}

	require.True(t, h.hiddenTime.IsZero())
	require.Zero(t, h.hiddenTS.GetSeconds())
	require.True(t, time.Time(h.Named).IsZero())
}

// A pointer destination that is not addressable itself still points at
// writable memory.
func TestNewWritesThroughPointerDestination(t *testing.T) {
	t.Parallel()

	fail := func(string, reflect.Value, reflect.Value) { require.Fail(t, "next must not be called") }

	var at time.Time

	tspb.New()("At", reflect.ValueOf(timestamppb.New(time.Unix(1700000000, 5))), reflect.ValueOf(&at), fail)
	require.Equal(t, time.Unix(1700000000, 5).UTC(), at.UTC())

	var ts timestamppb.Timestamp

	tspb.New()("At", reflect.ValueOf(time.Unix(1700000000, 5)), reflect.ValueOf(&ts), fail)
	require.Equal(t, int64(1700000000), ts.GetSeconds())
	require.Equal(t, int32(5), ts.GetNanos())
}
