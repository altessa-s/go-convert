// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package googletype

import (
	"reflect"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMoneyShape_CachedPerType(t *testing.T) {
	t.Parallel()

	type price struct {
		Currency string
		Amount   int64
	}

	type other struct{ Value int64 }

	o := buildOptions([]Option{WithMoneyStruct("Amount", "Currency")})

	amount, currency, ok := o.moneyShape(reflect.TypeFor[price]())
	require.True(t, ok)
	require.Equal(t, []int{1}, amount)
	require.Equal(t, []int{0}, currency)

	// The resolved indexes are cached for later lookups, which may run
	// concurrently.
	cached, found := o.moneyShapes.Load(reflect.TypeFor[price]())
	require.True(t, found)
	require.Equal(t, moneyStructShape{amount: []int{1}, currency: []int{0}, ok: true}, cached)

	var wg sync.WaitGroup

	for range 8 {
		wg.Go(func() {
			if again, _, ok := o.moneyShape(reflect.TypeFor[price]()); !ok || again[0] != 1 {
				t.Error("unexpected cached shape")
			}
		})
	}

	wg.Wait()

	_, _, ok = o.moneyShape(reflect.TypeFor[other]())
	require.False(t, ok)

	cached, found = o.moneyShapes.Load(reflect.TypeFor[other]())
	require.True(t, found, "unsupported struct types are cached too")
	require.Equal(t, moneyStructShape{}, cached)
}
