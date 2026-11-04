// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package unixtime_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-convert/codec/unixtime"
)

// The codec declines destinations reflect does not allow to write, so its
// unsafe write never modifies read-only memory, and leaves mistyped
// destinations to the next handler.
func TestNewDeclinesUnwritableDestinations(t *testing.T) {
	t.Parallel()

	type stamp time.Time

	type holder struct {
		hidden time.Time
		Named  stamp
	}

	h := &holder{}
	codec := unixtime.New()

	tests := []struct {
		name string
		dst  reflect.Value
	}{
		{name: "unexported field", dst: reflect.ValueOf(h).Elem().Field(0)},
		{name: "not addressable", dst: reflect.ValueOf(time.Time{})},
		{name: "named time type", dst: reflect.ValueOf(h).Elem().Field(1)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			called := false

			codec("At", reflect.ValueOf(int64(1700000000)), tt.dst, func(string, reflect.Value, reflect.Value) { called = true })
			require.True(t, called, "the codec must delegate")
		})
	}

	require.True(t, h.hidden.IsZero())
	require.True(t, time.Time(h.Named).IsZero())
}

// A pointer destination that is not addressable itself still points at
// writable memory.
func TestNewWritesThroughPointerDestination(t *testing.T) {
	t.Parallel()

	var out time.Time

	unixtime.New()("At", reflect.ValueOf(int64(1700000000)), reflect.ValueOf(&out), func(string, reflect.Value, reflect.Value) {
		require.Fail(t, "next must not be called")
	})
	require.Equal(t, int64(1700000000), out.Unix())
}
