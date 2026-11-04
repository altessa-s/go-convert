// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

// Package durpb provides a codec for bidirectional conversion between
// durationpb.Duration and time.Duration.
//
// Zero-value durations can be skipped with [WithIgnoreZero]. Both pointer and
// non-pointer variants are supported.
//
// time.Duration <-> int64 is not handled here: time.Duration has an int64 kind,
// so the converter's built-in convertible path maps it without a codec.
//
// Uses unsafe pointer arithmetic for zero-allocation reads and writes when the
// underlying reflect.Value is addressable; falls back to standard reflect otherwise.
//
// Values outside the time.Duration range saturate at the nearest representable
// duration, matching [durationpb.Duration.AsDuration]. Values that originated as
// a time.Duration round-trip exactly.
//
// # Usage
//
//	codec := durpb.New(durpb.WithIgnoreZero())
package durpb
