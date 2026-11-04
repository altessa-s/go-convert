// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package convert_test

import (
	"maps"
	"testing"

	"github.com/stretchr/testify/require"

	converter "github.com/altessa-s/go-convert"
)

type Source struct {
	Name     string
	Age      int
	Active   bool
	Ignored  string
	MappedID int
	Tags     []string
	Meta     map[string]string
	Ptr      *int
}

type Dest struct {
	Name     string
	Age      int
	Active   bool
	Ignored  string
	TargetID int
	Tags     []string
	Meta     map[string]string
	Ptr      *int
	Extra    string
}

type NestedSource struct {
	Info Source
}

type NestedDest struct {
	Info Dest
}

func TestConvert(t *testing.T) {
	ptrVal := 42
	src := Source{
		Name:     "Test",
		Age:      30,
		Active:   true,
		Ignored:  "ignore me",
		MappedID: 100,
		Tags:     []string{"a", "b"},
		Meta:     map[string]string{"k": "v"},
		Ptr:      &ptrVal,
	}

	t.Run("Basic Struct Conversion", func(t *testing.T) {
		var dst Dest
		converter.Convert(src, &dst)

		require.Equal(t, src.Name, dst.Name)
		require.Equal(t, src.Age, dst.Age)
		require.Equal(t, src.Active, dst.Active)
		// Without Ignore option, it should be copied
		require.Equal(t, src.Ignored, dst.Ignored)
		// Without a MappedID -> TargetID mapping, TargetID stays 0
		require.Equal(t, 0, dst.TargetID)
		require.Equal(t, src.Tags, dst.Tags)
		require.Equal(t, src.Meta, dst.Meta)
		require.NotNil(t, dst.Ptr)
		require.Equal(t, *src.Ptr, *dst.Ptr)
	})

	t.Run("With Options", func(t *testing.T) {
		var dst Dest
		converter.Convert(src, &dst,
			converter.WithIgnoreFields("ignored"),
			converter.WithFieldMappings(map[string]string{"MappedID": "TargetID"}),
		)

		require.Equal(t, "", dst.Ignored)
		require.Equal(t, src.MappedID, dst.TargetID)
	})

	t.Run("Nested Structs", func(t *testing.T) {
		nSrc := NestedSource{Info: src}

		var nDst NestedDest
		converter.Convert(nSrc, &nDst, converter.WithFieldMappings(map[string]string{"MappedID": "TargetID"}))

		require.Equal(t, src.Name, nDst.Info.Name)
		require.Equal(t, src.MappedID, nDst.Info.TargetID)
	})

	t.Run("Ignore Zero Values", func(t *testing.T) {
		zeroSrc := Source{Name: "", Age: 0}
		dst := Dest{Name: "Keep", Age: 99}

		converter.Convert(zeroSrc, &dst, converter.WithValueMode(converter.IgnoreZeroValues))

		require.Equal(t, "Keep", dst.Name)
		require.Equal(t, 99, dst.Age)
	})

	t.Run("Ignore Nil Values", func(t *testing.T) {
		nilSrc := Source{Ptr: nil}
		existingVal := 10
		dst := Dest{Ptr: &existingVal}

		converter.Convert(nilSrc, &dst, converter.WithValueMode(converter.IgnoreNilValues))

		require.NotNil(t, dst.Ptr)
		require.Equal(t, 10, *dst.Ptr)
	})
}

type SrcUser struct {
	Name  string
	Email string
	Age   int
}

type DstUser struct {
	Name  string
	Email string
	Age   int
}

func TestConvertStructPointers(t *testing.T) {
	src := &SrcUser{Name: "Alice", Email: "a@b.com", Age: 30}
	dst := &DstUser{}

	converter.Convert(src, dst)

	require.Equal(t, "Alice", dst.Name)
	require.Equal(t, "a@b.com", dst.Email)
	require.Equal(t, 30, dst.Age)
}

func TestFieldMappingsRenameField(t *testing.T) {
	type Src struct {
		FullName string
	}

	type Dst struct {
		Name string
	}

	src := &Src{FullName: "Dave"}
	dst := &Dst{}

	converter.Convert(src, dst, converter.WithFieldMappings(map[string]string{"FullName": "Name"}))

	require.Equal(t, "Dave", dst.Name)
}

func TestConvertSlices(t *testing.T) {
	src := []Source{{Name: "A"}, {Name: "B"}}

	t.Run("Slice to Slice", func(t *testing.T) {
		var dst []Dest
		converter.Convert(src, &dst)

		require.Len(t, dst, 2)
		require.Equal(t, "A", dst[0].Name)
		require.Equal(t, "B", dst[1].Name)
	})
}

func TestConvertNestedSliceField(t *testing.T) {
	type SrcItem struct {
		ID   int
		Name string
	}

	type DstItem struct {
		ID   int
		Name string
	}

	type SrcList struct {
		Items []SrcItem
	}

	type DstList struct {
		Items []DstItem
	}

	src := &SrcList{Items: []SrcItem{{ID: 1, Name: "a"}, {ID: 2, Name: "b"}}}
	dst := &DstList{}

	converter.Convert(src, dst)

	require.Len(t, dst.Items, 2)
	require.Equal(t, "a", dst.Items[0].Name)
}

type SmallStruct struct {
	A int
	B int
}

func TestConvertSmallStruct(t *testing.T) {
	// Structs with four or fewer fields take a dedicated fast path.
	src := SmallStruct{A: 1, B: 2}

	var dst SmallStruct

	converter.Convert(src, &dst)

	require.Equal(t, 1, dst.A)
	require.Equal(t, 2, dst.B)
}

func TestNew(t *testing.T) {
	conv := converter.New[Source, *Dest]()
	require.NotNil(t, conv)

	src := Source{Name: "test", Age: 25}

	var dst Dest
	conv.Convert(src, &dst)
	require.Equal(t, "test", dst.Name)
	require.Equal(t, 25, dst.Age)
}

func TestNewWithPointerTypes(t *testing.T) {
	conv := converter.New[*SrcUser, *DstUser]()
	require.NotNil(t, conv)

	src := &SrcUser{Name: "Eve", Age: 20}
	dst := &DstUser{}
	conv.Convert(src, dst)

	require.Equal(t, "Eve", dst.Name)
}

func TestNewAny(t *testing.T) {
	conv := converter.NewAny()
	require.NotNil(t, conv)
}

func TestConvertSeq(t *testing.T) {
	src := []Source{{Name: "A"}, {Name: "B"}}
	iter := converter.ConvertSeq[Dest](src)

	count := 0
	for item := range iter {
		require.Equal(t, src[count].Name, item.Name)
		count++
	}

	require.Equal(t, 2, count)
}

func TestConvertSeqYieldsEveryElement(t *testing.T) {
	src := []SrcUser{
		{Name: "A", Age: 1},
		{Name: "B", Age: 2},
	}

	count := 0
	for dst := range converter.ConvertSeq[DstUser](src) {
		count++

		require.NotEmpty(t, dst.Name)
	}

	require.Equal(t, 2, count)
}

func TestConvertMapSeq(t *testing.T) {
	src := map[string]int{"one": 1, "two": 2}
	// Converting Int values to Int values (direct assignable) with Any->Any

	iter := converter.ConvertMapSeq[string, int](src)

	res := maps.Collect(iter)

	require.Equal(t, map[string]int{"one": 1, "two": 2}, res)
}

type errorSource struct {
	Name string
}

type errorDest struct {
	Name string
}

func TestConvertE(t *testing.T) {
	t.Parallel()

	t.Run("successful conversion", func(t *testing.T) {
		t.Parallel()

		var dst errorDest

		out, err := converter.ConvertE(errorSource{Name: "ok"}, &dst)
		require.NoError(t, err)
		require.Same(t, &dst, out)
		require.Equal(t, "ok", dst.Name)
	})

	t.Run("invalid type combination returns an error", func(t *testing.T) {
		t.Parallel()

		var dst string

		_, err := converter.ConvertE(errorSource{Name: "x"}, &dst)
		require.Error(t, err)
	})

	t.Run("nil slice element stays nil", func(t *testing.T) {
		t.Parallel()

		src := []*errorSource{nil}

		var dst []*errorDest

		_, err := converter.ConvertE(src, &dst)
		require.NoError(t, err)
		require.Nil(t, dst[0])
	})
}
