// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package tspb_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-convert/proto/codec/tspb"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestMillisecondsBeforeEpoch(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		millis int64
	}{
		{name: "negative fraction", millis: -1},
		{name: "negative second", millis: -1000},
		{name: "negative second and fraction", millis: -1001},
		{name: "epoch", millis: 0},
		{name: "positive fraction", millis: 1},
		{name: "positive second and fraction", millis: 1001},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			convert := tspb.New(tspb.WithMilliseconds())
			var dst *timestamppb.Timestamp
			convert("CreatedAt", reflect.ValueOf(tc.millis), reflect.ValueOf(&dst).Elem(),
				func(_ string, _, _ reflect.Value) { require.FailNow(t, "unexpected fallback") })

			require.NotNil(t, dst)
			require.NoError(t, dst.CheckValid())
			want := timestamppb.New(time.UnixMilli(tc.millis))
			require.Equal(t, want.Seconds, dst.Seconds)
			require.Equal(t, want.Nanos, dst.Nanos)

			var back int64
			convert("CreatedAt", reflect.ValueOf(dst), reflect.ValueOf(&back).Elem(),
				func(_ string, _, _ reflect.Value) { require.FailNow(t, "unexpected fallback") })
			require.Equal(t, tc.millis, back)
		})
	}
}
