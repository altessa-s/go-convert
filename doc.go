// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

// Package convert converts structs into structs by matching field names,
// recursing through nested and embedded structs, slices, arrays and maps.
//
//	type User struct{ Name string; Age int }
//	type UserDTO struct{ Name string; Age int }
//
//	dto := convert.Convert(user, &UserDTO{})
//
// Use [Convert] for one-off conversions and [New] to build a reusable,
// concurrency-safe [Converter]. [ConvertE] and [Converter.ConvertE] return
// failures as errors instead of panicking and should be used for untrusted
// input. [ConvertSeq] and [ConvertMapSeq] convert collections lazily.
//
// # Options
//
// Fields are skipped by name with [WithIgnoreFields] or by dotted path with
// [WithIgnorePaths], and renamed with [WithFieldMappings]. Source values are
// applied according to one of three mutually exclusive modes:
// [IgnoreZeroValues], [IgnoreNilValues] and [SparseMerge], the last of which
// treats the source as a partial update, selected with [WithValueMode].
// [WithEmbeddedStructs] fills embedded destination structs from the whole
// source. Custom conversions are
// plugged in as codecs with [WithCodecs] and [WithErrorCodecs]; see the codec
// sub-packages and the separate github.com/altessa-s/go-convert/proto module.
//
// # Errors
//
// Failures are reported as [UnsupportedTypeError], [OverflowError] (with
// [WithOverflowCheck]), [DepthError], or the error returned by a codec. The
// traversal depth is bounded by [WithMaxDepth], so cyclic pointer graphs fail
// with a [DepthError] instead of exhausting the stack.
package convert
