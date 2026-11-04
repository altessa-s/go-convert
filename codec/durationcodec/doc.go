// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

// Package durationcodec provides an error codec for bidirectional conversion
// between time.Duration and strings or numbers.
//
// Strings use the time package format: time.Duration.String produces values
// such as "1h30m0s" and time.ParseDuration reads them back. Numbers count
// units of time.Second by default; [WithUnit] selects another unit, such as
// time.Millisecond. Integer destinations truncate toward zero, float
// destinations keep the fraction, and float sources such as 1.5 seconds are
// rounded to the nanosecond.
//
// Parse errors, values that overflow the destination and non-finite floats
// are returned as an [*Error], so the codec is an error codec
// ([convcodec.CodecE]) registered with converter.WithErrorCodecs;
// converter.ConvertE returns the error unchanged. An empty string sets the
// destination to its zero value (nil for pointer destinations) and a nil
// source pointer leaves the destination unchanged. Both pointer and
// non-pointer variants are supported.
//
// Example:
//
//	converter.ConvertE(src, &dst, converter.WithErrorCodecs(durationcodec.New(durationcodec.WithUnit(time.Millisecond))))
package durationcodec
