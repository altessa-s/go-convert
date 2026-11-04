// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package convert_test

import (
	"reflect"
	"testing"

	converter "github.com/altessa-s/go-convert"
	convcodec "github.com/altessa-s/go-convert/codec"
)

// benchUser is a typical service-layer model with scalars, a nested struct
// pointer, a slice and a map.
type benchUser struct {
	ID      int64
	Age     int32
	Name    string
	Email   string
	Score   float64
	Active  bool
	Address *benchAddress
	Tags    []string
	Labels  map[string]string
}

type benchAddress struct {
	Street, City, Zip string
	Number            int
}

// benchUserDTO mirrors benchUser with distinct types, so fields convert
// through the primitive and struct paths instead of being assigned.
type benchUserDTO struct {
	ID      int
	Age     int64
	Name    string
	Email   string
	Score   float32
	Active  bool
	Address *benchAddressDTO
	Tags    []string
	Labels  map[string]string
}

type benchAddressDTO struct {
	Street, City, Zip string
	Number            int64
}

// newBenchUser returns a fully populated benchUser.
func newBenchUser(i int) benchUser {
	return benchUser{
		ID: int64(i), Age: 42, Name: "name", Email: "mail@example.com", Score: 1.5, Active: true,
		Address: &benchAddress{Street: "street", City: "city", Zip: "zip", Number: i},
		Tags:    []string{"a", "b"},
		Labels:  map[string]string{"k": "v"},
	}
}

// BenchmarkConvert_OneOff_NoOptions measures the package-level Convert of a
// realistic model without options, the most common one-off call.
func BenchmarkConvert_OneOff_NoOptions(b *testing.B) {
	src := newBenchUser(1)

	b.ReportAllocs()

	for b.Loop() {
		var dst benchUserDTO
		converter.Convert(&src, &dst)
	}
}

// BenchmarkConvert_OneOff_ConvertE is BenchmarkConvert_OneOff through ConvertE.
func BenchmarkConvert_OneOff_ConvertE(b *testing.B) {
	src := BenchSmall{A: 1, B: 2, C: 3}

	b.ReportAllocs()

	for b.Loop() {
		var dst BenchSmall
		if _, err := converter.ConvertE(src, &dst); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkConvert_OneOff_WithOptions measures the package-level Convert with
// a few common options.
func BenchmarkConvert_OneOff_WithOptions(b *testing.B) {
	src := newBenchUser(1)

	b.ReportAllocs()

	for b.Loop() {
		var dst benchUserDTO
		converter.Convert(&src, &dst,
			converter.WithIgnoreFields("Email"),
			converter.WithOverflowCheck(),
			converter.WithValueMode(converter.IgnoreNilValues),
		)
	}
}

// BenchmarkConvert_Model measures a reused converter on a realistic model.
func BenchmarkConvert_Model(b *testing.B) {
	src := newBenchUser(1)
	conv := converter.New[*benchUser, *benchUserDTO]()

	b.Run("pointer_src", func(b *testing.B) {
		b.ReportAllocs()

		for b.Loop() {
			var dst benchUserDTO
			conv.Convert(&src, &dst)
		}
	})
	b.Run("value_src", func(b *testing.B) {
		conv := converter.New[benchUser, *benchUserDTO]()

		b.ReportAllocs()

		for b.Loop() {
			var dst benchUserDTO
			conv.Convert(src, &dst)
		}
	})
	b.Run("parallel", func(b *testing.B) {
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				var dst benchUserDTO
				conv.Convert(&src, &dst)
			}
		})
	})
}

// BenchmarkConvert_SliceOfStructs1000 converts 1000 structs into a distinct
// element type, at the root and as a struct field.
func BenchmarkConvert_SliceOfStructs1000(b *testing.B) {
	users := make([]benchUser, 1000)
	for i := range users {
		users[i] = newBenchUser(i)
	}

	small := make([]BenchSmall, 1000)
	for i := range small {
		small[i] = BenchSmall{A: i, B: i, C: i}
	}

	b.Run("root_models", func(b *testing.B) {
		conv := converter.New[[]benchUser, *[]benchUserDTO]()

		b.ReportAllocs()

		for b.Loop() {
			var dst []benchUserDTO
			conv.Convert(users, &dst)
		}
	})
	b.Run("root_small", func(b *testing.B) {
		conv := converter.New[[]BenchSmall, *[]benchSmallDTO]()

		b.ReportAllocs()

		for b.Loop() {
			var dst []benchSmallDTO
			conv.Convert(small, &dst)
		}
	})
	b.Run("field_pointers", func(b *testing.B) {
		conv := converter.New[benchPtrSliceHolder, *benchSliceDTO]()
		src := benchPtrSliceHolder{PtrItems: small}

		b.ReportAllocs()

		for b.Loop() {
			var dst benchSliceDTO
			conv.Convert(src, &dst)
		}
	})
}

type benchMapHolder struct {
	Items map[string]BenchSmall
	Ptrs  map[int]*BenchSmall
}

type benchMapDTO struct {
	Items map[string]benchSmallDTO
	Ptrs  map[int64]*benchSmallDTO
}

// BenchmarkConvert_MapOfStructs converts maps whose values are structs of a
// distinct type, by value and by pointer.
func BenchmarkConvert_MapOfStructs(b *testing.B) {
	src := benchMapHolder{Items: make(map[string]BenchSmall, 100), Ptrs: make(map[int]*BenchSmall, 100)}
	for i := range 100 {
		src.Items[string(rune('a'+i%26))+string(rune('a'+i/26))] = BenchSmall{A: i, B: i, C: i}
		src.Ptrs[i] = &BenchSmall{A: i, B: i, C: i}
	}

	conv := converter.New[*benchMapHolder, *benchMapDTO]()

	b.ReportAllocs()

	for b.Loop() {
		var dst benchMapDTO
		conv.Convert(&src, &dst)
	}
}

type benchDeep struct {
	Name  string
	Inner *benchDeepInner
}

type benchDeepInner struct {
	Value int
	Leaf  *benchDeepLeaf
}

type benchDeepLeaf struct {
	A, B int32
	S    *string
}

type benchDeepDTO struct {
	Name  string
	Inner *benchDeepInnerDTO
}

type benchDeepInnerDTO struct {
	Value int64
	Leaf  **benchDeepLeafDTO
}

type benchDeepLeafDTO struct {
	A, B int64
	S    *string
}

// BenchmarkConvert_NestedPointers converts a chain of struct pointers into
// distinct types, allocating every destination level.
func BenchmarkConvert_NestedPointers(b *testing.B) {
	s := "leaf"
	src := benchDeep{Name: "n", Inner: &benchDeepInner{Value: 1, Leaf: &benchDeepLeaf{A: 1, B: 2, S: &s}}}
	conv := converter.New[*benchDeep, *benchDeepDTO]()

	b.ReportAllocs()

	for b.Loop() {
		var dst benchDeepDTO
		conv.Convert(&src, &dst)
	}
}

// BenchmarkConvert_ConvertSeq iterates ConvertSeq over 1000 elements.
func BenchmarkConvert_ConvertSeq(b *testing.B) {
	small := make([]BenchSmall, 1000)
	for i := range small {
		small[i] = BenchSmall{A: i, B: i, C: i}
	}

	conv := converter.New[BenchSmall, benchSmallDTO]()

	b.ReportAllocs()

	for b.Loop() {
		for dst := range conv.ConvertSeq(small) {
			_ = dst
		}
	}
}

// BenchmarkConvert_Codecs measures a realistic model through one and two
// passthrough codecs and through a passthrough error codec.
func BenchmarkConvert_Codecs(b *testing.B) {
	src := newBenchUser(1)
	passthrough := func(field string, s, d reflect.Value, next convcodec.CodecHandler) {
		next(field, s, d)
	}
	passthroughE := func(path string, s, d reflect.Value, next convcodec.CodecHandlerE) error {
		return next(path, s, d)
	}

	for _, bc := range []struct {
		name string
		opts []converter.Option
	}{
		{"one_codec", []converter.Option{converter.WithCodecs(passthrough)}},
		{"two_codecs", []converter.Option{converter.WithCodecs(passthrough, passthrough)}},
		{"error_codec", []converter.Option{converter.WithErrorCodecs(passthroughE)}},
	} {
		b.Run(bc.name, func(b *testing.B) {
			conv := converter.New[*benchUser, *benchUserDTO](bc.opts...)

			b.ReportAllocs()

			for b.Loop() {
				var dst benchUserDTO
				conv.Convert(&src, &dst)
			}
		})
	}
}

// BenchmarkConvert_IgnorePaths measures static ignore paths on a realistic
// model, which disable the planned fast paths.
func BenchmarkConvert_IgnorePaths(b *testing.B) {
	src := newBenchUser(1)
	conv := converter.New[*benchUser, *benchUserDTO](converter.WithIgnorePaths("address.zip", "email"))

	b.ReportAllocs()

	for b.Loop() {
		var dst benchUserDTO
		conv.Convert(&src, &dst)
	}
}

// BenchmarkConvert_IgnoreFieldsMappings measures ignored field names and
// field mappings, which use a per-engine plan cache.
func BenchmarkConvert_IgnoreFieldsMappings(b *testing.B) {
	src := newBenchUser(1)
	conv := converter.New[*benchUser, *benchUserDTO](
		converter.WithIgnoreFields("Tags"),
		converter.WithFieldMappings(map[string]string{"Email": "Name"}),
	)

	b.ReportAllocs()

	for b.Loop() {
		var dst benchUserDTO
		conv.Convert(&src, &dst)
	}
}
