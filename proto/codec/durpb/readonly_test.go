// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package durpb_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/altessa-s/go-convert/proto/codec/durpb"
)

// The codec declines destinations reflect does not allow to write, so its
// unsafe write never modifies read-only memory, and leaves mistyped
// destinations to the next handler.
func TestNewDeclinesUnwritableDestinations(t *testing.T) {
	t.Parallel()

	type span durationpb.Duration

	type holder struct {
		hidden durationpb.Duration
		Named  span
	}

	h := &holder{}
	codec := durpb.New()

	tests := []struct {
		name string
		dst  reflect.Value
	}{
		{name: "unexported field", dst: reflect.ValueOf(h).Elem().Field(0)},
		{name: "not addressable", dst: reflect.ValueOf(durationpb.Duration{})},
		{name: "named duration type", dst: reflect.ValueOf(h).Elem().Field(1)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			called := false

			codec("TTL", reflect.ValueOf(90*time.Second), tt.dst, func(string, reflect.Value, reflect.Value) { called = true })
			require.True(t, called, "the codec must delegate")
		})
	}

	require.Zero(t, h.hidden.GetSeconds())
	require.Zero(t, h.Named.Seconds)
}

// A pointer destination that is not addressable itself still points at
// writable memory.
func TestNewWritesThroughPointerDestination(t *testing.T) {
	t.Parallel()

	var out durationpb.Duration

	durpb.New()("TTL", reflect.ValueOf(90*time.Second), reflect.ValueOf(&out), func(string, reflect.Value, reflect.Value) {
		require.Fail(t, "next must not be called")
	})
	require.Equal(t, int64(90), out.GetSeconds())
}
