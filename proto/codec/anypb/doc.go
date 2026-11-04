// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

// Package anypb provides an error codec for conversion between
// google.protobuf.Any and concrete protobuf messages or Go domain types.
//
// An Any unpacks into a protobuf message field of the type its type URL
// names, and a message packs into an Any. A [Registry] maps message types to
// Go domain types: the codec unpacks the Any into the registered message and
// delegates the message <-> domain conversion to the rest of the codec chain,
// so a registered timestamppb.Timestamp <-> time.Time pair converts through
// tspb. Interface destinations, such as any, receive the registered domain
// value or the message resolved by [WithResolver].
//
// Type URL mismatches and unknown types are reported as errors. Register a
// codec for every registered pair: the chain does not report whether one
// converted it. Interface fields holding values the codec cannot pack are
// passed to the rest of the chain.
//
// Example:
//
//	registry := anypb.NewRegistry().
//	    MustRegister(&timestamppb.Timestamp{}, reflect.TypeFor[time.Time]())
//	_, err := convert.ConvertE(src, &dst,
//	    convert.WithErrorCodecs(anypb.New(anypb.WithRegistry(registry))),
//	    convert.WithCodecs(tspb.New()))
package anypb
