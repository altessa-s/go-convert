// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package convert

//go:generate optgen generate

import (
	"strings"

	"github.com/altessa-s/go-convert/internal/engine"

	convcodec "github.com/altessa-s/go-convert/codec"
)

// DefaultMaxDepth is the nesting limit used unless [WithMaxDepth] sets one.
const DefaultMaxDepth = 10_000

// ValueMode selects which source values the converter applies to the
// destination. Values other than the constants below act as
// [OverwriteValues].
type ValueMode uint8

const (
	// OverwriteValues applies every source value, zero and nil values
	// included. It is the default.
	OverwriteValues ValueMode = iota
	// IgnoreZeroValues skips source values that are the zero value of their
	// type (empty strings, 0, false, nil pointers, slices and maps, zero
	// structs), leaving the destination unchanged.
	IgnoreZeroValues
	// IgnoreNilValues skips nil pointers, slices and maps, leaving the
	// destination unchanged.
	IgnoreNilValues
	// SparseMerge applies a sparse source onto an existing destination as a
	// partial update. For each source field:
	//   - nil pointer/slice/map -> skipped (destination left unchanged)
	//   - non-nil scalar pointer (including a pointer to the zero value) ->
	//     written to the destination, so an explicit empty value clears it
	//   - non-nil nested struct pointer with at least one set field -> merged
	//     recursively
	//   - non-nil nested struct pointer with no set fields -> destination
	//     field zeroed, clearing the whole nested object
	//
	// The semantics apply at every nesting depth. The source must express
	// optionality through pointers:
	//   - A non-pointer scalar field is always "present" and is copied as-is,
	//     including its zero value — it overwrites the destination. Use
	//     pointer fields for every optional scalar.
	//   - A non-pointer nested struct field is likewise always "present": it
	//     is merged field-by-field and never triggers the clear-on-empty
	//     rule, so an untouched zero struct merges nothing instead of wiping
	//     the destination.
	//
	// Structs without exported fields (time.Time and similar opaque types)
	// cannot be merged field-by-field and are assigned to the destination
	// wholesale.
	SparseMerge
)

// EmbeddedStructs selects how embedded (anonymous) destination structs are
// filled. Values other than the constants below act as [EmbeddedByName].
type EmbeddedStructs uint8

const (
	// EmbeddedByName converts an embedded struct like any other field, by
	// its type name; an embedded source struct without a destination field
	// of that name is flattened into the destination. It is the default.
	EmbeddedByName EmbeddedStructs = iota
	// EmbeddedFill additionally converts the whole source into every
	// embedded destination struct, which allows flattening and composition
	// in both directions. A nil embedded pointer is allocated only when the
	// converted struct is not the zero value.
	EmbeddedFill
	// EmbeddedFillAlways is [EmbeddedFill] that always allocates nil
	// embedded pointers, even when the converted struct is the zero value.
	EmbeddedFillAlways
)

// options holds the converter configuration.
type options struct {
	// codecs are conversion middleware run in registration order for every
	// field, slice or array element and map value the converter visits,
	// including values whose type is assignable to the destination; the
	// built-in conversion ends the chain. A value assigned by the built-in
	// conversion is not walked further, so the codecs see an assignable
	// struct, slice or map as a whole. Map keys reach the codecs only when
	// Go can neither assign nor convert them. Without codecs, assignable
	// values keep the direct assignment fast path.
	codecs []convcodec.Codec `opt:"Codecs" optgen:"append"`
	// errorCodecs run before codecs, receive the full dotted source path and
	// abort the conversion with the first returned error.
	errorCodecs []convcodec.CodecE `opt:"ErrorCodecs" optgen:"append"`
	// valueMode selects which source values are applied; see [ValueMode].
	// The modes exclude each other, so the last one set wins.
	valueMode ValueMode
	// copySlices gives assignable slices their own backing array instead of
	// aliasing the source. A nil slice stays nil; an empty slice stays empty.
	copySlices bool
	// ignoreFields skips source fields with these names at every level,
	// matched case-insensitively.
	ignoreFields []string `opt:"IgnoreFields" optgen:"append" optval:"notrim"`
	// ignorePaths skips source fields by dotted path from the root, such as
	// "address.street", matched case-insensitively.
	ignorePaths []string `opt:"IgnorePaths" optgen:"append" optval:"notrim"`
	// fieldMappings renames source fields (keys) to destination fields
	// (values), matched case-insensitively.
	fieldMappings map[string]string `opt:"FieldMappings"`
	// embeddedStructs selects how embedded destination structs are filled;
	// see [EmbeddedStructs].
	embeddedStructs EmbeddedStructs
	// overflowCheck makes lossy narrowing numeric conversions fail with an
	// [OverflowError] instead of truncating.
	overflowCheck bool
	// maxDepth bounds the nesting depth; exceeding it fails with a
	// [DepthError]. Non-positive values select [DefaultMaxDepth].
	maxDepth int `optgen:"default=DefaultMaxDepth"`
}

// config derives the immutable engine configuration from the applied options.
// Empty sets and mappings stay nil, which the engine reads as empty, so a
// configuration without them allocates nothing for them.
func (o *options) config() engine.Config {
	cfg := engine.Config{
		IgnoreZeroValues: o.valueMode == IgnoreZeroValues,
		// Sparse merge skips nil sources like IgnoreNilValues.
		IgnoreNilValues:       o.valueMode == IgnoreNilValues || o.valueMode == SparseMerge,
		SparseMerge:           o.valueMode == SparseMerge,
		CopySlices:            o.copySlices,
		HandleEmbeddedStructs: o.embeddedStructs == EmbeddedFill || o.embeddedStructs == EmbeddedFillAlways,
		InitEmbeddedStructs:   o.embeddedStructs == EmbeddedFillAlways,
		OverflowCheck:         o.overflowCheck,
		MaxDepth:              o.maxDepth,
	}
	if len(o.codecs) > 0 || len(o.errorCodecs) > 0 {
		cfg.Codecs = convcodec.NewCodecsSet(o.codecs...).AddError(o.errorCodecs...)
	}

	if len(o.ignoreFields) > 0 {
		cfg.IgnoreFields = make(map[string]struct{}, len(o.ignoreFields))
		for _, field := range o.ignoreFields {
			cfg.IgnoreFields[strings.ToLower(field)] = struct{}{}
		}
	}

	for _, path := range o.ignorePaths {
		if path = strings.Trim(path, "."); path != "" {
			if cfg.IgnorePaths == nil {
				cfg.IgnorePaths = make(map[string]struct{}, len(o.ignorePaths))
			}

			cfg.IgnorePaths[strings.ToLower(path)] = struct{}{}
		}
	}

	if len(o.fieldMappings) > 0 {
		cfg.FieldMappings = make(map[string]string, len(o.fieldMappings))
		for source, destination := range o.fieldMappings {
			cfg.FieldMappings[strings.ToLower(source)] = strings.ToLower(destination)
		}
	}

	return cfg
}
