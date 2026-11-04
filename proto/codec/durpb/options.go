// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package durpb

//go:generate optgen generate

// options holds configuration options for Duration conversion behavior.
type options struct {
	// ignoreZero ([WithIgnoreZero]) skips conversion, leaving the destination
	// unchanged, when the source is nil or zero. For Duration: nil or
	// {seconds:0, nanos:0}. For time.Duration: 0.
	ignoreZero bool
}
