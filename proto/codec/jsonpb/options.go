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
	// is identical, but the intent is explicit.
	ignoreNil bool
}
