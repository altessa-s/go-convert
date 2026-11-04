// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package jsonpb

//go:generate optgen generate

// options holds configuration options for JSON <-> structpb.Struct conversion behavior.
type options struct {
	// ignoreNil ([WithIgnoreNil]) skips conversion when the source pointer is
	// nil. Without this option, a nil source still marks the conversion as
	// handled (destination stays zero-value). With this option the behavior
	// is identical, but the intent is explicit. For [NewE], a nil source
	// sets the destination to its zero value unless this option is set, in
	// which case the destination is left untouched.
	ignoreNil bool
	// maxDepth ([WithMaxDepth]) limits how deeply lists and maps may nest in
	// a value converted by [NewE]; deeper values, and lists or maps that
	// contain themselves, fail with a [ValueError] instead of exhausting the
	// stack. Values below 1 select [DefaultMaxDepth].
	maxDepth int `optgen:"default=DefaultMaxDepth"`
}
