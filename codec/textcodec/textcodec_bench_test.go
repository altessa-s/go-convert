// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package textcodec

import (
	"net/netip"
	"reflect"
	"testing"
	"uuid"
)

func benchNext(string, reflect.Value, reflect.Value) error { return nil }

func BenchmarkNew_UUIDToString(b *testing.B) {
	codec := New()
	src := reflect.ValueOf(uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8"))

	for b.Loop() {
		var dst string

		_ = codec("ID", src, reflect.ValueOf(&dst).Elem(), benchNext)
	}
}

func BenchmarkNew_StringToAddr(b *testing.B) {
	codec := New()
	src := reflect.ValueOf("192.0.2.1")

	for b.Loop() {
		var dst netip.Addr

		_ = codec("Addr", src, reflect.ValueOf(&dst).Elem(), benchNext)
	}
}

func BenchmarkNew_PassThrough(b *testing.B) {
	codec := New()
	src := reflect.ValueOf("hello")

	for b.Loop() {
		var dst string

		_ = codec("Name", src, reflect.ValueOf(&dst).Elem(), benchNext)
	}
}
