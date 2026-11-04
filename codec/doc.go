// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

// Package convcodec provides core types for custom conversion codecs and a chain execution engine.
//
// [CodecE] and [CodecHandlerE] add error-returning middleware with full field
// paths. [Set.AddError] registers this chain; [Set.RunError] runs it before the
// legacy Codec chain and terminal handler. [Set.Run] remains unchanged.
// When registered with converter.WithErrorCodecs, CodecE errors propagate
// unchanged through converter.ConvertE, preserving error identity.
//
// Individual codec implementations live in subpackages:
//
//   - [github.com/altessa-s/go-convert/proto/codec/pbwrap] — protobuf wrapper ↔ primitive conversion
//   - [github.com/altessa-s/go-convert/proto/codec/oneof] — struct ↔ protobuf oneof conversion
//   - [github.com/altessa-s/go-convert/codec/mapslice] — map keys/values → slice conversion
//   - [github.com/altessa-s/go-convert/codec/unixtime] — time.Time ↔ int64 Unix timestamp conversion
//   - [github.com/altessa-s/go-convert/proto/codec/jsonpb] — json.RawMessage ↔ structpb.Struct conversion
//   - [github.com/altessa-s/go-convert/proto/codec/tspb] — timestamppb.Timestamp ↔ time.Time / int64 conversion
//   - [github.com/altessa-s/go-convert/proto/codec/durpb] — durationpb.Duration ↔ time.Duration conversion
//
// Example usage:
//
//	codecSet := convcodec.NewCodecsSet(pbwrap.New(), unixtime.New())
//	codecSet.Run("fieldName", srcVal, dstVal, defaultHandler)
package convcodec
