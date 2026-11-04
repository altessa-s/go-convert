// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package tspb

//go:generate optgen generate

// options holds configuration options for Timestamp conversion behavior.
type options struct {
	// ignoreZero ([WithIgnoreZero]) skips conversion, leaving the destination
	// unchanged, when the source is nil or zero. For Timestamp: nil or
	// {seconds:0, nanos:0}. For time.Time: IsZero(). For int64: 0.
	ignoreZero bool
	// milliseconds ([WithMilliseconds]) uses millisecond precision for
	// Timestamp <-> int64: Timestamp -> int64 uses AsTime().UnixMilli(),
	// int64 -> Timestamp uses time.UnixMilli().
	milliseconds bool
}
