// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package converttest

import (
	"testing"

	"github.com/altessa-s/go-convert"
)

// RequireConvert converts src into dst and fails the test if conversion returns an error.
func RequireConvert[T, U any](t testing.TB, src T, dst U, opts ...convert.Option) U {
	t.Helper()

	out, err := convert.ConvertE(src, dst, opts...)
	if err != nil {
		t.Fatalf("convert value: %v", err)
	}

	return out
}
