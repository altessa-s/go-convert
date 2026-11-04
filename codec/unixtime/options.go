// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package unixtime

//go:generate optgen generate

// options holds configuration options for Unix time conversion behavior.
type options struct {
	// ignoreZero ([WithIgnoreZero]) skips conversion, leaving the destination
	// unchanged, when the source is nil or zero. For time.Time sources, zero
	// means time.Time{}. For int64 sources, zero means 0.
	ignoreZero bool
	// milliseconds ([WithMilliseconds]) uses millisecond precision instead of
	// seconds: time.Time -> int64 uses UnixMilli(), int64 -> time.Time uses
	// time.UnixMilli().
	milliseconds bool
}
