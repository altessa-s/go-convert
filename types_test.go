// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package convert_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	converter "github.com/altessa-s/go-convert"
)

// Conversions where reflect.ConvertibleTo is true but a plain
// reflect.Value.Convert changes the value's meaning or panics.

type intSide struct {
	Scalar   int64
	Unsigned uint16
	Ptr      *int32
	Items    []int
	Values   map[string]int
	Keys     map[int]string
	Pair     []int64
	PairMap  map[string][]int64
	Bytes    string
	Runes    string
}

type textSide struct {
	Scalar   string
	Unsigned string
	Ptr      *string
	Items    []string
	Values   map[string]string
	Keys     map[string]string
	Pair     [2]int64
	PairMap  map[string][2]int64
	Bytes    []byte
	Runes    []rune
}

func TestIntegerToStringIsDecimal(t *testing.T) {
	t.Parallel()

	src := intSide{
		Scalar:   65,
		Unsigned: 1000,
		Ptr:      new(int32(-7)),
		Items:    []int{65, 66},
		Values:   map[string]int{"a": 65},
		Keys:     map[int]string{65: "x"},
	}
	got := converter.Convert(src, &textSide{})

	require.Equal(t, "65", got.Scalar)
	require.Equal(t, "1000", got.Unsigned)
	require.Equal(t, "-7", *got.Ptr)
	require.Equal(t, []string{"65", "66"}, got.Items)
	require.Equal(t, map[string]string{"a": "65"}, got.Values)
	require.Equal(t, map[string]string{"65": "x"}, got.Keys)
}

func TestShortSliceToArrayDoesNotPanic(t *testing.T) {
	t.Parallel()

	src := intSide{Pair: []int64{1}, PairMap: map[string][]int64{"k": {1}}}

	var got *textSide

	require.NotPanics(t, func() { got = converter.Convert(src, &textSide{}) })
	require.Equal(t, [2]int64{}, got.Pair)
	require.Equal(t, [2]int64{}, got.PairMap["k"])

	full := converter.Convert(intSide{Pair: []int64{1, 2}, PairMap: map[string][]int64{"k": {3, 4}}}, &textSide{})
	require.Equal(t, [2]int64{1, 2}, full.Pair)
	require.Equal(t, [2]int64{3, 4}, full.PairMap["k"])
}

func TestStringBytesAndRunesKeepGoSemantics(t *testing.T) {
	t.Parallel()

	got := converter.Convert(intSide{Bytes: "héllo", Runes: "héllo"}, &textSide{})
	require.Equal(t, []byte("héllo"), got.Bytes)
	require.Equal(t, []rune("héllo"), got.Runes)

	back := converter.Convert(textSide{Bytes: []byte("héllo"), Runes: []rune("héllo")}, &intSide{})
	require.Equal(t, "héllo", back.Bytes)
	require.Equal(t, "héllo", back.Runes)
}

func TestNilContainersRemainNilAcrossElementConversions(t *testing.T) {
	t.Parallel()

	type source struct {
		Items  []int
		Values map[string]int
	}

	type destination struct {
		Items  []int64
		Values map[string]int64
	}

	got := converter.Convert(source{}, &destination{
		Items:  []int64{1, 2},
		Values: map[string]int64{"old": 1},
	})
	require.Nil(t, got.Items)
	require.Nil(t, got.Values)
}
