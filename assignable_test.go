// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package convert_test

import (
	"reflect"
	"testing"
	"time"

	converter "github.com/altessa-s/go-convert"
	convcodec "github.com/altessa-s/go-convert/codec"
)

func TestAssignableFieldsWithAndWithoutCodecs(t *testing.T) {
	for _, codecs := range []bool{false, true} {
		name := "direct"
		if codecs {
			name = "after codecs"
		}

		t.Run(name, func(t *testing.T) {
			opts := []converter.Option{converter.WithCopySlices()}
			if codecs {
				opts = append(opts, converter.WithErrorCodecs(func(p string, s, d reflect.Value, next convcodec.CodecHandlerE) error { return next(p, s, d) }))
			}

			t.Run("pointer slice", func(t *testing.T) {
				items := []int{1, 2}
				src := struct{ Items *[]int }{&items}

				var dst struct{ Items *[]int }
				converter.Convert(src, &dst, opts...)

				if dst.Items == src.Items || !reflect.DeepEqual(*dst.Items, items) {
					t.Fatalf("slice pointer was not copied: %+v", dst)
				}

				(*dst.Items)[0] = 9

				if items[0] != 1 {
					t.Fatal("slice storage aliases source")
				}
			})
			t.Run("pointer slice to interface", func(t *testing.T) {
				items := []int{1, 2}
				src := struct{ Items *[]int }{&items}

				var dst struct{ Items any }
				converter.Convert(src, &dst, opts...)

				got, ok := dst.Items.(*[]int)
				if !ok || got == nil {
					t.Fatalf("dst.Items = %v (%T), want non-nil *[]int", dst.Items, dst.Items)
				}

				if got == src.Items {
					t.Fatal("slice pointer aliases source")
				}

				if !reflect.DeepEqual(*got, items) {
					t.Fatalf("slice contents = %v, want %v", *got, items)
				}

				(*got)[0] = 9

				if items[0] != 1 {
					t.Fatal("slice storage aliases source")
				}
			})
			t.Run("nil containers", func(t *testing.T) {
				type item struct{ Secret string }

				type root struct {
					Items []item
					Map   map[string]item
				}

				src := root{}
				dst := root{[]item{{"old"}}, map[string]item{"k": {"old"}}}
				converter.Convert(src, &dst, append(opts, converter.WithIgnorePaths("items.secret", "map.secret"))...)

				if dst.Items != nil || dst.Map != nil {
					t.Fatalf("nil containers = %+v", dst)
				}
			})
			t.Run("dynamic ignore", func(t *testing.T) {
				type item struct{ Secret, Name string }

				type root struct{ Data any }

				src := root{item{"source", "new"}}
				dst := root{item{"keep", "old"}}
				converter.Convert(src, &dst, append(opts, converter.WithIgnorePaths("data.secret"))...)

				if dst.Data != (item{"keep", "new"}) {
					t.Fatalf("dynamic = %+v", dst.Data)
				}
			})
			t.Run("sparse and opaque", func(t *testing.T) {
				type item struct {
					Keep   *string
					Change string
				}

				type root struct {
					Item item
					Time time.Time
				}

				keep := "keep"
				src := root{item{nil, "new"}, time.Unix(123, 0)}
				dst := root{item{&keep, "old"}, time.Unix(456, 0)}
				converter.Convert(src, &dst, append(opts, converter.WithSparseMerge())...)

				if dst.Item != (item{&keep, "new"}) || !dst.Time.Equal(src.Time) {
					t.Fatalf("sparse = %+v", dst)
				}
			})
		})
	}
}
