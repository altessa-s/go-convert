// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package pbwrap_test

import (
	"reflect"
	"testing"

	"github.com/altessa-s/go-convert/proto/codec/pbwrap"

	"google.golang.org/protobuf/types/known/wrapperspb"
)

func BenchmarkNew_WrapString(b *testing.B) {
	codec := pbwrap.New()
	src := reflect.ValueOf("hello")

	var dst *wrapperspb.StringValue

	dstVal := reflect.ValueOf(&dst).Elem()
	nop := func(_ string, _, _ reflect.Value) {}

	b.ReportAllocs()

	for b.Loop() {
		codec("Name", src, dstVal, nop)
	}
}

func BenchmarkNew_UnwrapString(b *testing.B) {
	codec := pbwrap.New()
	src := reflect.ValueOf(&wrapperspb.StringValue{Value: "hello"})

	var dst string

	dstVal := reflect.ValueOf(&dst).Elem()
	nop := func(_ string, _, _ reflect.Value) {}

	b.ReportAllocs()

	for b.Loop() {
		codec("Name", src, dstVal, nop)
	}
}
