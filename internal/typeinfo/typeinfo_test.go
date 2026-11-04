// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package typeinfo

import (
	"reflect"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func syntheticType(i int) reflect.Type {
	return reflect.StructOf([]reflect.StructField{{
		Name: "Value",
		Type: reflect.TypeFor[int](),
		Tag:  reflect.StructTag(`index:"` + strconv.Itoa(i) + `"`),
	}})
}

func TestCacheDescribesStructFields(t *testing.T) {
	t.Parallel()

	type Base struct{ ID int }

	type item struct {
		Base
		Name   string
		hidden int //nolint:unused // checks that unexported fields are skipped
	}

	info := NewCache().Get(reflect.TypeFor[*item]())
	require.Len(t, info.Fields, 3)
	require.Equal(t, map[string]int{"base": 0, "name": 1}, info.ByName)
	require.Equal(t, map[string]int{"base": 0}, info.Embedded)
	require.Equal(t, Field{Lower: "hidden"}, info.Fields[2])

	require.Empty(t, NewCache().Get(reflect.TypeFor[int]()).Fields)
}

func TestCacheIsBoundedUnderConcurrency(t *testing.T) {
	t.Parallel()

	cache := NewCache()

	var wg sync.WaitGroup
	for i := range MaxTypes + 64 {
		wg.Go(func() { cache.Get(syntheticType(i)) })
	}

	wg.Wait()
	require.LessOrEqual(t, cache.Len(), MaxTypes)
}

func TestBoundedZeroValueUsesRelationLimit(t *testing.T) {
	t.Parallel()

	var memo Bounded[bool]
	for i := range MaxRelations + 64 {
		memo.Get(syntheticType(i), func() bool { return true })
	}

	require.LessOrEqual(t, memo.Len(), MaxRelations)
}

func TestInfoMemoizesTypeRelations(t *testing.T) {
	t.Parallel()

	type celsius float64

	src := reflect.TypeFor[float64]()
	info := NewCache().Get(src)

	require.True(t, info.AssignableTo(src, src))
	require.False(t, info.AssignableTo(src, reflect.TypeFor[celsius]()))
	require.True(t, info.ConvertibleTo(src, reflect.TypeFor[celsius]()))
	require.False(t, info.ConvertibleTo(src, reflect.TypeFor[string]()))
}

func TestKeyedSupportsComparableKeysUnderConcurrency(t *testing.T) {
	t.Parallel()

	type pair struct{ src, dst reflect.Type }

	memo := NewKeyed[pair, int](32)
	dst := reflect.TypeFor[string]()

	var wg sync.WaitGroup
	for i := range 128 {
		wg.Go(func() {
			key := pair{syntheticType(i % 64), dst}
			if got := memo.Get(key, func() int { return i % 64 }); got != i%64 {
				panic("memo returned a value of another key")
			}
		})
	}

	wg.Wait()
	require.LessOrEqual(t, memo.Len(), 32)
	require.Equal(t, 7, memo.Get(pair{dst, dst}, func() int { return 7 }))
	require.Equal(t, 7, memo.Get(pair{dst, dst}, func() int { return 8 }))
}
