// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package textcodec

import (
	"errors"
	"net/netip"
	"reflect"
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

	cached, ok := claims.Load(typePair{src: srcType, dst: dstType})
	require.True(t, ok, "the claim decision must be cached")

	c, _ := cached.(*claim)
	require.Equal(t, &claim{srcType: srcType, dstType: reflect.TypeFor[string](), handled: true, srcMarshaler: true}, c)
	require.Same(t, c, claimOf(srcType, dstType), "a cached pair is not recomputed")

	declined := claimOf(reflect.TypeFor[int](), reflect.TypeFor[string]())
	require.False(t, declined.handled)
	require.Same(t, declined, claimOf(reflect.TypeFor[int](), reflect.TypeFor[string]()), "declined pairs are cached too")
}
