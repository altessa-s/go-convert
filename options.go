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

// options holds the converter configuration. Options tagged manual are
// written by hand below because they update several fields together.
type options struct {
	// codecs are conversion middleware run in registration order for every
	// field; the built-in conversion ends the chain.
	codecs []convcodec.Codec `opt:"Codecs" optgen:"append"`
	// errorCodecs run before codecs, receive the full dotted source path and
	// abort the conversion with the first returned error.
	errorCodecs      []convcodec.CodecE `opt:"ErrorCodecs" optgen:"append"`
	ignoreZeroValues bool               `optgen:"manual"`
	ignoreNilValues  bool               `optgen:"manual"`
	sparseMerge      bool               `optgen:"manual"`
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
	fieldMappings                  map[string]string `opt:"FieldMappings"`
	handleEmbeddedStructs          bool              `optgen:"manual"`
	alwaysInitializeEmbeddedStruct bool              `opt:"-" optgen:"manual"`
	// overflowCheck makes lossy narrowing numeric conversions fail with an
	// [OverflowError] instead of truncating.
	overflowCheck bool
	// maxDepth bounds the nesting depth; exceeding it fails with a
	// [DepthError]. Non-positive values select [DefaultMaxDepth].
	maxDepth int `optgen:"default=DefaultMaxDepth"`
}

// config derives the immutable engine configuration from the applied options.
func (o *options) config() engine.Config {
	cfg := engine.Config{
		Codecs:                convcodec.NewCodecsSet(o.codecs...).AddError(o.errorCodecs...),
		IgnoreFields:          make(map[string]struct{}, len(o.ignoreFields)),
		IgnorePaths:           make(map[string]struct{}, len(o.ignorePaths)),
		FieldMappings:         make(map[string]string, len(o.fieldMappings)),
		IgnoreZeroValues:      o.ignoreZeroValues,
		IgnoreNilValues:       o.ignoreNilValues,
		SparseMerge:           o.sparseMerge,
		CopySlices:            o.copySlices,
		HandleEmbeddedStructs: o.handleEmbeddedStructs,
		InitEmbeddedStructs:   o.alwaysInitializeEmbeddedStruct,
		OverflowCheck:         o.overflowCheck,
		MaxDepth:              o.maxDepth,
	}
	for _, field := range o.ignoreFields {
		cfg.IgnoreFields[strings.ToLower(field)] = struct{}{}
	}
	for _, path := range o.ignorePaths {
		if path = strings.Trim(path, "."); path != "" {
			cfg.IgnorePaths[strings.ToLower(path)] = struct{}{}
		}
	}
	for source, destination := range o.fieldMappings {
		cfg.FieldMappings[strings.ToLower(source)] = strings.ToLower(destination)
	}
	return cfg
}

// WithIgnoreZeroValues configures the converter to skip zero values during conversion.
// When enabled, fields with zero values (empty strings, 0 for numbers, nil pointers, etc.)
// in the source struct will not be copied to the destination struct.
// This option is mutually exclusive with WithIgnoreNilValues and WithSparseMerge.
func WithIgnoreZeroValues() Option {
	return func(o *options) {
		o.ignoreZeroValues = true
		o.ignoreNilValues = false
		o.sparseMerge = false
	}
}

// WithIgnoreNilValues configures the converter to skip nil values during conversion.
// When enabled, fields with nil values (nil pointers, nil slices, nil maps)
// in the source struct will not be copied to the destination struct.
// This option is mutually exclusive with WithIgnoreZeroValues and WithSparseMerge.
func WithIgnoreNilValues() Option {
	return func(o *options) {
		o.ignoreNilValues = true
		o.ignoreZeroValues = false
		o.sparseMerge = false
	}
}

// WithHandleEmbeddedStructs enables processing of embedded (anonymous) structs during conversion.
// When enabled, the converter will recursively process embedded struct fields,
// allowing flattening of nested structures or proper handling of composition patterns.
//
// The initializeEmbeddedStruct parameter controls initialization behavior:
// - true: Always initialize embedded struct pointers, even if they would be empty
// - false: Only initialize embedded struct pointers if they contain non-zero values
func WithHandleEmbeddedStructs(initializeEmbeddedStruct bool) Option {
	return func(o *options) {
		o.handleEmbeddedStructs = true
		o.alwaysInitializeEmbeddedStruct = initializeEmbeddedStruct
	}
}

// WithSparseMerge configures the converter for sparse-merge (partial-update)
// semantics, applying a sparse source onto an existing destination. For each
// source field:
//   - nil pointer/slice/map -> skipped (destination left unchanged)
//   - non-nil scalar pointer (including a pointer to the zero value) -> written
//     to the destination, so an explicit empty value clears it
//   - non-nil nested struct pointer with at least one set field -> merged
//     recursively
//   - non-nil nested struct pointer with no set fields -> destination field
//     zeroed, clearing the whole nested object
//
// The semantics apply at every nesting depth. Nil sources are skipped just like
// WithIgnoreNilValues, so this option is mutually exclusive with
// WithIgnoreZeroValues and supersedes WithIgnoreNilValues.
//
// The source must express optionality through pointers:
//   - A non-pointer scalar field is always "present" and is copied as-is,
//     including its zero value — it overwrites the destination. Use pointer
//     fields for every optional scalar.
//   - A non-pointer nested struct field is likewise always "present": it is
//     merged field-by-field and never triggers the clear-on-empty rule, so an
//     untouched zero struct merges nothing instead of wiping the destination.
//
// Structs without exported fields (time.Time and similar opaque types) cannot
// be merged field-by-field and are assigned to the destination wholesale.
func WithSparseMerge() Option {
	return func(o *options) {
		o.sparseMerge = true
		o.ignoreNilValues = true
		o.ignoreZeroValues = false
	}
}
