// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package convert_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	converter "github.com/altessa-s/go-convert"
)

type copySource struct {
	Tags []string
}

type copyDest struct {
	Tags []string
}

func TestCopySlices(t *testing.T) {
	t.Parallel()

	t.Run("copies backing array", func(t *testing.T) {
		t.Parallel()

		src := copySource{Tags: []string{"a", "b"}}

		var dst copyDest
		converter.Convert(src, &dst, converter.WithCopySlices())

		require.Equal(t, src.Tags, dst.Tags)
		dst.Tags[0] = "changed"
		require.Equal(t, "a", src.Tags[0])
	})

	t.Run("nil stays nil", func(t *testing.T) {
		t.Parallel()

		var dst copyDest
		converter.Convert(copySource{}, &dst, converter.WithCopySlices())
		require.Nil(t, dst.Tags)
	})

	t.Run("empty non-nil stays empty non-nil", func(t *testing.T) {
		t.Parallel()

		src := copySource{Tags: []string{}}

		var dst copyDest
		converter.Convert(src, &dst, converter.WithCopySlices())
		require.NotNil(t, dst.Tags)
		require.Empty(t, dst.Tags)
	})
}

type profile struct {
	Tags []string
}

type profileSource struct {
	Profile profile
}

type profileDest struct {
	Profile profile
}

func TestCopySlicesNestedStruct(t *testing.T) {
	t.Parallel()

	src := profileSource{Profile: profile{Tags: []string{"a"}}}

	var dst profileDest
	converter.Convert(src, &dst, converter.WithCopySlices())
	dst.Profile.Tags[0] = "changed"
	require.Equal(t, "a", src.Profile.Tags[0])
}

type slicePointerSource struct {
	Tags *[]string
}

type slicePointerDest struct {
	Tags *[]string
}

func TestCopySlicesPointerToSlice(t *testing.T) {
	t.Parallel()

	tags := []string{"a"}
	src := slicePointerSource{Tags: &tags}

	var dst slicePointerDest
	converter.Convert(src, &dst, converter.WithCopySlices())
	(*dst.Tags)[0] = "changed"

	require.Equal(t, "a", tags[0])
}

type unexportedProfile struct {
	Tags []string
	hash string
}

func newUnexportedProfile() unexportedProfile {
	return unexportedProfile{Tags: []string{"a"}, hash: "h"}
}

func (p unexportedProfile) Hash() string { return p.hash }

type unexportedOuter struct {
	Profile unexportedProfile
}

func TestCopySlicesPreservesUnexportedFields(t *testing.T) {
	t.Parallel()

	src := unexportedOuter{Profile: newUnexportedProfile()}

	var dst unexportedOuter
	converter.Convert(src, &dst, converter.WithCopySlices())
	require.Equal(t, "h", dst.Profile.Hash())
	dst.Profile.Tags[0] = "changed"
	require.Equal(t, "a", src.Profile.Tags[0])
}

type recursiveNode struct {
	Parent *recursiveNode
	Tags   []string
}

type recursiveHolder struct {
	Items []recursiveNode
}

func TestCopySlicesRecursiveType(t *testing.T) {
	t.Parallel()

	src := recursiveHolder{Items: []recursiveNode{{Tags: []string{"a"}}}}

	var dst recursiveHolder

	require.NotPanics(t, func() {
		converter.Convert(src, &dst, converter.WithCopySlices())
	})

	dst.Items[0].Tags[0] = "changed"
	require.Equal(t, "a", src.Items[0].Tags[0])
}
