// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package textcodec

//go:generate optgen generate

// options holds configuration options for text conversion behavior.
type options struct {
	// parseEmpty ([WithParseEmpty]) passes empty source text to UnmarshalText
	// instead of setting the destination to its zero value. Types that reject
	// empty input, such as time.Time, then report an error.
	parseEmpty bool
}
