// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package convert_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	converter "github.com/altessa-s/go-convert"
)

func TestIgnoreFieldsLeavesFieldUnset(t *testing.T) {
	src := &SrcUser{Name: "Carol", Email: "c@d.com", Age: 40}
	dst := &DstUser{}

	converter.Convert(src, dst, converter.WithIgnoreFields("Email"))

	require.Equal(t, "Carol", dst.Name)
	require.Equal(t, "", dst.Email, "should be ignored")
}

type primitiveSliceSource struct {
	Data []byte
}

type primitiveSliceDest struct {
	Data []byte
}

func TestIgnoreFieldsDoesNotForcePrimitiveSliceCopy(t *testing.T) {
	t.Parallel()

	var dst primitiveSliceDest
	converter.Convert(primitiveSliceSource{}, &dst, converter.WithIgnoreFields("missing"))
	require.Nil(t, dst.Data)
}

type pathLeaf struct {
	Password string
	Headers  string
}

type pathSource struct {
	SMTP  pathLeaf
	Other pathLeaf
	Items []pathLeaf
	ByID  map[string]pathLeaf
}

type pathDest struct {
	SMTP  pathLeaf
	Other pathLeaf
	Items []pathLeaf
	ByID  map[string]pathLeaf
}

func pathFixture() pathSource {
	return pathSource{
		SMTP:  pathLeaf{Password: "smtp-password", Headers: "smtp-headers"},
		Other: pathLeaf{Password: "other-password", Headers: "other-headers"},
		Items: []pathLeaf{{Password: "item-password", Headers: "item-headers"}},
		ByID:  map[string]pathLeaf{"k": {Password: "map-password", Headers: "map-headers"}},
	}
}

func TestIgnorePaths(t *testing.T) {
	t.Parallel()

	t.Run("exact nested path", func(t *testing.T) {
		t.Parallel()

		var dst pathDest
		converter.Convert(pathFixture(), &dst, converter.WithIgnorePaths("smtp.password"))

		require.Empty(t, dst.SMTP.Password)
		require.Equal(t, "smtp-headers", dst.SMTP.Headers)
		require.Equal(t, "other-password", dst.Other.Password)
		require.Equal(t, "item-password", dst.Items[0].Password)
		require.Equal(t, "map-password", dst.ByID["k"].Password)
	})

	t.Run("parent path ignores subtree", func(t *testing.T) {
		t.Parallel()

		var dst pathDest
		converter.Convert(pathFixture(), &dst, converter.WithIgnorePaths("smtp"))

		require.Empty(t, dst.SMTP.Password)
		require.Empty(t, dst.SMTP.Headers)
		require.Equal(t, "other-password", dst.Other.Password)
	})

	t.Run("slice element path", func(t *testing.T) {
		t.Parallel()

		var dst pathDest
		converter.Convert(pathFixture(), &dst, converter.WithIgnorePaths("items.password"))

		require.Empty(t, dst.Items[0].Password)
		require.Equal(t, "item-headers", dst.Items[0].Headers)
		require.Equal(t, "smtp-password", dst.SMTP.Password)
	})

	t.Run("map value path", func(t *testing.T) {
		t.Parallel()

		var dst pathDest
		converter.Convert(pathFixture(), &dst, converter.WithIgnorePaths("byid.password"))

		require.Empty(t, dst.ByID["k"].Password)
		require.Equal(t, "map-headers", dst.ByID["k"].Headers)
		require.Equal(t, "item-password", dst.Items[0].Password)
	})

	t.Run("nil containers stay nil", func(t *testing.T) {
		t.Parallel()

		var dst pathDest
		converter.Convert(pathSource{}, &dst, converter.WithIgnoreFields("password"))
		require.Nil(t, dst.Items)
		require.Nil(t, dst.ByID)
	})

	t.Run("leaf option keeps assignable nested values whole", func(t *testing.T) {
		t.Parallel()

		var dst pathDest
		converter.Convert(pathFixture(), &dst, converter.WithIgnoreFields("password"))

		// Same-type nested values take the assignable fast path, so the
		// leaf option only reaches fields that are actually walked.
		require.Equal(t, "smtp-password", dst.SMTP.Password)
		require.Equal(t, "other-password", dst.Other.Password)
		require.Equal(t, "item-password", dst.Items[0].Password)
		require.Equal(t, "map-password", dst.ByID["k"].Password)
		require.Equal(t, "smtp-headers", dst.SMTP.Headers)
	})
}

func TestIgnorePathsNormalizesCaseAndDots(t *testing.T) {
	t.Parallel()

	var dst pathDest
	converter.Convert(pathFixture(), &dst, converter.WithIgnorePaths(".SMTP.Password."))
	require.Empty(t, dst.SMTP.Password)
	require.Equal(t, "smtp-headers", dst.SMTP.Headers)
}

func TestIgnorePathsRootContainers(t *testing.T) {
	t.Parallel()

	t.Run("root slice", func(t *testing.T) {
		t.Parallel()

		var dst []pathLeaf
		converter.Convert([]pathLeaf{{Password: "x", Headers: "h"}}, &dst, converter.WithIgnorePaths("password"))
		require.Empty(t, dst[0].Password)
		require.Equal(t, "h", dst[0].Headers)
	})

	t.Run("ConvertSeq", func(t *testing.T) {
		t.Parallel()

		conv := converter.New[pathLeaf, pathLeaf](converter.WithIgnorePaths("password"))

		var got []pathLeaf
		for out := range conv.ConvertSeq([]pathLeaf{{Password: "x", Headers: "h"}}) {
			got = append(got, out)
		}

		require.Len(t, got, 1)
		require.Empty(t, got[0].Password)
		require.Equal(t, "h", got[0].Headers)
	})

	t.Run("ConvertMapSeq", func(t *testing.T) {
		t.Parallel()

		var got []pathLeaf
		for _, out := range converter.ConvertMapSeq[string, pathLeaf](
			map[string]pathLeaf{"k": {Password: "x", Headers: "h"}},
			converter.WithIgnorePaths("password"),
		) {
			got = append(got, out)
		}

		require.Len(t, got, 1)
		require.Empty(t, got[0].Password)
		require.Equal(t, "h", got[0].Headers)
	})
}

func TestIgnorePathsConvertMapSeqKeepsKeys(t *testing.T) {
	t.Parallel()

	src := map[pathLeaf]string{{Password: "k"}: "v"}

	var got []pathLeaf
	for key := range converter.ConvertMapSeq[pathLeaf, string](
		src, converter.WithIgnorePaths("password"),
	) {
		got = append(got, key)
	}

	require.Len(t, got, 1)
	require.Equal(t, "k", got[0].Password)
}

type arraySource struct {
	Pair [2]pathLeaf
}

type arrayDest struct {
	Pair [2]pathLeaf
}

func TestIgnorePathsArray(t *testing.T) {
	t.Parallel()

	src := arraySource{Pair: [2]pathLeaf{{Password: "a", Headers: "ha"}, {Password: "b", Headers: "hb"}}}

	var dst arrayDest
	converter.Convert(src, &dst, converter.WithIgnorePaths("pair.password"))
	require.Empty(t, dst.Pair[0].Password)
	require.Empty(t, dst.Pair[1].Password)
	require.Equal(t, "ha", dst.Pair[0].Headers)
}

type nestedMapSource struct {
	Groups map[string][]pathLeaf
}

type nestedMapDest struct {
	Groups map[string][]pathLeaf
}

func TestIgnorePathsNestedMapOfSlices(t *testing.T) {
	t.Parallel()

	src := nestedMapSource{Groups: map[string][]pathLeaf{"g": {{Password: "x", Headers: "h"}}}}

	var dst nestedMapDest
	converter.Convert(src, &dst, converter.WithIgnorePaths("groups.password"))
	require.Empty(t, dst.Groups["g"][0].Password)
	require.Equal(t, "h", dst.Groups["g"][0].Headers)
}

type pointerSliceSource struct {
	Items []*pathLeaf
	ByID  map[string]*pathLeaf
}

type pointerSliceDest struct {
	Items []*pathLeaf
	ByID  map[string]*pathLeaf
}

func TestIgnorePathsNilPointerElements(t *testing.T) {
	t.Parallel()

	src := pointerSliceSource{Items: []*pathLeaf{nil}, ByID: map[string]*pathLeaf{"k": nil}}

	var dst pointerSliceDest

	require.NotPanics(t, func() {
		converter.Convert(src, &dst, converter.WithIgnorePaths("password"))
	})
	require.Nil(t, dst.Items[0])
	require.Nil(t, dst.ByID["k"])
}

type pointerPathSource struct {
	SMTP *pathLeaf
}

type pointerPathDest struct {
	SMTP *pathLeaf
}

func TestIgnorePathsPointerField(t *testing.T) {
	t.Parallel()

	src := pointerPathSource{SMTP: &pathLeaf{Password: "secret", Headers: "h"}}

	var dst pointerPathDest
	converter.Convert(src, &dst, converter.WithIgnorePaths("smtp.password"))
	require.Empty(t, dst.SMTP.Password)
	require.Equal(t, "h", dst.SMTP.Headers)
}

type ptrContainerSource struct {
	Items *[]pathLeaf
	Pair  *[2]pathLeaf
	ByID  *map[string]pathLeaf
}

type ptrContainerDest struct {
	Items *[]pathLeaf
	Pair  *[2]pathLeaf
	ByID  *map[string]pathLeaf
}

func TestIgnorePathsPointerContainers(t *testing.T) {
	t.Parallel()

	items := []pathLeaf{{Password: "i", Headers: "ih"}}
	pair := [2]pathLeaf{{Password: "a", Headers: "ah"}, {Password: "b", Headers: "bh"}}
	byID := map[string]pathLeaf{"k": {Password: "m", Headers: "mh"}}
	src := ptrContainerSource{Items: &items, Pair: &pair, ByID: &byID}

	var dst ptrContainerDest
	converter.Convert(src, &dst,
		converter.WithIgnorePaths("items.password", "pair.password", "byid.password"))
	require.Empty(t, (*dst.Items)[0].Password)
	require.Empty(t, (*dst.Pair)[0].Password)
	require.Empty(t, (*dst.Pair)[1].Password)
	require.Empty(t, (*dst.ByID)["k"].Password)
	require.Equal(t, "ih", (*dst.Items)[0].Headers)
}

type keySource struct {
	ID string
}

type keyDest struct {
	ID string
}

type keyMapSource struct {
	Items map[keySource]string
}

type keyMapDest struct {
	Items map[keyDest]string
}

func TestIgnorePathsDoesNotTouchMapKeys(t *testing.T) {
	t.Parallel()

	src := keyMapSource{Items: map[keySource]string{{ID: "a"}: "1", {ID: "b"}: "2"}}

	var dst keyMapDest
	converter.Convert(src, &dst, converter.WithIgnorePaths("id"))
	require.Len(t, dst.Items, 2)

	for k := range dst.Items {
		require.NotEmpty(t, k.ID)
	}
}

type distinctKeySource struct {
	ID string
}

type distinctKeyDest struct {
	ID    string
	Extra int
}

type distinctKeyMapSource struct {
	Items map[distinctKeySource]string
}

type distinctKeyMapDest struct {
	Items map[distinctKeyDest]string
}

func TestIgnorePathsDoesNotTouchNonConvertibleMapKeys(t *testing.T) {
	t.Parallel()

	src := distinctKeyMapSource{Items: map[distinctKeySource]string{{ID: "a"}: "1", {ID: "b"}: "2"}}

	var dst distinctKeyMapDest
	converter.Convert(src, &dst, converter.WithIgnorePaths("id"))
	require.Len(t, dst.Items, 2)

	for k := range dst.Items {
		require.NotEmpty(t, k.ID)
	}
}

type updateLeaf struct {
	Password string
	TLS      *string
}

type updateSource struct {
	SMTP updateLeaf
}

type updateDest struct {
	SMTP updateLeaf
}

func TestIgnorePathsPreservesPrePopulatedDestination(t *testing.T) {
	t.Parallel()

	tls := "on"
	dst := updateDest{SMTP: updateLeaf{Password: "old", TLS: &tls}}
	converter.Convert(updateSource{SMTP: updateLeaf{Password: "new"}}, &dst,
		converter.WithIgnorePaths("smtp.password"))
	require.Equal(t, "old", dst.SMTP.Password)
	require.NotNil(t, dst.SMTP.TLS)
	require.Equal(t, "on", *dst.SMTP.TLS)
}

type recursiveMap map[string]recursiveMap

type recursiveMapHolder struct {
	Meta recursiveMap
}

func TestIgnorePathsSelfReferentialContainer(t *testing.T) {
	t.Parallel()

	src := recursiveMapHolder{Meta: recursiveMap{"a": recursiveMap{}}}

	var dst recursiveMapHolder

	require.NotPanics(t, func() {
		converter.Convert(src, &dst, converter.WithIgnorePaths("other"))
	})
	require.Len(t, dst.Meta, 1)
}

type privateState struct {
	Secret   string
	Name     string
	Optional *string
	state    int
}

func newPrivateState(secret, name string, state int) privateState {
	return privateState{Secret: secret, Name: name, state: state}
}
func (s privateState) State() int { return s.state }

func TestIgnorePathsRootPreservesPrivateState(t *testing.T) {
	type item struct {
		Visible, Name string
		state         int
	}

	src, dst := item{"source", "new", 7}, item{"keep", "old", 3}
	converter.Convert(src, &dst, converter.WithIgnorePaths("visible"))

	if want := (item{"keep", "new", 3}); dst != want {
		t.Fatalf("root = %+v, want %+v", dst, want)
	}

	var out []item
	converter.Convert([]item{src}, &out, converter.WithIgnorePaths("visible"))

	if want := (item{"", "new", 7}); len(out) != 1 || out[0] != want {
		t.Fatalf("root slice = %+v, want [%+v]", out, want)
	}
}

func TestIgnorePathsPreservesPrivateState(t *testing.T) {
	t.Parallel()

	type root struct {
		Value   privateState
		Pointer *privateState
		Items   []privateState
		Map     map[string]privateState
		Array   [1]privateState
	}

	for _, tc := range []struct {
		name   string
		option converter.Option
		state  int
	}{
		{"overwrite", nil, 7},
		{"sparse", converter.WithSparseMerge(), 3},
		{"zero", converter.WithIgnoreZeroValues(), 3},
		{"nil", converter.WithIgnoreNilValues(), 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			source := newPrivateState("source secret", "source", 7)
			old := newPrivateState("old secret", "old", 3)
			optional := "keep"
			old.Optional = &optional
			src := root{source, &source, []privateState{source}, map[string]privateState{"k": source}, [1]privateState{source}}
			dst := root{Value: old, Pointer: &old}

			opts := []converter.Option{converter.WithIgnorePaths("value.secret", "pointer.secret", "items.secret", "map.secret", "array.secret")}
			if tc.option != nil {
				opts = append(opts, tc.option)
			}

			converter.Convert(src, &dst, opts...)

			for _, got := range []privateState{dst.Value, *dst.Pointer} {
				require.Equal(t, "old secret", got.Secret)
				require.Equal(t, "source", got.Name)
				require.Equal(t, tc.state, got.State())
				require.Equal(t, &optional, got.Optional)
			}

			if tc.option == nil {
				for _, got := range []privateState{dst.Items[0], dst.Map["k"], dst.Array[0]} {
					require.Equal(t, 7, got.State())
					require.Empty(t, got.Secret)
				}
			}

			require.Equal(t, "source secret", source.Secret)

			if tc.option == nil {
				require.Equal(t, "old", old.Name)
			}
		})
	}
}

func TestIgnorePathsAliasedDestination(t *testing.T) {
	t.Parallel()

	type source struct{ Item *privateState }

	type destination struct{ Item *privateState }

	item := newPrivateState("secret", "name", 7)
	src, dst := source{&item}, destination{&item}
	converter.Convert(src, &dst, converter.WithIgnorePaths("item.secret"), converter.WithIgnoreFields("name"))
	require.NotSame(t, src.Item, dst.Item)
	require.Equal(t, "secret", src.Item.Secret)
	require.Equal(t, "name", dst.Item.Name)
	require.Equal(t, 7, dst.Item.State())
}

func TestIgnorePathsLeafAndCopySlices(t *testing.T) {
	t.Parallel()

	type item struct {
		Secret string
		Name   string
		Tags   []string
		state  int
	}

	type root struct{ Item item }

	src := root{item{"source secret", "source name", []string{"source tag"}, 7}}
	dst := root{Item: item{Secret: "keep secret", Name: "keep name"}}
	converter.Convert(src, &dst, converter.WithIgnorePaths("item.secret"), converter.WithIgnoreFields("name"), converter.WithCopySlices())
	require.Equal(t, "keep secret", dst.Item.Secret)
	require.Equal(t, "keep name", dst.Item.Name)
	require.Equal(t, 7, dst.Item.state)
	dst.Item.Tags[0] = "changed"
	require.Equal(t, "source tag", src.Item.Tags[0])
}

func TestIgnorePathsWithFieldMappingsPreservesDestination(t *testing.T) {
	t.Parallel()

	type item struct {
		Original string
		Target   string
		Secret   string
		state    int
	}

	source := item{"new", "unused", "source secret", 7}
	old := item{"keep original", "old target", "keep secret", 3}
	want := item{"keep original", "new", "keep secret", 7}
	opts := []converter.Option{
		converter.WithFieldMappings(map[string]string{"original": "target", "target": "missing"}),
		converter.WithIgnorePaths("item.secret"),
	}

	t.Run("static", func(t *testing.T) {
		src := struct{ Item item }{source}
		dst := struct{ Item item }{old}
		converter.Convert(src, &dst, opts...)
		require.Equal(t, want, dst.Item)
	})
	t.Run("dynamic", func(t *testing.T) {
		src := struct{ Item any }{source}
		dst := struct{ Item any }{old}
		converter.Convert(src, &dst, opts...)
		require.Equal(t, want, dst.Item)
	})
}
