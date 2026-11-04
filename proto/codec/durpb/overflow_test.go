// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package durpb_test

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/altessa-s/go-convert/proto/codec/durpb"

	converter "github.com/altessa-s/go-convert"
)

func TestDurationOverflowSaturates(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		seconds int64
		nanos   int32
		want    time.Duration
	}{
		{name: "maximum", seconds: 9223372036, nanos: 854775807, want: math.MaxInt64},
		{name: "minimum", seconds: -9223372036, nanos: -854775808, want: math.MinInt64},
		{name: "positive nanos overflow", seconds: 9223372036, nanos: 854775808, want: math.MaxInt64},
		{name: "negative nanos overflow", seconds: -9223372036, nanos: -854775809, want: math.MinInt64},
		{name: "positive seconds overflow", seconds: 315576000000, want: math.MaxInt64},
		{name: "negative seconds overflow", seconds: -315576000000, want: math.MinInt64},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			pb := &durationpb.Duration{Seconds: tc.seconds, Nanos: tc.nanos}
			require.NoError(t, pb.CheckValid())
			src := durProto{TTL: pb, Grace: pb, Splits: []*durationpb.Duration{pb}, Meta: map[string]*durationpb.Duration{"ttl": pb}}

			var dst durDomain
			converter.Convert(src, &dst, converter.WithCodecs(durpb.New()))

			require.Equal(t, tc.want, dst.TTL)
			require.NotNil(t, dst.Grace)
			require.Equal(t, tc.want, *dst.Grace)
			require.Equal(t, []time.Duration{tc.want}, dst.Splits)
			require.Equal(t, map[string]time.Duration{"ttl": tc.want}, dst.Meta)
		})
	}
}
