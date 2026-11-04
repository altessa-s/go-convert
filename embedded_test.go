// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package convert_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	converter "github.com/altessa-s/go-convert"
)

type EmbeddedSrc struct {
	SmallStruct
	Extra string
}

type EmbeddedDst struct {
	SmallStruct
	Extra string
}

func TestEmbeddedStructFieldsAreCopied(t *testing.T) {
	src := EmbeddedSrc{
		A: 10, B: 20,
		Extra: "ext",
	}

	t.Run("Default (No Handle Embedded)", func(t *testing.T) {
		// Without option, embedded fields are treated as normal fields struct-to-struct
		// Since names match (SmallStruct), it should work if types match.
		var dst EmbeddedDst
		converter.Convert(src, &dst)

		require.Equal(t, 10, dst.A)
		require.Equal(t, 20, dst.B)
	})

	t.Run("With Handle Embedded", func(t *testing.T) {
		var dst EmbeddedDst
		converter.Convert(src, &dst, converter.WithHandleEmbeddedStructs(true))

		require.Equal(t, 10, dst.A)
		require.Equal(t, 20, dst.B)
	})
}

func TestHandleEmbeddedStructsCopiesSiblingFields(t *testing.T) {
	type Base struct {
		ID int
	}

	type SrcObj struct {
		Base
		Val string
	}

	type DstObj struct {
		Base
		Val string
	}

	src := &SrcObj{ID: 1, Val: "test"}
	dst := &DstObj{}

	converter.Convert(src, dst, converter.WithHandleEmbeddedStructs(true))

	require.Equal(t, "test", dst.Val)
}

type IDBase struct{ ID string }

func TestEmbeddedNilPointerDestinationAllocation(t *testing.T) {
	t.Parallel()

	type flat struct{ ID, Name string }

	type nested struct {
		*IDBase
		Name string
	}

	tests := []struct {
		name     string
		src      flat
		initEmb  bool
		wantBase *IDBase
	}{
		{name: "lazy_non_zero", src: flat{ID: "x", Name: "n"}, wantBase: &IDBase{ID: "x"}},
		{name: "lazy_zero", src: flat{Name: "n"}, wantBase: nil},
		{name: "always_init_zero", src: flat{Name: "n"}, initEmb: true, wantBase: &IDBase{}},
		{name: "always_init_non_zero", src: flat{ID: "x", Name: "n"}, initEmb: true, wantBase: &IDBase{ID: "x"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var dst nested
			converter.Convert(tc.src, &dst, converter.WithHandleEmbeddedStructs(tc.initEmb))
			require.Equal(t, tc.wantBase, dst.IDBase)
			require.Equal(t, "n", dst.Name)
		})
	}
}

func TestEmbeddedValueIntoNilPointerField(t *testing.T) {
	t.Parallel()

	type from struct {
		IDBase
		Name string
	}

	type to struct {
		*IDBase
		Name string
	}

	var dst to
	converter.Convert(from{ID: "x", Name: "n"}, &dst)
	require.Equal(t, &IDBase{ID: "x"}, dst.IDBase)
	require.Equal(t, "n", dst.Name)
}

func TestIgnorePathsEmbeddedSourceKeepsDestination(t *testing.T) {
	t.Parallel()

	type item struct {
		IDBase
		Name string
	}

	src := item{ID: "new", Name: "new"}
	dst := item{ID: "old", Name: "old"}
	converter.Convert(src, &dst, converter.WithIgnorePaths("idbase"))
	require.Equal(t, item{ID: "old", Name: "new"}, dst)
}

type embeddedBase struct {
	Password string
}

type embeddedSource struct {
	Password string
}

type embeddedDest struct {
	embeddedBase
	Password string
}

func TestIgnorePathsEmbeddedDestination(t *testing.T) {
	t.Parallel()

	src := embeddedSource{Password: "secret"}

	var dst embeddedDest
	converter.Convert(src, &dst, converter.WithHandleEmbeddedStructs(true), converter.WithIgnorePaths("password"))
	require.Empty(t, dst.Password)
	require.Empty(t, dst.embeddedBase.Password)
}

type embeddedSourceSide struct {
	embeddedBase
	Name string
}

type embeddedDestSide struct {
	Password string
	Name     string
}

func TestIgnorePathsEmbeddedSourceSubtree(t *testing.T) {
	t.Parallel()

	src := embeddedSourceSide{Password: "secret", Name: "n"}

	var dst embeddedDestSide
	converter.Convert(src, &dst, converter.WithHandleEmbeddedStructs(true), converter.WithIgnorePaths("embeddedbase"))
	require.Empty(t, dst.Password)
	require.Equal(t, "n", dst.Name)
}

type promotedBase struct {
	Password string
}

type promotedSource struct {
	promotedBase
	Name string
}

type promotedDest struct {
	Password string
	Name     string
}

func TestIgnorePathsPromotedField(t *testing.T) {
	t.Parallel()

	src := promotedSource{Password: "secret", Name: "n"}

	var dst promotedDest
	converter.Convert(src, &dst, converter.WithHandleEmbeddedStructs(true), converter.WithIgnorePaths("password"))
	require.Empty(t, dst.Password)
	require.Equal(t, "n", dst.Name)
}

type embedPtrBase struct {
	Password string
}

type embedPtrSource struct {
	*embedPtrBase
	Name string
}

type embedPtrDest struct {
	*embedPtrBase
	Name string
}

func TestIgnorePathsEmbeddedPointerStaysNil(t *testing.T) {
	t.Parallel()

	src := embedPtrSource{embedPtrBase: &embedPtrBase{Password: "secret"}, Name: "n"}

	var dst embedPtrDest
	converter.Convert(src, &dst, converter.WithHandleEmbeddedStructs(true), converter.WithIgnorePaths("embedptrbase"))
	require.Nil(t, dst.embedPtrBase)
	require.Equal(t, "n", dst.Name)
}
