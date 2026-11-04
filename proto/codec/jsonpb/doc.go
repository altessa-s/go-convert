// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

// Package jsonpb provides codecs for converting between the structpb
// messages and their plain Go counterparts.
//
// [New] returns a codec for json.RawMessage <-> structpb.Struct. [NewE]
// returns an error codec that also converts structpb.Value <-> any,
// structpb.ListValue <-> []any and structpb.Struct <-> map[string]any, and
// reports values google.protobuf.Value cannot hold, such as NaN, as errors.
//
// Both pointer and non-pointer variants are supported. Nil sources are handled
// gracefully and can be skipped with [WithIgnoreNil].
//
// Example:
//
//	codec := jsonpb.New()
//	converter.Convert(src, &dst, converter.WithCodecs(codec))
//
//	codecE := jsonpb.NewE()
//	_, err := converter.ConvertE(src, &dst, converter.WithErrorCodecs(codecE))
package jsonpb
