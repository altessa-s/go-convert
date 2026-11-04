// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package reflectx_test

import (
	"reflect"
	"testing"

	"github.com/altessa-s/go-convert/reflectx"
)

func FuzzIsPrimitive(f *testing.F) {
	// Seed with all valid Kind values (0-26)
	for i := range 27 {
		f.Add(uint8(i))
	}

	f.Fuzz(func(t *testing.T, k uint8) {
		kind := reflect.Kind(k)
		// Should not panic for any kind value
		_ = reflectx.IsPrimitive(kind)
	})
}
