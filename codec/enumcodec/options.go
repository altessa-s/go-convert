// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package enumcodec

//go:generate optgen generate

// options holds configuration options for enum conversion behavior.
type options struct {
	// caseInsensitive ([WithCaseInsensitive]) matches source strings against
	// the table ignoring case. Formatting always writes the table spelling.
	// It applies to [New] only; [NewInt] panics when given it.
	caseInsensitive bool
	// unspecified ([WithUnspecified]) maps the zero value of the enum type to
	// "" or 0 and back, even when the table has no entry for it.
	unspecified bool
	// unknownAsZero ([WithUnknownAsZero]) maps values missing from the table
	// to the zero value of the destination instead of returning an
	// [UnknownValueError].
	unknownAsZero bool
}
