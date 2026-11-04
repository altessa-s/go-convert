// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package converttest_test

import (
	"testing"

	"github.com/altessa-s/go-convert/converttest"
)

func TestRequireConvert(t *testing.T) {
	t.Parallel()

	type source struct{ Value int }

	type destination struct{ Value int64 }

	got := converttest.RequireConvert(t, source{Value: 42}, &destination{})
	if got.Value != 42 {
		t.Fatalf("value = %d, want 42", got.Value)
	}
}
