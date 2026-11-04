// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package anypb_test

import (
	"reflect"
	"testing"

	"github.com/altessa-s/go-convert/proto/codec/anypb"

	"google.golang.org/protobuf/types/known/wrapperspb"

	pbany "google.golang.org/protobuf/types/known/anypb"
)

func nopNext(string, reflect.Value, reflect.Value) error { return nil }

func BenchmarkNew_Pack(b *testing.B) {
	codec := anypb.New()
	src := reflect.ValueOf(wrapperspb.String("hello"))

	b.ReportAllocs()

	for b.Loop() {
		var dst *pbany.Any

		_ = codec("Body", src, reflect.ValueOf(&dst).Elem(), nopNext)
	}
}

func BenchmarkNew_Unpack(b *testing.B) {
	codec := anypb.New()
	src := reflect.ValueOf(mustAny(b, wrapperspb.String("hello")))

	b.ReportAllocs()

	for b.Loop() {
		var dst *wrapperspb.StringValue

		_ = codec("Body", src, reflect.ValueOf(&dst).Elem(), nopNext)
	}
}

func BenchmarkNew_UnpackInterface(b *testing.B) {
	codec := anypb.New()
	src := reflect.ValueOf(mustAny(b, wrapperspb.String("hello")))

	b.ReportAllocs()

	for b.Loop() {
		var dst any

		_ = codec("Body", src, reflect.ValueOf(&dst).Elem(), nopNext)
	}
}

func BenchmarkNew_PassThrough(b *testing.B) {
	codec := anypb.New()
	src := reflect.ValueOf("hello")

	b.ReportAllocs()

	for b.Loop() {
		var dst string

		_ = codec("Name", src, reflect.ValueOf(&dst).Elem(), nopNext)
	}
}
