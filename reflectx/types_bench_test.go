// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package reflectx_test

import (
	"reflect"
	"testing"

	"github.com/altessa-s/go-convert/reflectx"
)

func BenchmarkIndirectType_NonPointer(b *testing.B) {
	t := reflect.TypeFor[testStruct]()
	for b.Loop() {
		_ = reflectx.IndirectType(t)
	}
}

func BenchmarkIndirectType_Pointer(b *testing.B) {
	t := reflect.TypeFor[*testStruct]()
	for b.Loop() {
		_ = reflectx.IndirectType(t)
	}
}

func BenchmarkIndirectType_DoublePointer(b *testing.B) {
	t := reflect.TypeFor[**testStruct]()
	for b.Loop() {
		_ = reflectx.IndirectType(t)
	}
}

func BenchmarkIsPrimitive(b *testing.B) {
	for b.Loop() {
		_ = reflectx.IsPrimitive(reflect.Int)
	}
}

func BenchmarkConvertValue_IntToString(b *testing.B) {
	v, t := reflect.ValueOf(int64(1234567)), reflect.TypeFor[string]()
	for b.Loop() {
		_, _ = reflectx.ConvertValue(v, t)
	}
}

func BenchmarkConvertValue_IntWidening(b *testing.B) {
	v, t := reflect.ValueOf(int32(42)), reflect.TypeFor[int64]()
	for b.Loop() {
		_, _ = reflectx.ConvertValue(v, t)
	}
}
