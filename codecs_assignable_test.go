// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package convert_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	converter "github.com/altessa-s/go-convert"
	convcodec "github.com/altessa-s/go-convert/codec"
)

// upperCodec upper-cases every string value and delegates everything else.
func upperCodec(field string, src, dst reflect.Value, next convcodec.CodecHandler) {
	if src.Kind() == reflect.String && dst.Kind() == reflect.String {
		dst.SetString(strings.ToUpper(src.String()))
		return
	}

	next(field, src, dst)
}

// Codecs run for fields whose type is assignable to the destination, not
// only for fields the built-in conversion cannot assign.
func TestCodecsRunForAssignableFields(t *testing.T) {
	t.Parallel()

	type inner struct{ City string }

	type src struct {
		Name  string
		Inner inner
		Ptr   *inner
		Tags  map[string]string
		Count int
	}

	type dst struct {
		Name  string
		Inner inner
		Ptr   *inner
		Tags  map[string]string
		Count int64
	}

	in := src{Name: "ada", Inner: inner{City: "rome"}, Ptr: &inner{City: "oslo"}, Tags: map[string]string{"k": "v"}, Count: 7}

	tests := []struct {
		name string
		opts []converter.Option
	}{
		{name: "codecs only", opts: []converter.Option{converter.WithCodecs(upperCodec)}},
		{name: "two codecs", opts: []converter.Option{converter.WithCodecs(upperCodec, passthroughCodec)}},
		{name: "with ignore paths", opts: []converter.Option{converter.WithCodecs(upperCodec), converter.WithIgnorePaths("missing")}},
		{name: "with copy slices", opts: []converter.Option{converter.WithCodecs(upperCodec), converter.WithCopySlices()}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var out dst
			converter.Convert(in, &out, tt.opts...)

			require.Equal(t, "ADA", out.Name)
			// An assignable struct reaches the codecs as a whole; the
			// built-in assignment ending the chain does not walk it.
			require.Equal(t, inner{City: "rome"}, out.Inner)
			require.Equal(t, &inner{City: "oslo"}, out.Ptr)
			require.Equal(t, map[string]string{"k": "v"}, out.Tags)
			require.Equal(t, int64(7), out.Count)
		})
	}
}

// Codecs see every element and map value the converter walks, assignable
// ones included.
func TestCodecsRunForAssignableElementsAndMapValues(t *testing.T) {
	t.Parallel()

	type label string

	type src struct {
		Names  []string
		ByID   map[int]string
		Nested struct{ Name string }
	}

	type dst struct {
		Names  []label
		ByID   map[int64]string
		Nested struct{ Name label }
	}

	in := src{Names: []string{"a", "b"}, ByID: map[int]string{1: "x"}}
	in.Nested.Name = "n"

	var out dst
	converter.Convert(in, &out, converter.WithCodecs(upperCodec))

	require.Equal(t, []label{"A", "B"}, out.Names)
	require.Equal(t, map[int64]string{1: "X"}, out.ByID)
	require.Equal(t, label("N"), out.Nested.Name)
}

// The fast assignment path stays in place without codecs.
func TestAssignableFieldsWithoutCodecs(t *testing.T) {
	t.Parallel()

	type src struct {
		Name string
		Tags map[string]string
	}

	in := src{Name: "ada", Tags: map[string]string{"k": "v"}}

	var out src
	converter.Convert(in, &out)

	require.Equal(t, in, out)
}

// A codec that handles a value of an assignable type wins over the built-in
// assignment for nested and repeated values too.
func TestCodecsRunForSliceOfStructsElements(t *testing.T) {
	t.Parallel()

	type item struct{ Name string }

	type srcItems struct{ Items []item }

	type dstItem struct{ Name string }

	type dstItems struct{ Items []dstItem }

	var out dstItems
	converter.Convert(srcItems{Items: []item{{Name: "a"}}}, &out, converter.WithCodecs(upperCodec))

	require.Equal(t, []dstItem{{Name: "A"}}, out.Items)
}

func passthroughCodec(field string, src, dst reflect.Value, next convcodec.CodecHandler) {
	next(field, src, dst)
}
