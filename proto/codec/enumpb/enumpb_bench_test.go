// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package enumpb_test

import (
	"reflect"
	"testing"

	"github.com/altessa-s/go-convert/proto/codec/enumpb"

	"google.golang.org/protobuf/types/known/typepb"
)

func noopNext(string, reflect.Value, reflect.Value) error { return nil }

func BenchmarkNew_EnumToString(b *testing.B) {
	codec := enumpb.MustNew(enumpb.WithTrimEnumPrefix(), enumpb.WithCase(enumpb.CaseLower))
	src := reflect.ValueOf(typepb.Field_CARDINALITY_REPEATED)

	var dst string

	dstVal := reflect.ValueOf(&dst).Elem()

	b.ReportAllocs()

	for b.Loop() {
		_ = codec("Cardinality", src, dstVal, noopNext)
	}
}

func BenchmarkNew_StringToEnum(b *testing.B) {
	codec := enumpb.MustNew(enumpb.WithTrimEnumPrefix(), enumpb.WithCase(enumpb.CaseLower))
	src := reflect.ValueOf("repeated")

	var dst typepb.Field_Cardinality

	dstVal := reflect.ValueOf(&dst).Elem()

	b.ReportAllocs()

	for b.Loop() {
		_ = codec("Cardinality", src, dstVal, noopNext)
	}
}

func BenchmarkNew_PassThrough(b *testing.B) {
	codec := enumpb.MustNew()
	src := reflect.ValueOf("hello")

	var dst string

	dstVal := reflect.ValueOf(&dst).Elem()

	b.ReportAllocs()

	for b.Loop() {
		_ = codec("Name", src, dstVal, noopNext)
	}
}

func BenchmarkNewMapping_EnumToGo(b *testing.B) {
	codec := enumpb.MustNewMapping(cardinalities)
	src := reflect.ValueOf(typepb.Field_CARDINALITY_REPEATED)

	var dst cardinality

	dstVal := reflect.ValueOf(&dst).Elem()

	b.ReportAllocs()

	for b.Loop() {
		_ = codec("Cardinality", src, dstVal, noopNext)
	}
}

func BenchmarkNewMapping_GoToEnum(b *testing.B) {
	codec := enumpb.MustNewMapping(cardinalities)
	src := reflect.ValueOf(cardinalityRepeated)

	var dst typepb.Field_Cardinality

	dstVal := reflect.ValueOf(&dst).Elem()

	b.ReportAllocs()

	for b.Loop() {
		_ = codec("Cardinality", src, dstVal, noopNext)
	}
}
