// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package fieldmaskpb_test

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/typepb"

	"github.com/altessa-s/go-convert/proto/codec/fieldmaskpb"

	converter "github.com/altessa-s/go-convert"
	fmpb "google.golang.org/protobuf/types/known/fieldmaskpb"
)

func failNextE(t *testing.T) func(string, reflect.Value, reflect.Value) error {
	t.Helper()

	return func(string, reflect.Value, reflect.Value) error {
		require.Fail(t, "next should not be called")
		return nil
	}
}

func TestNewE_Syntax(t *testing.T) {
	t.Parallel()

	codec := fieldmaskpb.NewE()

	for _, paths := range [][]string{{"name"}, {"address.city", "display_name2"}, {"*"}, {}} {
		var dst *fmpb.FieldMask
		require.NoError(t, codec("mask", reflect.ValueOf(paths), reflect.ValueOf(&dst).Elem(), failNextE(t)), "%q", paths)
		require.Equal(t, paths, dst.GetPaths())

		var back []string
		require.NoError(t, codec("mask", reflect.ValueOf(dst), reflect.ValueOf(&back).Elem(), failNextE(t)))
	}

	for _, tc := range []struct {
		paths  []string
		reason string
	}{
		{paths: []string{""}, reason: "empty path"},
		{paths: []string{"a..b"}, reason: "empty field name"},
		{paths: []string{"name."}, reason: "empty field name"},
		{paths: []string{"displayName"}, reason: `field name "displayName" is not lower snake_case`},
		{paths: []string{"1st"}, reason: `field name "1st" must start with a lower-case letter`},
		{paths: []string{"name", "*"}, reason: "the wildcard must be the only path"},
	} {
		dst := mask("kept")
		err := codec("mask", reflect.ValueOf(tc.paths), reflect.ValueOf(&dst).Elem(), failNextE(t))

		var pathErr fieldmaskpb.PathError
		require.ErrorAs(t, err, &pathErr, "%q", tc.paths)
		require.Equal(t, tc.reason, pathErr.Reason)
		require.Empty(t, pathErr.Message)
		require.Equal(t, []string{"kept"}, dst.GetPaths(), "the destination is left unchanged")

		// The FieldMask -> []string direction validates too.
		out := []string{"kept"}
		err = codec("mask", reflect.ValueOf(mask(tc.paths...)), reflect.ValueOf(&out).Elem(), failNextE(t))
		require.ErrorAs(t, err, &pathErr)
		require.Equal(t, []string{"kept"}, out)
	}
}

func TestNewE_WithMessage(t *testing.T) {
	t.Parallel()

	codec := fieldmaskpb.NewE(fieldmaskpb.WithMessage((*typepb.Field)(nil)), fieldmaskpb.WithNormalize())

	var dst *fmpb.FieldMask
	require.NoError(t, codec("mask", reflect.ValueOf([]string{"name", "json_name", "name"}), reflect.ValueOf(&dst).Elem(), failNextE(t)))
	require.True(t, proto.Equal(mask("json_name", "name"), dst))

	err := codec("mask", reflect.ValueOf([]string{"name", "missing"}), reflect.ValueOf(&dst).Elem(), failNextE(t))
	require.Equal(t, fieldmaskpb.PathError{
		Message: "google.protobuf.Field", Path: "missing", Reason: `google.protobuf.Field has no field "missing"`,
	}, err)

	// Through the converter the error is returned unchanged.
	type request struct{ UpdateMask []string }

	type requestPB struct{ UpdateMask *fmpb.FieldMask }

	_, err = converter.ConvertE(request{UpdateMask: []string{"options.name.x"}}, &requestPB{}, converter.WithErrorCodecs(codec))

	var pathErr fieldmaskpb.PathError
	require.ErrorAs(t, err, &pathErr)
	require.Equal(t, "options.name.x", pathErr.Path)
}

func TestNewE_Delegates(t *testing.T) {
	t.Parallel()

	called := false
	err := fieldmaskpb.NewE()("f", reflect.ValueOf("a"), reflect.ValueOf(new(string)).Elem(), func(string, reflect.Value, reflect.Value) error {
		called = true
		return nil
	})
	require.NoError(t, err)
	require.True(t, called)
}
