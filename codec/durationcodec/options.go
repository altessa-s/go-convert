// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package durationcodec

import "time"

//go:generate optgen generate

// options holds configuration options for duration conversion behavior.
type options struct {
	// unit ([WithUnit]) is the duration of one numeric unit, such as
	// time.Second or time.Millisecond. A time.Duration converts to a number
	// as d / unit and a number n to a time.Duration as n * unit. The default
	// is time.Second; non-positive values are ignored.
	unit time.Duration `optgen:"default=time.Second"`
}
