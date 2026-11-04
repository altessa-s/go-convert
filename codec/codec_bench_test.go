// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package codec

import (
	"reflect"
	"testing"
)

func BenchmarkSet_Run_Chain(b *testing.B) {
	c1 := Codec(func(_ string, src, dst reflect.Value, next CodecHandler) { next("", src, dst) })
	c2 := Codec(func(_ string, src, dst reflect.Value, next CodecHandler) { next("", src, dst) })
	c3 := Codec(func(_ string, src, dst reflect.Value, next CodecHandler) { next("", src, dst) })
	s := NewCodecsSet(c1, c2, c3)

	src := reflect.ValueOf(42)
	dst := reflect.ValueOf(0)
	handler := func(_ string, _, _ reflect.Value) {}

	for b.Loop() {
		s.Run("field", src, dst, handler)
	}
}

func BenchmarkRunError(b *testing.B) {
	pass := func(path string, src, dst reflect.Value, next CodecHandlerE) error { return next(path, src, dst) }
	set := NewCodecsSet().AddError(pass, pass)
	src := reflect.ValueOf(1)
	dst := reflect.New(reflect.TypeFor[int]()).Elem()
	finish := func(string, reflect.Value, reflect.Value) error { return nil }

	b.ReportAllocs()

	for b.Loop() {
		_ = set.RunError("a.b", "b", src, dst, finish)
	}
}
