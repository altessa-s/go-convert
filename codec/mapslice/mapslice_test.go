// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package mapslice_test

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-convert/codec/mapslice"
)

type namedInt int
type namedString string

func TestValues(t *testing.T) {
	src := reflect.ValueOf(map[string]int{"a": 1, "b": 2})
	dst := reflect.New(reflect.SliceOf(reflect.TypeFor[int]())).Elem()

	mapslice.Values("test", src, dst, func(_ string, _, _ reflect.Value) {})

	require.Equal(t, 2, dst.Len())
}

func TestValues_PointerToNilSliceDestination(t *testing.T) {
	src := reflect.ValueOf(map[string]int{"a": 1, "b": 2})

	var dst *[]int

	dstPointer := reflect.ValueOf(&dst).Elem()

	mapslice.Values("test", src, dstPointer, func(_ string, _, _ reflect.Value) {
		require.Fail(t, "next should not be called")
	})

	require.NotNil(t, dst)
	require.Len(t, *dst, 2)
}

func TestValues_PassThrough(t *testing.T) {
	src := reflect.ValueOf("hello")

	var dst string

	dstVal := reflect.ValueOf(&dst).Elem()

	nextCalled := false

	mapslice.Values("test", src, dstVal, func(_ string, _, _ reflect.Value) {
		nextCalled = true
	})

	require.True(t, nextCalled, "next should be called for non-map types")
}

func TestValues_NotNil(t *testing.T) {
	require.NotNil(t, mapslice.Values)
}

func TestKeys(t *testing.T) {
	src := reflect.ValueOf(map[string]int{"a": 1, "b": 2})
	dst := reflect.New(reflect.SliceOf(reflect.TypeFor[string]())).Elem()

	mapslice.Keys("test", src, dst, func(_ string, _, _ reflect.Value) {})

	require.Equal(t, 2, dst.Len())
}

func TestKeys_PointerToNonNilSliceDestination(t *testing.T) {
	src := reflect.ValueOf(map[string]int{"a": 1})
	dst := []string{"existing"}
	dstPointer := reflect.ValueOf(&dst)

	mapslice.Keys("test", src, dstPointer, func(_ string, _, _ reflect.Value) {
		require.Fail(t, "next should not be called")
	})

	require.Len(t, dst, 2)
	require.Contains(t, dst, "existing")
	require.Contains(t, dst, "a")
}

func TestKeys_PassThrough(t *testing.T) {
	src := reflect.ValueOf("hello")

	var dst string

	dstVal := reflect.ValueOf(&dst).Elem()

	nextCalled := false

	mapslice.Keys("test", src, dstVal, func(_ string, _, _ reflect.Value) {
		nextCalled = true
	})

	require.True(t, nextCalled, "next should be called for non-map types")
}

func TestKeys_NotNil(t *testing.T) {
	require.NotNil(t, mapslice.Keys)
}

func TestValues_NamedElementTypeDelegates(t *testing.T) {
	src := reflect.ValueOf(map[string]namedInt{"a": 1})
	dst := reflect.New(reflect.SliceOf(reflect.TypeFor[int]())).Elem()
	nextCalled := false

	mapslice.Values("test", src, dst, func(_ string, _, _ reflect.Value) { nextCalled = true })
	require.True(t, nextCalled)
	require.Zero(t, dst.Len())
}

func TestKeys_NamedElementTypeDelegates(t *testing.T) {
	src := reflect.ValueOf(map[namedString]int{"a": 1})
	dst := reflect.New(reflect.SliceOf(reflect.TypeFor[string]())).Elem()
	nextCalled := false

	mapslice.Keys("test", src, dst, func(_ string, _, _ reflect.Value) { nextCalled = true })
	require.True(t, nextCalled)
	require.Zero(t, dst.Len())
}

func TestValues_PointerMapSource(t *testing.T) {
	values := map[string]int{"a": 1}
	src := reflect.ValueOf(&values)
	dst := reflect.New(reflect.SliceOf(reflect.TypeFor[int]())).Elem()
	nextCalled := false

	mapslice.Values("test", src, dst, func(_ string, _, _ reflect.Value) { nextCalled = true })
	require.False(t, nextCalled)
	require.Equal(t, []int{1}, dst.Interface())
}

func TestKeys_PointerMapSource(t *testing.T) {
	values := map[string]int{"a": 1}
	src := reflect.ValueOf(&values)
	dst := reflect.New(reflect.SliceOf(reflect.TypeFor[string]())).Elem()
	nextCalled := false

	mapslice.Keys("test", src, dst, func(_ string, _, _ reflect.Value) { nextCalled = true })
	require.False(t, nextCalled)
	require.Equal(t, []string{"a"}, dst.Interface())
}

func TestValues_NilPointerMapSourceDelegates(t *testing.T) {
	var values *map[string]int

	src := reflect.ValueOf(values)
	dst := reflect.New(reflect.SliceOf(reflect.TypeFor[int]())).Elem()
	nextCalled := false

	require.NotPanics(t, func() {
		mapslice.Values("test", src, dst, func(_ string, _, _ reflect.Value) { nextCalled = true })
	})
	require.True(t, nextCalled)
}
