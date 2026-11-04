// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package enumpb_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-convert/proto/codec/enumpb"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/typepb"

	converter "github.com/altessa-s/go-convert"
)

func TestTable_ToGo(t *testing.T) {
	t.Parallel()

	table := enumpb.MustNewTable(cardinalities)

	got, err := table.ToGo("field.cardinality", typepb.Field_CARDINALITY_REQUIRED)
	require.NoError(t, err)
	require.Equal(t, cardinalityRequired, got)

	_, err = table.ToGo("field.cardinality", typepb.Field_CARDINALITY_UNKNOWN)
	unknown, ok := errors.AsType[enumpb.UnknownValueError](err)
	require.True(t, ok, "error %v is not an UnknownValueError", err)
	require.Equal(t, "field.cardinality", unknown.Path)
	require.Equal(t, protoreflect.EnumNumber(0), unknown.Value)
}

func TestTable_ToEnum(t *testing.T) {
	t.Parallel()

	table := enumpb.MustNewTable(cardinalities)

	got, err := table.ToEnum("", cardinalityRepeated)
	require.NoError(t, err)
	require.Equal(t, typepb.Field_CARDINALITY_REPEATED, got)

	got, err = table.ToEnum("cardinality", cardinalityMissing)
	unknown, ok := errors.AsType[enumpb.UnknownValueError](err)
	require.True(t, ok, "error %v is not an UnknownValueError", err)
	require.Equal(t, "cardinality", unknown.Path)
	require.Equal(t, typepb.Field_CARDINALITY_UNKNOWN, got)
}

func TestTable_UnspecifiedAsEmpty(t *testing.T) {
	t.Parallel()

	table := enumpb.MustNewTable(cardinalities, enumpb.WithUnspecifiedAsEmpty())

	value, err := table.ToGo("cardinality", typepb.Field_CARDINALITY_UNKNOWN)
	require.NoError(t, err)
	require.Equal(t, cardinalityNone, value)

	enum, err := table.ToEnum("cardinality", cardinalityNone)
	require.NoError(t, err)
	require.Equal(t, typepb.Field_CARDINALITY_UNKNOWN, enum)

	_, err = table.ToEnum("cardinality", cardinalityMissing)
	require.Error(t, err, "a non-zero value missing from the table still fails")
}

func TestTable_Codec(t *testing.T) {
	t.Parallel()

	type src struct{ Cardinality typepb.Field_Cardinality }
	type dst struct{ Cardinality cardinality }

	codec := enumpb.MustNewTable(cardinalities).Codec()

	out, err := converter.ConvertE(src{Cardinality: typepb.Field_CARDINALITY_OPTIONAL}, &dst{}, converter.WithErrorCodecs(codec))
	require.NoError(t, err)
	require.Equal(t, cardinalityOptional, out.Cardinality)
}

func TestNewTable_RejectsDuplicateValue(t *testing.T) {
	t.Parallel()

	table := map[typepb.Field_Cardinality]cardinality{
		typepb.Field_CARDINALITY_OPTIONAL: cardinalityOptional,
		typepb.Field_CARDINALITY_REQUIRED: cardinalityOptional,
	}

	got, err := enumpb.NewTable(table)
	require.Nil(t, got)

	var cfg enumpb.ConfigError
	require.ErrorAs(t, err, &cfg)
	require.Equal(t, "NewTable", cfg.Func)
	require.ErrorContains(t, err, "both map to")
	require.Panics(t, func() { enumpb.MustNewTable(table) })
}
