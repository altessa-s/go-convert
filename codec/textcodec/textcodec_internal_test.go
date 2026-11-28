// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package textcodec

import (
	"errors"
	"net/netip"
	"reflect"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// errNext is returned by failNext.
var errNext = errors.New("next called")

// failNext is a next handler that must not be reached.
func failNext(string, reflect.Value, reflect.Value) error { return errNext }

func TestClaimIsCachedPerTypePair(t *testing.T) {
	t.Parallel()

	codec := New()
	srcType, dstType := reflect.TypeFor[netip.Addr](), reflect.TypeFor[*string]()

	var dst *string
	require.NoError(t, codec("Addr", reflect.ValueOf(netip.MustParseAddr("10.0.0.1")), reflect.ValueOf(&dst).Elem(), failNext))
	require.Equal(t, "10.0.0.1", *dst)

	cached, ok := claims.entries.Load(typePair{src: srcType, dst: dstType})
	require.True(t, ok, "the claim decision must be cached")

	c, _ := cached.(*claim)
	require.Equal(t, &claim{srcType: srcType, dstType: reflect.TypeFor[string](), handled: true, srcMarshaler: true}, c)
	require.Same(t, c, claimOf(srcType, dstType), "a cached pair is not recomputed")

	declined := claimOf(reflect.TypeFor[int](), reflect.TypeFor[string]())
	require.False(t, declined.handled)
	require.Same(t, declined, claimOf(reflect.TypeFor[int](), reflect.TypeFor[string]()), "declined pairs are cached too")
}

func TestClaimCacheIsBounded(t *testing.T) {
	t.Parallel()

	var cache claimCache

	var wg sync.WaitGroup
	for i := range maxClaims + 64 {
		wg.Go(func() {
			src := reflect.StructOf([]reflect.StructField{{
				Name: "V",
				Type: reflect.TypeFor[int](),
				Tag:  reflect.StructTag(`n:"` + strconv.Itoa(i) + `"`),
			}})
			dst := reflect.TypeFor[string]()
			cache.publish(typePair{src: src, dst: dst}, &claim{srcType: src, dstType: dst})
		})
	}

	wg.Wait()

	require.Equal(t, maxClaims, cache.count)
}

func TestClaimCacheReusesPublishedClaim(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name    string
		prefill int
	}{
		{name: "room remaining"},
		{name: "last slot", prefill: maxClaims - 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var cache claimCache

			src, dst := reflect.TypeFor[int](), reflect.TypeFor[string]()
			for i := range tt.prefill {
				array := reflect.ArrayOf(i, src)
				cache.publish(typePair{src: array, dst: dst}, &claim{srcType: array, dstType: dst})
			}

			// Both callers have missed the same key and computed a claim before either publishes.
			key := typePair{src: src, dst: dst}
			first := &claim{srcType: src, dstType: dst}
			second := &claim{srcType: src, dstType: dst}

			if got := cache.publish(key, first); got != first {
				t.Fatalf("first publication returned %p, want %p", got, first)
			}

			if got := cache.publish(key, second); got != first {
				t.Errorf("second publication returned %p, want the cached claim %p", got, first)
			}

			if got, want := cache.count, tt.prefill+1; got != want {
				t.Errorf("cache contains %d claims, want %d", got, want)
			}
		})
	}
}
