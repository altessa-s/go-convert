// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package jsonpb

import (
	"encoding/json"
	"reflect"
	"testing"

	"google.golang.org/protobuf/types/known/structpb"
)

func BenchmarkNew_RawMessageToStruct(b *testing.B) {
	raw := json.RawMessage(`{"key":"value","num":42}`)
	codec := New()
	src := reflect.ValueOf(&raw)
	handler := func(_ string, _, _ reflect.Value) {}

	for b.Loop() {
		var dst *structpb.Struct
		dstVal := reflect.ValueOf(&dst).Elem()
		codec("Metadata", src, dstVal, handler)
	}
}

func BenchmarkNew_StructToRawMessage(b *testing.B) {
	pb, _ := structpb.NewStruct(map[string]any{"key": "value", "num": float64(42)})
	codec := New()
	src := reflect.ValueOf(pb)
	handler := func(_ string, _, _ reflect.Value) {}

	for b.Loop() {
		var dst *json.RawMessage
		dstVal := reflect.ValueOf(&dst).Elem()
		codec("Metadata", src, dstVal, handler)
	}
}

func BenchmarkNew_PassThrough(b *testing.B) {
	codec := New()
	src := reflect.ValueOf("hello")
	handler := func(_ string, _, _ reflect.Value) {}

	for b.Loop() {
		var dst string
		dstVal := reflect.ValueOf(&dst).Elem()
		codec("Name", src, dstVal, handler)
	}
}

func BenchmarkNewE_MapToStruct(b *testing.B) {
	m := map[string]any{"key": "value", "num": 42, "list": []any{true, nil, 1.5}}
	codec := NewE()
	src := reflect.ValueOf(m)
	next := func(string, reflect.Value, reflect.Value) error { return nil }

	b.ReportAllocs()
	for b.Loop() {
		var dst *structpb.Struct
		_ = codec("Metadata", src, reflect.ValueOf(&dst).Elem(), next)
	}
}

func BenchmarkNewE_StructToMap(b *testing.B) {
	pb, _ := structpb.NewStruct(map[string]any{"key": "value", "num": 42, "list": []any{true, nil, 1.5}})
	codec := NewE()
	src := reflect.ValueOf(pb)
	next := func(string, reflect.Value, reflect.Value) error { return nil }

	b.ReportAllocs()
	for b.Loop() {
		var dst map[string]any
		_ = codec("Metadata", src, reflect.ValueOf(&dst).Elem(), next)
	}
}

func BenchmarkNewE_PassThrough(b *testing.B) {
	codec := NewE()
	src := reflect.ValueOf("hello")
	next := func(string, reflect.Value, reflect.Value) error { return nil }

	b.ReportAllocs()
	for b.Loop() {
		var dst string
		_ = codec("Name", src, reflect.ValueOf(&dst).Elem(), next)
	}
}
