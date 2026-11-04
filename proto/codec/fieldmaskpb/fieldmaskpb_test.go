// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package fieldmaskpb_test

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-convert/proto/codec/fieldmaskpb"

	"google.golang.org/protobuf/proto"

	converter "github.com/altessa-s/go-convert"
	fmpb "google.golang.org/protobuf/types/known/fieldmaskpb"
)

// path is a named string type, to check that any string kind is supported.
type path string

func failNext(t *testing.T) func(string, reflect.Value, reflect.Value) {
	t.Helper()
	return func(string, reflect.Value, reflect.Value) {
		require.Fail(t, "next should not be called")
	}
}

func TestNew_MaskToPaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		opts []fieldmaskpb.Option
		src  any
		dst  any
		want any
	}{
		{name: "paths", src: mask("name", "address.city"), dst: new([]string), want: []string{"name", "address.city"}},
		{name: "mask value", src: fmpb.FieldMask{Paths: []string{"name"}}, dst: new([]string), want: []string{"name"}},
		{name: "named string elements", src: mask("name"), dst: new([]path), want: []path{"name"}},
		{name: "slice pointer", src: mask("name"), dst: new(*[]string), want: &[]string{"name"}},
		{name: "mask without paths", src: mask(), dst: &[]string{"stale"}, want: []string(nil)},
		{name: "nil mask leaves the destination", src: (*fmpb.FieldMask)(nil), dst: &[]string{"kept"}, want: []string{"kept"}},
		{name: "ignore empty", opts: []fieldmaskpb.Option{fieldmaskpb.WithIgnoreEmpty()}, src: mask(), dst: &[]string{"kept"}, want: []string{"kept"}},
		{name: "normalize", opts: []fieldmaskpb.Option{fieldmaskpb.WithNormalize()},
			src: mask("name", "address.city", "address", "name"), dst: new([]string), want: []string{"address", "name"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dst := reflect.ValueOf(tt.dst).Elem()
			fieldmaskpb.New(tt.opts...)("UpdateMask", reflect.ValueOf(tt.src), dst, failNext(t))
			require.Equal(t, tt.want, dst.Interface())
		})
	}
}

func TestNew_PathsToMask(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		opts []fieldmaskpb.Option
		src  any
		dst  *fmpb.FieldMask
		want *fmpb.FieldMask
	}{
		{name: "paths", src: []string{"name", "address.city"}, want: mask("name", "address.city")},
		{name: "named string elements", src: []path{"name"}, want: mask("name")},
		{name: "slice pointer", src: &[]string{"name"}, want: mask("name")},
		{name: "empty slice", src: []string{}, want: mask()},
		{name: "existing mask is overwritten", src: []string{"name"}, dst: mask("stale"), want: mask("name")},
		{name: "nil slice leaves the destination", src: []string(nil), dst: mask("kept"), want: mask("kept")},
		{name: "nil slice pointer leaves the destination", src: (*[]string)(nil), want: nil},
		{name: "ignore empty", opts: []fieldmaskpb.Option{fieldmaskpb.WithIgnoreEmpty()}, src: []string{}, want: nil},
		{name: "normalize", opts: []fieldmaskpb.Option{fieldmaskpb.WithNormalize()},
			src: []string{"name", "address.city", "address"}, want: mask("address", "name")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dst := tt.dst
			fieldmaskpb.New(tt.opts...)("UpdateMask", reflect.ValueOf(tt.src), reflect.ValueOf(&dst).Elem(), failNext(t))
			if tt.want == nil {
				require.Nil(t, dst)
				return
			}
			require.True(t, proto.Equal(tt.want, dst), "got %v", dst)
		})
	}
}

func TestNew_CopiesPaths(t *testing.T) {
	t.Parallel()

	codec := fieldmaskpb.New()

	src := mask("name")
	var paths []string
	codec("f", reflect.ValueOf(src), reflect.ValueOf(&paths).Elem(), failNext(t))
	paths[0] = "changed"
	require.Equal(t, []string{"name"}, src.GetPaths())

	var dst *fmpb.FieldMask
	codec("f", reflect.ValueOf(paths), reflect.ValueOf(&dst).Elem(), failNext(t))
	paths[0] = "again"
	require.Equal(t, []string{"changed"}, dst.GetPaths())
}

func TestNew_Delegates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		src  any
		dst  any
	}{
		{name: "mask to string", src: mask("a"), dst: new(string)},
		{name: "int slice to mask", src: []int{1}, dst: new(*fmpb.FieldMask)},
		{name: "strings to strings", src: []string{"a"}, dst: new([]string)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			called := false
			fieldmaskpb.New()("f", reflect.ValueOf(tt.src), reflect.ValueOf(tt.dst).Elem(), func(string, reflect.Value, reflect.Value) {
				called = true
			})
			require.True(t, called)
		})
	}
}

type updateRequest struct {
	Name       string
	UpdateMask []string
}

type updateRequestPB struct {
	Name       string
	UpdateMask *fmpb.FieldMask
}

func TestConverter_BothDirections(t *testing.T) {
	t.Parallel()

	opts := []converter.Option{converter.WithCodecs(fieldmaskpb.New(fieldmaskpb.WithNormalize()))}

	pb, err := converter.ConvertE(&updateRequest{Name: "n", UpdateMask: []string{"title", "name"}}, &updateRequestPB{}, opts...)
	require.NoError(t, err)
	require.Equal(t, "n", pb.Name)
	require.True(t, proto.Equal(mask("name", "title"), pb.UpdateMask))

	back, err := converter.ConvertE(pb, &updateRequest{}, opts...)
	require.NoError(t, err)
	require.Equal(t, &updateRequest{Name: "n", UpdateMask: []string{"name", "title"}}, back)
}
