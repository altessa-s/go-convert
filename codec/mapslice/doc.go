// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

// Package mapslice provides codecs for converting map values or keys to slices.
//
// Two package-level codecs are available:
//
//   - [Values] extracts map values into a destination slice
//   - [Keys] extracts map keys into a destination slice
//
// Element types must match or the destination must be []any.
// Iteration order follows Go map semantics (non-deterministic).
//
// Example:
//
//	convert.Convert(src, &dst, convert.WithCodecs(mapslice.Values))
package mapslice
