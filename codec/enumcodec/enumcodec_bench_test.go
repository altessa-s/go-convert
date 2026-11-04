// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package enumcodec_test

import (
	"reflect"
	"testing"

	"github.com/altessa-s/go-convert/codec/enumcodec"
)

func nopNext(string, reflect.Value, reflect.Value) error { return nil }

func BenchmarkNew_Format(b *testing.B) {
	codec := enumcodec.New(colorNames)
	src := reflect.ValueOf(colorGreen)
	dst := reflect.New(reflect.TypeFor[string]()).Elem()

	b.ReportAllocs()
	for b.Loop() {
		_ = codec("Color", src, dst, nopNext)
	}
}

func BenchmarkNew_Parse(b *testing.B) {
	codec := enumcodec.New(colorNames, enumcodec.WithCaseInsensitive())
	src := reflect.ValueOf("Green")
	dst := reflect.New(reflect.TypeFor[color]()).Elem()

	b.ReportAllocs()
	for b.Loop() {
		_ = codec("Color", src, dst, nopNext)
	}
}

func BenchmarkNewInt_Parse(b *testing.B) {
	codec := enumcodec.NewInt(colorNumbers)
	src := reflect.ValueOf(int32(20))
	dst := reflect.New(reflect.TypeFor[color]()).Elem()

	b.ReportAllocs()
	for b.Loop() {
		_ = codec("Color", src, dst, nopNext)
	}
}

func BenchmarkNew_PassThrough(b *testing.B) {
	codec := enumcodec.New(colorNames)
	src := reflect.ValueOf("hello")
	dst := reflect.New(reflect.TypeFor[string]()).Elem()

	b.ReportAllocs()
	for b.Loop() {
		_ = codec("Name", src, dst, nopNext)
	}
}
