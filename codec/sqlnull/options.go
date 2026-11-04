// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package sqlnull

//go:generate optgen generate

// options holds configuration options for database/sql null type conversion.
type options struct {
	// zeroAsNull ([WithZeroAsNull]) stores a zero value as an invalid (NULL)
	// value instead of a valid zero: a zero T source, a non-pointer source
	// the chain converts to a zero T, and a valid null source whose value
	// converts to a zero T2. A non-nil pointer source is always valid; a nil
	// one is always NULL.
	zeroAsNull bool
}
