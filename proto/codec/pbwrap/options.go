// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package pbwrap

//go:generate optgen generate

// options holds configuration options for protobuf wrapper conversion behavior.
// These options control when conversions should be skipped based on zero values.
type options struct {
	// ignoreZeroValues ([WithIgnoreZeroValues]) skips conversion when the
	// source primitive value is zero. This avoids creating protobuf wrapper
	// objects for zero values, which reduces unnecessary allocations.
	ignoreZeroValues bool
	// ignoreZeroWrappers ([WithIgnoreZeroWrappers]) skips conversion when the
	// protobuf wrapper contains a zero value, keeping the distinction between
	// nil wrappers and wrappers containing zero values.
	ignoreZeroWrappers bool
	registry           *Registry `opt:"-"` // Internal registry for conversions
}
