// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package sqlnull_test

import (
	"database/sql"
	"reflect"
	"testing"

	"github.com/altessa-s/go-convert/codec/sqlnull"
)

func nopNext(string, reflect.Value, reflect.Value) error { return nil }

func BenchmarkNew_NullToValue(b *testing.B) {
	codec := sqlnull.New()
	src := reflect.ValueOf(sql.NullString{String: "a", Valid: true})
	dst := reflect.New(reflect.TypeFor[string]()).Elem()

	b.ReportAllocs()
	for b.Loop() {
		_ = codec("Name", src, dst, nopNext)
	}
}

func BenchmarkNew_PointerToNull(b *testing.B) {
	codec := sqlnull.New()
	v := int64(7)
	src := reflect.ValueOf(&v)
	dst := reflect.New(reflect.TypeFor[sql.Null[int64]]()).Elem()

	b.ReportAllocs()
	for b.Loop() {
		_ = codec("Age", src, dst, nopNext)
	}
}

func BenchmarkNew_PassThrough(b *testing.B) {
	codec := sqlnull.New()
	src := reflect.ValueOf("hello")
	dst := reflect.New(reflect.TypeFor[string]()).Elem()

	b.ReportAllocs()
	for b.Loop() {
		_ = codec("Name", src, dst, nopNext)
	}
}
