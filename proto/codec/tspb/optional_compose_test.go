// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package tspb_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-atlas-core/types/optional"

	"github.com/altessa-s/go-convert/codec/optionalcodec"
	"github.com/altessa-s/go-convert/proto/codec/durpb"
	"github.com/altessa-s/go-convert/proto/codec/tspb"

	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	converter "github.com/altessa-s/go-convert"
)

// composeOpts mirrors how a service registers the codec chain at the proto
// boundary: optionalcodec first (so it unwraps before the type-specific codecs
// see the value), then tspb / durpb for the inner conversions.
func composeToProto() []converter.Option {
	return []converter.Option{
		converter.WithCodecs(
			optionalcodec.Codec,
			tspb.New(tspb.WithIgnoreZero()),
			durpb.New(durpb.WithIgnoreZero()),
		),
		converter.WithValueMode(converter.IgnoreZeroValues),
	}
}

func composeFromProto() []converter.Option {
	return []converter.Option{
		converter.WithCodecs(
			optionalcodec.Codec,
			tspb.New(tspb.WithIgnoreZero()),
			durpb.New(durpb.WithIgnoreZero()),
		),
	}
}

type composeDomain struct {
	DeleteTime optional.Optional[time.Time]
	TTL        optional.Optional[time.Duration]
}

type composeProto struct {
	DeleteTime *timestamppb.Timestamp
	TTL        *durationpb.Duration
}

func TestCompose_OptionalToProto_Some(t *testing.T) {
	t.Parallel()

	now := time.Unix(1700000000, 0).UTC()

	var pb composeProto
	converter.Convert(&composeDomain{
		DeleteTime: optional.Some(now),
		TTL:        optional.Some(90 * time.Minute),
	}, &pb, composeToProto()...)

	require.NotNil(t, pb.DeleteTime, "Some[time.Time] must map to a proto Timestamp")
	require.True(t, pb.DeleteTime.AsTime().Equal(now))
	require.NotNil(t, pb.TTL, "Some[Duration] must map to a proto Duration")
	require.Equal(t, 90*time.Minute, pb.TTL.AsDuration())
}

func TestCompose_OptionalToProto_None(t *testing.T) {
	t.Parallel()

	var pb composeProto
	converter.Convert(&composeDomain{
		DeleteTime: optional.None[time.Time](),
		TTL:        optional.None[time.Duration](),
	}, &pb, composeToProto()...)

	require.Nil(t, pb.DeleteTime, "None must map to a nil proto Timestamp")
	require.Nil(t, pb.TTL, "None must map to a nil proto Duration")
}

func TestCompose_ProtoToOptional_Present(t *testing.T) {
	t.Parallel()

	now := time.Unix(1700000000, 0).UTC()

	var dom composeDomain
	converter.Convert(&composeProto{
		DeleteTime: timestamppb.New(now),
		TTL:        durationpb.New(90 * time.Minute),
	}, &dom, composeFromProto()...)

	require.True(t, dom.DeleteTime.IsSome(), "non-nil Timestamp must map to Some")
	require.True(t, dom.DeleteTime.Value().Equal(now))
	require.True(t, dom.TTL.IsSome(), "non-nil Duration must map to Some")
	require.Equal(t, 90*time.Minute, dom.TTL.Value())
}

func TestCompose_ProtoToOptional_Nil(t *testing.T) {
	t.Parallel()

	var dom composeDomain
	converter.Convert(&composeProto{
		DeleteTime: nil,
		TTL:        nil,
	}, &dom, composeFromProto()...)

	require.True(t, dom.DeleteTime.IsNone(), "nil Timestamp must map to None")
	require.True(t, dom.TTL.IsNone(), "nil Duration must map to None")
}

// TestCompose_ProtoToOptional_ZeroInnerBecomesNone pins the documented
// presence convention on the W -> Optional[T] path: a non-nil proto whose
// converted inner value is the zero T collapses to None (not Some(zero)),
// distinct from the plain nil case above.
//
// "Zero" here is the struct-zero tested by reflect.Value.IsZero: a zero
// time.Duration (int64 0) qualifies directly, and a Unix-epoch *Timestamp is
// skipped by tspb's WithIgnoreZero so the inner time.Time is never written and
// stays the pristine zero value. (Note: a non-nil Timestamp that carries any
// real instant — including the Go year-1 zero, which tspb materializes with a
// location — is struct-non-zero and stays Some; only true absence maps to None.)
func TestCompose_ProtoToOptional_ZeroInnerBecomesNone(t *testing.T) {
	t.Parallel()

	var dom composeDomain
	converter.Convert(&composeProto{
		DeleteTime: timestamppb.New(time.Unix(0, 0)), // non-nil, Unix epoch
		TTL:        durationpb.New(0),                // non-nil, zero duration
	}, &dom, composeFromProto()...)

	require.True(t, dom.DeleteTime.IsNone(), "non-nil epoch Timestamp must collapse to None")
	require.True(t, dom.TTL.IsNone(), "non-nil zero Duration must collapse to None")
}

// TestCompose_WithoutDownstreamCodec_IsFootgun pins the documented contract
// that composition requires the inner-type codec to be registered after
// optionalcodec. Without tspb the Optional[time.Time] -> *Timestamp path falls
// through to the converter's terminal field-by-field copy, which produces a
// non-nil but semantically empty *Timestamp (Seconds=0, Nanos=0) instead of
// either the source instant or a clean failure. The test exists so that any
// future change to the fallback behavior (e.g. promoting the mismatch to a
// panic, or leaving dst nil) shows up as a deliberate test update rather than
// a silent regression.
func TestCompose_WithoutDownstreamCodec_IsFootgun(t *testing.T) {
	t.Parallel()

	now := time.Unix(1700000000, 0).UTC()

	var pb composeProto
	converter.Convert(&composeDomain{DeleteTime: optional.Some(now)}, &pb,
		converter.WithCodecs(optionalcodec.Codec), // no tspb / durpb
		converter.WithValueMode(converter.IgnoreZeroValues),
	)

	require.NotNil(t, pb.DeleteTime, "current behavior: field-by-field copy materializes a non-nil *Timestamp")
	require.NotEqual(t, now.Unix(), pb.DeleteTime.GetSeconds(),
		"without tspb the inner time.Time is not bridged into the *Timestamp value")
}

type composeOptInt struct {
	V optional.Optional[int]
}

type composeOptInt64 struct {
	V optional.Optional[int64]
}

// TestCompose_OptionalToOptional_DifferentInner pins the documented limit:
// Optional[A] -> Optional[B] with different inner types is not handled by the
// codec and is left to the chain. With no downstream codec for int -> int64
// the field-by-field copy refuses the assignment and the destination stays at
// its zero (None) value. This anchors the limitation called out in the godoc
// so a future change that adds Optional<->Optional bridging shows up as a
// deliberate test update.
func TestCompose_OptionalToOptional_DifferentInner(t *testing.T) {
	t.Parallel()

	var dst composeOptInt64
	converter.Convert(&composeOptInt{V: optional.Some(42)}, &dst,
		converter.WithCodecs(optionalcodec.Codec),
	)

	require.True(t, dst.V.IsNone(), "Optional[int] -> Optional[int64] is not bridged; destination must stay None")
}

// The compose benchmarks run the full codec chain (optionalcodec, tspb and
// durpb), because composition only happens inside the chain.

type benchComposeDomain struct {
	DeleteTime optional.Optional[time.Time]
	TTL        optional.Optional[time.Duration]
}

type benchComposeProto struct {
	DeleteTime *timestamppb.Timestamp
	TTL        *durationpb.Duration
}

var benchComposeToProtoOpts = []converter.Option{
	converter.WithCodecs(
		optionalcodec.Codec,
		tspb.New(tspb.WithIgnoreZero()),
		durpb.New(durpb.WithIgnoreZero()),
	),
	converter.WithValueMode(converter.IgnoreZeroValues),
}

var benchComposeFromProtoOpts = []converter.Option{
	converter.WithCodecs(
		optionalcodec.Codec,
		tspb.New(tspb.WithIgnoreZero()),
		durpb.New(durpb.WithIgnoreZero()),
	),
}

func BenchmarkCompose_OptionalToProto_Some(b *testing.B) {
	src := &benchComposeDomain{
		DeleteTime: optional.Some(time.Unix(1700000000, 0).UTC()),
		TTL:        optional.Some(90 * time.Minute),
	}

	for b.Loop() {
		var pb benchComposeProto
		converter.Convert(src, &pb, benchComposeToProtoOpts...)
	}
}

func BenchmarkCompose_OptionalToProto_None(b *testing.B) {
	src := &benchComposeDomain{
		DeleteTime: optional.None[time.Time](),
		TTL:        optional.None[time.Duration](),
	}

	for b.Loop() {
		var pb benchComposeProto
		converter.Convert(src, &pb, benchComposeToProtoOpts...)
	}
}

func BenchmarkCompose_ProtoToOptional_Present(b *testing.B) {
	src := &benchComposeProto{
		DeleteTime: timestamppb.New(time.Unix(1700000000, 0).UTC()),
		TTL:        durationpb.New(90 * time.Minute),
	}

	for b.Loop() {
		var dom benchComposeDomain
		converter.Convert(src, &dom, benchComposeFromProtoOpts...)
	}
}

func BenchmarkCompose_ProtoToOptional_Nil(b *testing.B) {
	src := &benchComposeProto{DeleteTime: nil, TTL: nil}

	for b.Loop() {
		var dom benchComposeDomain
		converter.Convert(src, &dom, benchComposeFromProtoOpts...)
	}
}
