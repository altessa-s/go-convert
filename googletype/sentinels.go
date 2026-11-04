// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package googletype

import (
	"strconv"
)

// IndexError reports the element of a repeated field a list decoder
// rejected, so the caller can name it ("field[i]") on its own path.
type IndexError struct {
	// Index is the zero-based position of the rejected element.
	Index int
	// Err is the element's conversion error.
	Err error
}

// Error implements error.
func (e *IndexError) Error() string {
	return "element " + strconv.Itoa(e.Index) + ": " + e.Err.Error()
}

// Unwrap returns the element's conversion error.
func (e *IndexError) Unwrap() error { return e.Err }
