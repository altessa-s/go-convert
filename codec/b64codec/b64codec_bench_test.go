// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package b64codec

import (
	"reflect"
	"testing"
)

func benchNext(string, reflect.Value, reflect.Value) error { return nil }

func BenchmarkNew_Encode(b *testing.B) {
	codec := New()
	src := reflect.ValueOf([]byte("the quick brown fox jumps over the lazy dog"))

	for b.Loop() {
		var dst string

		_ = codec("Data", src, reflect.ValueOf(&dst).Elem(), benchNext)
	}
}

func BenchmarkNew_Decode(b *testing.B) {
	codec := New()
	src := reflect.ValueOf("dGhlIHF1aWNrIGJyb3duIGZveCBqdW1wcyBvdmVyIHRoZSBsYXp5IGRvZw==")

	for b.Loop() {
		var dst []byte

		_ = codec("Data", src, reflect.ValueOf(&dst).Elem(), benchNext)
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
