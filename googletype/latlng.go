// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package googletype

import (
	"reflect"

	"google.golang.org/genproto/googleapis/type/latlng"

	convcodec "github.com/altessa-s/go-convert/codec"
)

const (
	// maxLatitude bounds the latitude in degrees.
	maxLatitude = 90
	// maxLongitude bounds the longitude in degrees.
	maxLongitude = 180
)

var latLngType = reflect.TypeFor[latlng.LatLng]()

// NewLatLng creates a CodecE for latlng.LatLng.
//
// Supported conversions:
//   - latlng.LatLng <-> [2]float64 as [latitude, longitude] in degrees.
//   - Pointer variants of the above.
//
// Both directions require a latitude in [-90, 90] and a longitude in
// [-180, 180]; anything else, NaN included, fails with [ErrOutOfRange].
//
// Example:
//
//	codec := googletype.NewLatLng()
func NewLatLng(opts ...Option) convcodec.CodecE {
	return newCodec(buildOptions(opts), convertLatLng)
}

// convertLatLng handles the pairs with latlng.LatLng on one side.
func convertLatLng(p pair, o *options) (bool, error) {
	switch {
	case p.srcType == latLngType && p.dstType == latLngPair:
		if !p.src.IsValid() {
			clearNil(p, o)

			return true, nil
		}

		ll := messagePtr[latlng.LatLng](p.src)

		coords := [2]float64{ll.GetLatitude(), ll.GetLongitude()}
		if o.ignoreZero && coords == [2]float64{} {
			return true, nil
		}

		if err := validateLatLng(p.path, coords); err != nil {
			return true, err
		}

		setValue(p.dst, p.dstType, coords)

		return true, nil
	case p.srcType == latLngPair && p.dstType == latLngType:
		if !p.src.IsValid() {
			clearNil(p, o)

			return true, nil
		}

		coords := readValue[[2]float64](p.src)
		if o.ignoreZero && coords == [2]float64{} {
			return true, nil
		}

		if err := validateLatLng(p.path, coords); err != nil {
			return true, err
		}

		ll := messageDst[latlng.LatLng](p.dst, p.dstType)
		ll.Latitude, ll.Longitude = coords[0], coords[1]

		return true, nil
	default:
		return false, nil
	}
}

// validateLatLng checks that coords hold a latitude and longitude in range.
// The comparisons are written so that NaN fails them.
func validateLatLng(path string, coords [2]float64) error {
	if !(coords[0] >= -maxLatitude && coords[0] <= maxLatitude) {
		return newError(path, ErrOutOfRange, "latitude %v outside [-90, 90]", coords[0])
	}

	if !(coords[1] >= -maxLongitude && coords[1] <= maxLongitude) {
		return newError(path, ErrOutOfRange, "longitude %v outside [-180, 180]", coords[1])
	}

	return nil
}
