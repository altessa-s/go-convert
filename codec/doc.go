// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

// Package codec provides the codec contract and the chain execution engine
// of the converter. The examples and the codec packages of this repository
// import it as convcodec, which leaves the name codec free for variables
// holding a codec:
//
//	import convcodec "github.com/altessa-s/go-convert/codec"
//
// A [Codec] handles a value or delegates to the next codec of a [Set]; the
// converter's built-in conversion ends the chain. [CodecE] and
// [CodecHandlerE] add error-returning middleware that receives the full
// source field path: [Set.AddError] registers it and [Set.RunError] runs it
// before the [Codec] chain and the terminal handler. Registered with
// convert.WithErrorCodecs, a CodecE error propagates unchanged through
// convert.ConvertE, preserving its identity.
//
// Codec implementations live in sub-packages, such as
// [github.com/altessa-s/go-convert/codec/unixtime] and
// [github.com/altessa-s/go-convert/codec/textcodec], and in the nested
// modules github.com/altessa-s/go-convert/proto,
// github.com/altessa-s/go-convert/googletype and
// github.com/altessa-s/go-convert/codec/optionalcodec.
//
// Example:
//
//	codecSet := convcodec.NewCodecsSet(unixtime.New())
//	codecSet.Run("fieldName", srcVal, dstVal, defaultHandler)
package codec
