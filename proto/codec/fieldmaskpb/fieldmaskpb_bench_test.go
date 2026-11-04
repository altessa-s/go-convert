// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package fieldmaskpb_test

import (
	"reflect"
	"testing"

	"google.golang.org/protobuf/types/known/typepb"

	"github.com/altessa-s/go-convert/proto/codec/fieldmaskpb"

	fmpb "google.golang.org/protobuf/types/known/fieldmaskpb"
)

func BenchmarkNew_MaskToPaths(b *testing.B) {
	codec := fieldmaskpb.New()
	src := reflect.ValueOf(mask("name", "kind", "options"))
	handler := func(_ string, _, _ reflect.Value) {}

	b.ReportAllocs()

	for b.Loop() {
		var dst []string
		codec("UpdateMask", src, reflect.ValueOf(&dst).Elem(), handler)
	}
}

func BenchmarkNew_PathsToMask(b *testing.B) {
	codec := fieldmaskpb.New()
	src := reflect.ValueOf([]string{"name", "kind", "options"})
	handler := func(_ string, _, _ reflect.Value) {}

	b.ReportAllocs()

	for b.Loop() {
		var dst *fmpb.FieldMask
		codec("UpdateMask", src, reflect.ValueOf(&dst).Elem(), handler)
	}
}

func BenchmarkNew_PassThrough(b *testing.B) {
	codec := fieldmaskpb.New()
	src := reflect.ValueOf("hello")
	handler := func(_ string, _, _ reflect.Value) {}

	b.ReportAllocs()

	for b.Loop() {
		var dst string
		codec("Name", src, reflect.ValueOf(&dst).Elem(), handler)
	}
}

func BenchmarkIgnorePaths(b *testing.B) {
	m := mask("name", "json_name", "options")
	msg := &typepb.Field{}

	b.ReportAllocs()

	for b.Loop() {
		if _, err := fieldmaskpb.IgnorePaths(m, msg); err != nil {
			b.Fatal(err)
		}
	}
}
