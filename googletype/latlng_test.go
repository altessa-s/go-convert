// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package googletype_test

import (
	"math"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-convert/googletype"

	"google.golang.org/genproto/googleapis/type/latlng"
)

func TestNewLatLng(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		coords [2]float64
		valid  bool
	}{
		{name: "paris", coords: [2]float64{48.8566, 2.3522}, valid: true},
		{name: "south west", coords: [2]float64{-33.8688, -151.2093}, valid: true},
		{name: "bounds", coords: [2]float64{90, -180}, valid: true},
		{name: "other bounds", coords: [2]float64{-90, 180}, valid: true},
		{name: "origin", coords: [2]float64{0, 0}, valid: true},
		{name: "latitude above 90", coords: [2]float64{90.0001, 0}},
		{name: "latitude below -90", coords: [2]float64{-91, 0}},
		{name: "longitude above 180", coords: [2]float64{0, 180.5}},
		{name: "longitude below -180", coords: [2]float64{0, -181}},
		{name: "NaN latitude", coords: [2]float64{math.NaN(), 0}},
		{name: "infinite longitude", coords: [2]float64{0, math.Inf(1)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			codec := googletype.NewLatLng()

			var ll *latlng.LatLng
			err := run(codec, reflect.ValueOf(tt.coords), &ll)
			if !tt.valid {
				requireKind(t, err, googletype.ErrOutOfRange)
				require.Nil(t, ll)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.coords, [2]float64{ll.GetLatitude(), ll.GetLongitude()})
			}

			msg := &latlng.LatLng{Latitude: tt.coords[0], Longitude: tt.coords[1]}
			var coords [2]float64
			err = run(codec, reflect.ValueOf(msg), &coords)
			if !tt.valid {
				requireKind(t, err, googletype.ErrOutOfRange)
				require.Equal(t, [2]float64{}, coords)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.coords, coords)
		})
	}
}

func TestNewLatLng_NilZeroAndDelegation(t *testing.T) {
	t.Parallel()

	codec := googletype.NewLatLng()

	coords := [2]float64{1, 2}
	require.NoError(t, run(codec, reflect.ValueOf((*latlng.LatLng)(nil)), &coords))
	require.Equal(t, [2]float64{1, 2}, coords)

	ignoreZero := googletype.NewLatLng(googletype.WithIgnoreZero())
	require.NoError(t, run(ignoreZero, reflect.ValueOf(&latlng.LatLng{}), &coords))
	require.Equal(t, [2]float64{1, 2}, coords)
	var ll *latlng.LatLng
	require.NoError(t, run(ignoreZero, reflect.ValueOf([2]float64{}), &ll))
	require.Nil(t, ll)

	var pair [2]float32
	require.ErrorIs(t, run(codec, reflect.ValueOf(&latlng.LatLng{}), &pair), errNext)
	var slice []float64
	require.ErrorIs(t, run(codec, reflect.ValueOf(&latlng.LatLng{}), &slice), errNext)
}
