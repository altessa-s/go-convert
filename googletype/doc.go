// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

// Package googletype provides error-returning codecs for the common google.type
// messages from google.golang.org/genproto/googleapis/type:
//
//   - [NewMoney]: money.Money <-> int64 minor units, decimal string, or a
//     struct registered with [WithMoneyStruct].
//   - [NewDate]: date.Date <-> time.Time (UTC midnight) and "YYYY-MM-DD".
//   - [NewLatLng]: latlng.LatLng <-> [2]float64 as [latitude, longitude].
//   - [NewTimeOfDay]: timeofday.TimeOfDay <-> time.Duration since midnight
//     and "HH:MM:SS[.fffffffff]".
//   - [New]: all of the above in one codec.
//
// Every codec is a [convcodec.CodecE]: invalid input, such as an amount that
// overflows int64, a malformed currency code, a Money whose units and nanos
// have opposite signs, a partial date or a latitude out of range, fails the
// conversion with an [Error] that converter.ConvertE returns unchanged. Match
// it with errors.As, or its [ErrorKind] with errors.Is:
//
//	if errors.Is(err, googletype.ErrOverflow) { ... }
//
// A nil source zeroes the destination; with [WithIgnoreZero] it leaves the
// destination unchanged. Both pointer and non-pointer variants are supported
// on either side; a nil pointer destination is allocated.
//
// # Registration order
//
// Register the codecs with converter.WithErrorCodecs. Error codecs always run
// before the codecs registered with converter.WithCodecs, in registration
// order, so:
//
//   - The googletype codecs are disjoint: each one handles only pairs with its
//     own message on one side, so their order relative to each other does not
//     matter.
//   - Register them before any other error codec that claims values by kind
//     (any struct, any string), or that codec converts the message first.
//   - Codecs registered with converter.WithCodecs, such as
//     optionalcodec.Codec, run after them and cannot unwrap a value for them:
//     use pointer fields, not optional.Optional, around google.type values.
//
// Example:
//
//	opts := []convert.Option{
//		convert.WithErrorCodecs(
//			googletype.NewMoney(googletype.WithCurrency("USD")),
//			googletype.NewDate(),
//		),
//	}
//	order, err := convert.ConvertE(src, &orderv1.Order{}, opts...)
package googletype
