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
// JSON text is parsed and rendered with encoding/json/v2, which rejects
// duplicate object names and invalid UTF-8. [NewE] limits the nesting of
// lists and maps ([WithMaxDepth]) and rejects values that contain themselves.
//
// Both pointer and non-pointer variants are supported. Nil sources are handled
// gracefully and can be skipped with [WithIgnoreNil].
//
// Example:
//
//	codec := jsonpb.New()
//	convert.Convert(src, &dst, convert.WithCodecs(codec))
//
//	codecE := jsonpb.NewE()
//	_, err := convert.ConvertE(src, &dst, convert.WithErrorCodecs(codecE))
package jsonpb
