// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package engine

import (
	"maps"
	"reflect"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-convert/internal/typeinfo"

	convcodec "github.com/altessa-s/go-convert/codec"
)

type planInner struct{ X int }

type PlanEmbedded struct{ E int }

type planSrc struct {
	PlanEmbedded
	A      int
	B      int32
	C      string
	D      *int
	In     planInner
	S      []int
	Gone   int
	hidden int //nolint:unused // checks that unexported fields are skipped
}

type planDst struct {
	A  int
	B  int64
	C  string
	D  *int
	In planInner
	S  []int
}

// planKinds summarizes a plan as source field name -> step kind.
func planKinds(p *structPlan) map[string]stepKind {
	kinds := make(map[string]stepKind, len(p.steps))
	for _, step := range p.steps {
		kinds[step.name] = step.kind
	}

	return kinds
}

func set(names ...string) map[string]struct{} {
	m := make(map[string]struct{}, len(names))
	for _, n := range names {
		m[n] = struct{}{}
	}

	return m
}

func passthrough(name string, src, dst reflect.Value, next convcodec.CodecHandler) {
	next(name, src, dst)
}

func TestBuildPlanChoosesSteps(t *testing.T) {
	t.Parallel()

	src, dst := reflect.TypeFor[planSrc](), reflect.TypeFor[planDst]()
	fast := map[string]stepKind{
		"planembedded": stepEmbedded, "a": stepAssign, "b": stepPrimitive, "c": stepAssign,
		"d": stepAssign, "in": stepAssign, "s": stepAssign,
	}
	allGeneric := map[string]stepKind{
		"planembedded": stepEmbedded, "a": stepGeneric, "b": stepGeneric, "c": stepGeneric,
		"d": stepGeneric, "in": stepGeneric, "s": stepGeneric,
	}
	with := func(base map[string]stepKind, changes map[string]stepKind, drop ...string) map[string]stepKind {
		out := make(map[string]stepKind, len(base))
		maps.Copy(out, base)
		maps.Copy(out, changes)

		for _, k := range drop {
			delete(out, k)
		}

		return out
	}

	tests := []struct {
		name string
		cfg  Config
		want map[string]stepKind
	}{
		{name: "default", want: fast},
		{name: "nil and zero modes keep fast steps", cfg: Config{IgnoreNilValues: true, IgnoreZeroValues: true}, want: fast},
		{name: "overflow check keeps fast steps", cfg: Config{OverflowCheck: true}, want: fast},
		{
			name: "codecs keep assignment only",
			cfg:  Config{Codecs: convcodec.NewCodecsSet(passthrough)},
			want: with(fast, map[string]stepKind{"b": stepGeneric}),
		},
		{
			name: "error codecs are generic",
			cfg: Config{Codecs: convcodec.NewCodecsSet().AddError(
				func(p string, s, d reflect.Value, next convcodec.CodecHandlerE) error { return next(p, s, d) })},
			want: allGeneric,
		},
		{name: "ignore paths are generic", cfg: Config{IgnorePaths: set("zzz")}, want: allGeneric},
		{name: "sparse merge walks structs", cfg: Config{SparseMerge: true}, want: with(fast, map[string]stepKind{"in": stepGeneric})},
		{name: "copy slices clones slices", cfg: Config{CopySlices: true}, want: with(fast, map[string]stepKind{"s": stepGeneric})},
		{name: "ignored fields are dropped", cfg: Config{IgnoreFields: set("c", "b")}, want: with(fast, nil, "c", "b")},
		{
			name: "mappings redirect and drop",
			cfg:  Config{FieldMappings: map[string]string{"a": "c", "c": "missing"}},
			want: with(fast, map[string]stepKind{"a": stepGeneric}, "c"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := New(tt.cfg)
			plan := e.planFor(src, dst)
			require.Equal(t, tt.want, planKinds(plan))
			require.Same(t, plan, e.planFor(src, dst), "plans are cached")

			for _, step := range plan.steps {
				require.NotEqual(t, "gone", step.name, "missing destination fields are dropped")
				require.NotEqual(t, "hidden", step.name, "unexported fields are dropped")
				require.Equal(t, step.kind == stepPrimitive, step.convert != nil)
			}
		})
	}
}

func TestBuildPlanResolvesIndices(t *testing.T) {
	t.Parallel()

	e := New(Config{FieldMappings: map[string]string{"a": "c"}})
	plan := e.planFor(reflect.TypeFor[planSrc](), reflect.TypeFor[planDst]())

	byName := make(map[string]fieldStep)
	for _, step := range plan.steps {
		byName[step.name] = step
	}

	require.Equal(t, 1, byName["a"].src)
	require.Equal(t, 2, byName["a"].dst, "a maps to destination field c")
	require.Equal(t, -1, byName["planembedded"].dst, "the destination has no embedded field")
	require.True(t, byName["d"].srcPointer)
	require.False(t, byName["b"].srcPointer)
}

func TestPlanCacheSharing(t *testing.T) {
	t.Parallel()

	require.Same(t, sharedPlans, New(Config{}).plans)
	require.Same(t, sharedPlans, New(Config{SparseMerge: true, IgnorePaths: set("a")}).plans)
	require.NotSame(t, sharedPlans, New(Config{IgnoreFields: set("a")}).plans)
	require.NotSame(t, sharedPlans, New(Config{FieldMappings: map[string]string{"a": "b"}}).plans)

	// Shared plans are keyed by the flags they depend on.
	src, dst := reflect.TypeFor[planSrc](), reflect.TypeFor[planDst]()
	require.Equal(t, stepAssign, planKinds(New(Config{}).planFor(src, dst))["in"])
	require.Equal(t, stepGeneric, planKinds(New(Config{SparseMerge: true}).planFor(src, dst))["in"])
}

func syntheticStruct(i int) reflect.Type {
	return reflect.StructOf([]reflect.StructField{{
		Name: "Value",
		Type: reflect.TypeFor[int](),
		Tag:  reflect.StructTag(`index:"` + strconv.Itoa(i) + `"`),
	}})
}

func TestPlanCacheIsBoundedUnderConcurrency(t *testing.T) {
	t.Parallel()

	e := New(Config{IgnoreFields: set("other")})
	dst := reflect.TypeFor[planDst]()

	var wg sync.WaitGroup
	for i := range typeinfo.MaxTypes + 64 {
		wg.Go(func() {
			plan := e.planFor(syntheticStruct(i), dst)
			if len(plan.steps) != 0 {
				panic("synthetic struct has no matching destination field")
			}
		})
	}

	wg.Wait()
	require.LessOrEqual(t, e.plans.Len(), typeinfo.MaxTypes)
}

func TestPlannedConversionIsConcurrencySafe(t *testing.T) {
	t.Parallel()

	e := New(Config{OverflowCheck: true})

	var wg sync.WaitGroup
	for i := range 64 {
		wg.Go(func() {
			n := i

			var out planDst
			if err := e.Convert(&planSrc{A: i, B: int32(i), C: "c", D: &n, In: planInner{X: i}, S: []int{i}}, &out); err != nil {
				panic(err)
			}

			want := planDst{A: i, B: int64(i), C: "c", D: &n, In: planInner{X: i}, S: []int{i}}
			if !reflect.DeepEqual(want, out) {
				panic("unexpected conversion result")
			}
		})
	}

	wg.Wait()
}

func TestPlannedStepsKeepSemantics(t *testing.T) {
	t.Parallel()

	type narrowSrc struct{ V int64 }

	type narrowDst struct{ V int8 }

	t.Run("overflow", func(t *testing.T) {
		t.Parallel()

		err := New(Config{OverflowCheck: true}).Convert(&narrowSrc{V: 300}, &narrowDst{})
		require.Equal(t, OverflowError{Field: "v", From: reflect.Int64, To: reflect.Int8, Value: int64(300)}, err)
	})
	t.Run("unaddressable source", func(t *testing.T) {
		t.Parallel()

		var out narrowDst
		require.NoError(t, New(Config{}).Convert(narrowSrc{V: 7}, &out))
		require.Equal(t, int8(7), out.V)
	})
	t.Run("ignore zero and nil", func(t *testing.T) {
		t.Parallel()

		out := planDst{A: 1, B: 2, C: "keep", S: []int{9}}
		require.NoError(t, New(Config{IgnoreZeroValues: true, IgnoreNilValues: true}).Convert(&planSrc{}, &out))
		require.Equal(t, planDst{A: 1, B: 2, C: "keep", S: []int{9}}, out)
	})
	t.Run("depth limit applies to planned leaves", func(t *testing.T) {
		t.Parallel()

		type outer struct{ In planInner }

		err := New(Config{MaxDepth: 1}).Convert(&outer{In: planInner{X: 1}}, &planDst{})
		require.NoError(t, err, "a struct assigned as a leaf needs one level")

		type outerDst struct{ In struct{ X int64 } }

		err = New(Config{MaxDepth: 1}).Convert(&outer{In: planInner{X: 1}}, &outerDst{})
		require.Equal(t, DepthError{Limit: 1, Path: ""}, err)
	})
}
