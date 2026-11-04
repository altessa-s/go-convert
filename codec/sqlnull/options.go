// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package sqlnull

//go:generate optgen generate

// options holds configuration options for database/sql null type conversion.
type options struct {
	// zeroAsNull ([WithZeroAsNull]) stores a zero T source as an invalid
	// (NULL) value instead of a valid zero. It applies to T sources only; a
	// non-nil *T source is always valid.
	zeroAsNull bool
}
