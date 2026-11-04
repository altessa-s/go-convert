// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package engine

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

type kindSrc struct {
	Name string
	N    int32
}

type kindDst struct {
	Name string
	N    int64
}

type celsius float64

func TestConvertByKindDispatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		src  any
		dst  any // pointer to the destination
		want any
	}{
		{name: "struct to struct", src: kindSrc{Name: "a", N: 1}, dst: new(kindDst), want: kindDst{Name: "a", N: 1}},
		{name: "struct pointer to struct pointer", src: &kindSrc{Name: "p"}, dst: new(*kindDst), want: &kindDst{Name: "p"}},
		{name: "nil struct pointer zeroes", src: (*kindSrc)(nil), dst: func() any { d := &kindDst{Name: "x"}; return &d }(), want: (*kindDst)(nil)},
		{name: "struct to slice of struct", src: kindDst{Name: "one"}, dst: new([]kindDst), want: []kindDst{{Name: "one"}}},
		{name: "slice", src: []kindSrc{{N: 1}, {N: 2}}, dst: new([]kindDst), want: []kindDst{{N: 1}, {N: 2}}},
		{name: "nil slice", src: []kindSrc(nil), dst: func() any { d := []kindDst{{}}; return &d }(), want: []kindDst(nil)},
		{name: "array", src: [2]int32{1, 2}, dst: new([2]int64), want: [2]int64{1, 2}},
		{name: "map", src: map[string]kindSrc{"k": {N: 3}}, dst: new(map[string]kindDst), want: map[string]kindDst{"k": {N: 3}}},
		{name: "map with converted keys", src: map[int32]int32{1: 2}, dst: new(map[int64]int64), want: map[int64]int64{1: 2}},
		{name: "convertible", src: 21.5, dst: new(celsius), want: celsius(21.5)},
		{name: "convertible into pointer", src: 3.5, dst: new(*celsius), want: func() *celsius { c := celsius(3.5); return &c }()},
		{name: "not convertible leaves zero", src: kindSrc{}, dst: new(*string), want: (*string)(nil)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			w := New(Config{}).walker()
			defer w.release()

			dst := reflect.ValueOf(tt.dst).Elem()
			w.convertByKind("", "", reflect.ValueOf(tt.src), dst)
			require.Equal(t, tt.want, dst.Interface())
			require.Zero(t, w.depth)
		})
	}
}

func TestConvertByKindOverflow(t *testing.T) {
	t.Parallel()

	w := New(Config{OverflowCheck: true}).walker()
	defer w.release()

	var dst int8

	require.PanicsWithValue(t, OverflowError{Field: "f", From: reflect.Int, To: reflect.Int8, Value: 1000}, func() {
		w.convertByKind("", "f", reflect.ValueOf(1000), reflect.ValueOf(&dst).Elem())
	})
}

type dynamicHolder struct {
	V any
}

func TestDynamicTraversal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		ignore  []string
		src     dynamicHolder
		dst     dynamicHolder
		want    dynamicHolder
		wantErr bool
	}{
		{
			name:   "interface struct keeps ignored field",
			ignore: []string{"v.name"},
			src:    dynamicHolder{V: kindSrc{Name: "new", N: 2}},
			dst:    dynamicHolder{V: kindSrc{Name: "old", N: 1}},
			want:   dynamicHolder{V: kindSrc{Name: "old", N: 2}},
		},
		{
			name:   "map string keys are lower-cased path segments",
			ignore: []string{"v.keep"},
			src:    dynamicHolder{V: map[string]int{"Keep": 1, "Other": 2}},
			dst:    dynamicHolder{V: map[string]int{"Keep": 7}},
			want:   dynamicHolder{V: map[string]int{"Keep": 7, "Other": 2}},
		},
		{
			name:   "ignored destination keys survive when the source omits them",
			ignore: []string{"v.gone"},
			src:    dynamicHolder{V: map[string]int{"a": 1}},
			dst:    dynamicHolder{V: map[string]int{"gone": 5, "b": 3}},
			want:   dynamicHolder{V: map[string]int{"gone": 5, "a": 1}},
		},
		{
			name:   "nested interface values",
			ignore: []string{"v.inner.secret"},
			src:    dynamicHolder{V: map[string]any{"inner": map[string]any{"secret": "s", "x": 1}}},
			dst:    dynamicHolder{V: map[string]any{"inner": map[string]any{"secret": "kept"}}},
			want:   dynamicHolder{V: map[string]any{"inner": map[string]any{"secret": "kept", "x": 1}}},
		},
		{
			name:   "nil interface clears the destination",
			ignore: []string{"v.x"},
			src:    dynamicHolder{},
			dst:    dynamicHolder{V: 1},
			want:   dynamicHolder{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dst := tt.dst
			err := New(Config{IgnorePaths: set(tt.ignore...)}).Convert(&tt.src, &dst)
			require.NoError(t, err)
			require.Equal(t, tt.want, dst)
		})
	}
}

func TestDynamicTraversalBreaksCycles(t *testing.T) {
	t.Parallel()

	cyclic := map[string]any{"secret": "s", "x": 1}
	cyclic["self"] = cyclic
	dst := dynamicHolder{V: map[string]any{"secret": "kept"}}

	err := New(Config{IgnorePaths: set("v.secret", "v.self.secret")}).Convert(&dynamicHolder{V: cyclic}, &dst)
	require.NoError(t, err)

	out, ok := dst.V.(map[string]any)
	require.True(t, ok)
	require.Equal(t, "kept", out["secret"])
	require.Equal(t, 1, out["x"])
	require.Contains(t, out, "self")
}

func TestDynamicTraversalHonorsTheDepthLimit(t *testing.T) {
	t.Parallel()

	deep := any(1)
	for range 20 {
		deep = map[string]any{"n": deep}
	}

	err := New(Config{MaxDepth: 5, IgnorePaths: set("v.n.n.n.n.n.n.n.n.x")}).Convert(&dynamicHolder{V: deep}, &dynamicHolder{})

	var depthErr DepthError
	require.ErrorAs(t, err, &depthErr)
	require.Equal(t, 5, depthErr.Limit)
}

func TestMappedFieldIndex(t *testing.T) {
	t.Parallel()

	w := &walker{Engine: New(Config{FieldMappings: map[string]string{"a": "n", "b": "missing"}})}
	info := w.types.Get(reflect.TypeFor[kindDst]())

	idx, ok := w.mappedFieldIndex("name", 0, info)
	require.True(t, ok)
	require.Equal(t, 0, idx)
	idx, ok = w.mappedFieldIndex("a", 0, info)
	require.True(t, ok)
	require.Equal(t, 1, idx)

	_, ok = w.mappedFieldIndex("b", 0, info)
	require.False(t, ok)
}
