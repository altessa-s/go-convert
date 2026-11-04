// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package durpb_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/altessa-s/go-convert/proto/codec/durpb"

	"google.golang.org/protobuf/types/known/durationpb"
)

func BenchmarkNew_DurationToGo(b *testing.B) {
	codec := durpb.New()
	pb := durationpb.New(90*time.Minute + 500*time.Millisecond)
	src := reflect.ValueOf(pb).Elem()
	handler := func(_ string, _, _ reflect.Value) {}

	b.ReportAllocs()

	for b.Loop() {
		var dst time.Duration
		dstVal := reflect.ValueOf(&dst).Elem()
		codec("Ttl", src, dstVal, handler)
	}
}

func BenchmarkNew_GoToDuration(b *testing.B) {
	codec := durpb.New()
	d := 90*time.Minute + 500*time.Millisecond
	src := reflect.ValueOf(d)
	handler := func(_ string, _, _ reflect.Value) {}

	b.ReportAllocs()

	for b.Loop() {
		var dst durationpb.Duration
		dstVal := reflect.ValueOf(&dst).Elem()
		codec("Ttl", src, dstVal, handler)
	}
}

func BenchmarkNew_PassThrough(b *testing.B) {
	codec := durpb.New()
	src := reflect.ValueOf("hello")
	handler := func(_ string, _, _ reflect.Value) {}

	b.ReportAllocs()

	for b.Loop() {
		var dst string
		dstVal := reflect.ValueOf(&dst).Elem()
		codec("Name", src, dstVal, handler)
	}
}
