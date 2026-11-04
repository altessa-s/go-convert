// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

// Package reflect provides low-level reflection utilities shared across
// the converter and its codec sub-packages.
//
//   - [IndirectType] dereferences all pointer layers to obtain the underlying type
//   - [MakeDst] ensures a pointer destination is initialized before assignment
//   - [IsPrimitive] classifies a reflect.Kind as a numeric, bool, or string type
//   - [ConvertValue] performs a Go conversion with decimal integer -> string
//     formatting and without the short slice -> array panic
//
// This is an internal package; its API is not part of the public contract.
package reflect
