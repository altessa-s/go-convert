// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package jsoncodec_test

import (
	"encoding/json/jsontext"
	"reflect"
	"testing"

	"github.com/altessa-s/go-convert/codec/jsoncodec"
)

func nopNext(string, reflect.Value, reflect.Value) error { return nil }

func BenchmarkNew_Unmarshal(b *testing.B) {
	codec := jsoncodec.New()
	src := reflect.ValueOf(jsontext.Value(`{"x":1,"y":2}`))
	dst := reflect.New(reflect.TypeFor[point]()).Elem()

	b.ReportAllocs()

	for b.Loop() {
		_ = codec("Payload", src, dst, nopNext)
	}
}

func BenchmarkNew_Marshal(b *testing.B) {
	codec := jsoncodec.New()
	src := reflect.ValueOf(&point{X: 1, Y: 2}).Elem()
	dst := reflect.New(reflect.TypeFor[jsontext.Value]()).Elem()

	b.ReportAllocs()

	for b.Loop() {
		_ = codec("Payload", src, dst, nopNext)
	}
}

func BenchmarkNew_PassThrough(b *testing.B) {
	codec := jsoncodec.New()
	src := reflect.ValueOf("hello")
	dst := reflect.New(reflect.TypeFor[string]()).Elem()

	b.ReportAllocs()

	for b.Loop() {
		_ = codec("Name", src, dst, nopNext)
	}
}
