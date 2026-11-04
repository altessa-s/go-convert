// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package convert_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	converter "github.com/altessa-s/go-convert"
)

func TestIgnoreZeroValuesKeepsDestination(t *testing.T) {
	src := Source{Name: "", Age: 0}
	dst := Dest{Name: "original", Age: 42}

	converter.Convert(src, &dst, converter.WithValueMode(converter.IgnoreZeroValues))
	require.Equal(t, "original", dst.Name)
	require.Equal(t, 42, dst.Age)
}

func TestIgnoreZeroValuesSkipsOnlyZeroFields(t *testing.T) {
	src := &SrcUser{Name: "Bob", Age: 0}
	dst := &DstUser{Name: "Original", Age: 25}

	converter.Convert(src, dst, converter.WithValueMode(converter.IgnoreZeroValues))

	require.Equal(t, "Bob", dst.Name)
	require.Equal(t, 25, dst.Age, "zero should be ignored")
}

func TestIgnoreNilValuesKeepsDestinationPointer(t *testing.T) {
	src := Source{Ptr: nil, Name: "test"}
	dst := Dest{Name: "original"}
	pv := 99
	dst.Ptr = &pv

	converter.Convert(src, &dst, converter.WithValueMode(converter.IgnoreNilValues))
	require.NotNil(t, dst.Ptr)
	require.Equal(t, 99, *dst.Ptr)
}

// Types exercising SparseMerge: a sparse source merged onto an existing
// destination. Source and destination are distinct types so every field flows
// through the struct-merge path rather than a direct assignment.
type MergeDeepSource struct{ Code *string }

type MergeDeepDest struct{ Code *string }

type MergeNestedSource struct {
	Field *string
	Label *string
	Deep  *MergeDeepSource
}

type MergeNestedDest struct {
	Field *string
	Label string
	Deep  *MergeDeepDest
}

type MergeSource struct {
	Name   *string
	Title  *string
	Nested *MergeNestedSource
	Tags   []string
}

type MergeDest struct {
	Name   *string
	Title  *string
	Nested *MergeNestedDest
	Tags   []string
}

func TestSparseMerge(t *testing.T) {
	sp := func(s string) *string { return &s }

	tests := []struct {
		name string
		dst  MergeDest
		src  MergeSource
		want MergeDest
	}{
		{
			name: "scalar set, scalar clear, nil scalar skipped",
			dst:  MergeDest{Name: sp("Old"), Title: sp("M")},
			src:  MergeSource{Name: sp("New"), Title: sp("")},
			want: MergeDest{Name: sp("New"), Title: sp("")},
		},
		{
			name: "nested merge preserves untouched siblings",
			dst:  MergeDest{Nested: &MergeNestedDest{Field: sp("a"), Label: "b", Deep: &MergeDeepDest{Code: sp("c")}}},
			src:  MergeSource{Nested: &MergeNestedSource{Field: sp("x")}},
			want: MergeDest{Nested: &MergeNestedDest{Field: sp("x"), Label: "b", Deep: &MergeDeepDest{Code: sp("c")}}},
		},
		{
			name: "present-but-empty nested struct clears whole field",
			dst:  MergeDest{Nested: &MergeNestedDest{Field: sp("a"), Label: "b"}},
			src:  MergeSource{Nested: &MergeNestedSource{}},
			want: MergeDest{Nested: nil},
		},
		{
			name: "nested struct absent from update is preserved",
			dst:  MergeDest{Nested: &MergeNestedDest{Field: sp("a")}},
			src:  MergeSource{Name: sp("X")},
			want: MergeDest{Name: sp("X"), Nested: &MergeNestedDest{Field: sp("a")}},
		},
		{
			name: "clear one sub-field, update another, preserve the rest",
			dst:  MergeDest{Nested: &MergeNestedDest{Field: sp("a"), Label: "b", Deep: &MergeDeepDest{Code: sp("c")}}},
			src:  MergeSource{Nested: &MergeNestedSource{Field: sp(""), Label: sp("new")}},
			want: MergeDest{Nested: &MergeNestedDest{Field: sp(""), Label: "new", Deep: &MergeDeepDest{Code: sp("c")}}},
		},
		{
			name: "clears a deeply nested object while preserving its parent",
			dst:  MergeDest{Nested: &MergeNestedDest{Field: sp("a"), Deep: &MergeDeepDest{Code: sp("c")}}},
			src:  MergeSource{Nested: &MergeNestedSource{Deep: &MergeDeepSource{}}},
			want: MergeDest{Nested: &MergeNestedDest{Field: sp("a"), Deep: nil}},
		},
		{
			name: "nil slice is skipped",
			dst:  MergeDest{Tags: []string{"a", "b"}},
			src:  MergeSource{Name: sp("X")},
			want: MergeDest{Name: sp("X"), Tags: []string{"a", "b"}},
		},
		{
			name: "non-nil slice replaces",
			dst:  MergeDest{Tags: []string{"a", "b"}},
			src:  MergeSource{Tags: []string{"c"}},
			want: MergeDest{Tags: []string{"c"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dst := tt.dst
			converter.Convert(&tt.src, &dst, converter.WithValueMode(converter.SparseMerge))
			require.Equal(t, tt.want, dst)
		})
	}
}

// Types exercising SparseMerge when source and destination share the same
// type: the assignability fast path must not replace nested structs wholesale.
type MergeSameNested struct {
	Field *string
	Label *string
}

type MergeSame struct {
	Name   *string
	Nested *MergeSameNested
	Stamp  *time.Time
}

func TestSparseMergeSameType(t *testing.T) {
	sp := func(s string) *string { return &s }
	now := time.Now()

	tests := []struct {
		name string
		dst  MergeSame
		src  MergeSame
		want MergeSame
	}{
		{
			name: "nested merge preserves untouched siblings",
			dst:  MergeSame{Nested: &MergeSameNested{Field: sp("a"), Label: sp("b")}},
			src:  MergeSame{Nested: &MergeSameNested{Field: sp("x")}},
			want: MergeSame{Nested: &MergeSameNested{Field: sp("x"), Label: sp("b")}},
		},
		{
			name: "present-but-empty nested struct clears whole field",
			dst:  MergeSame{Nested: &MergeSameNested{Field: sp("a"), Label: sp("b")}},
			src:  MergeSame{Nested: &MergeSameNested{}},
			want: MergeSame{Nested: nil},
		},
		{
			name: "opaque struct without exported fields is assigned wholesale",
			dst:  MergeSame{Name: sp("Old")},
			src:  MergeSame{Stamp: &now},
			want: MergeSame{Name: sp("Old"), Stamp: &now},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dst := tt.dst
			converter.Convert(&tt.src, &dst, converter.WithValueMode(converter.SparseMerge))
			require.Equal(t, tt.want, dst)
		})
	}
}

func TestSparseMergeDoesNotAliasSource(t *testing.T) {
	sp := func(s string) *string { return &s }

	dst := MergeSame{Nested: &MergeSameNested{Field: sp("a"), Label: sp("b")}}
	src := MergeSame{Nested: &MergeSameNested{Field: sp("x")}}

	converter.Convert(&src, &dst, converter.WithValueMode(converter.SparseMerge))

	require.NotSame(t, src.Nested, dst.Nested,
		"destination must not alias the source nested struct")
}

// Types exercising SparseMerge with non-pointer nested structs: they carry
// no presence signal, so a zero value merges normally and never clears.
type MergeValMetaSource struct{ Note *string }

type MergeValMetaDest struct{ Note *string }

type MergeValSource struct {
	Name *string
	Meta MergeValMetaSource
}

type MergeValDest struct {
	Name *string
	Meta MergeValMetaDest
}

func TestSparseMergeNonPointerNestedStruct(t *testing.T) {
	sp := func(s string) *string { return &s }

	tests := []struct {
		name string
		dst  MergeValDest
		src  MergeValSource
		want MergeValDest
	}{
		{
			name: "untouched zero non-pointer struct preserves destination",
			dst:  MergeValDest{Meta: MergeValMetaDest{Note: sp("important")}},
			src:  MergeValSource{Name: sp("X")},
			want: MergeValDest{Name: sp("X"), Meta: MergeValMetaDest{Note: sp("important")}},
		},
		{
			name: "non-pointer struct with set field merges",
			dst:  MergeValDest{Meta: MergeValMetaDest{Note: sp("old")}},
			src:  MergeValSource{Meta: MergeValMetaSource{Note: sp("new")}},
			want: MergeValDest{Meta: MergeValMetaDest{Note: sp("new")}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dst := tt.dst
			converter.Convert(&tt.src, &dst, converter.WithValueMode(converter.SparseMerge))
			require.Equal(t, tt.want, dst)
		})
	}
}
