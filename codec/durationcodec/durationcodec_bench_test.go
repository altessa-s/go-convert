// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package durationcodec

import (
	"reflect"
	"testing"
	"time"
)

func benchNext(string, reflect.Value, reflect.Value) error { return nil }

func BenchmarkNew_DurationToString(b *testing.B) {
	codec := New()
	src := reflect.ValueOf(90 * time.Minute)

	for b.Loop() {
		var dst string

		_ = codec("Timeout", src, reflect.ValueOf(&dst).Elem(), benchNext)
	}
}

func BenchmarkNew_StringToDuration(b *testing.B) {
	codec := New()
	src := reflect.ValueOf("1h30m")

	for b.Loop() {
		var dst time.Duration

		_ = codec("Timeout", src, reflect.ValueOf(&dst).Elem(), benchNext)
	}
}

func BenchmarkNew_FloatToDuration(b *testing.B) {
	codec := New()
	src := reflect.ValueOf(1.5)

	for b.Loop() {
		var dst time.Duration

		_ = codec("Timeout", src, reflect.ValueOf(&dst).Elem(), benchNext)
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
