// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

// Package reflectx provides the reflection helpers shared by the converter
// and its codecs, including the codecs of the nested modules
// (github.com/altessa-s/go-convert/proto, .../googletype and
// .../codec/optionalcodec). It is a public, versioned API: a codec written
// outside this repository may use it too.
//
//   - [IndirectType] dereferences all pointer layers to obtain the underlying type
//   - [IndirectValue] follows non-nil pointer layers to the underlying value
//   - [MakeDst] allocates nil pointer layers of a destination before a write
//   - [SetZero] zeroes a valid, settable value
//   - [Writable] reports whether MakeDst and a write through a destination succeed
//   - [IsPrimitive] classifies a reflect.Kind as a numeric, bool, or string type
//   - [ConvertValue] performs a Go conversion with decimal integer -> string
//     formatting, single rounding to float32 and without the short slice ->
//     array panic
//
// # Compatibility
//
// The nested modules are released together with the root module but may be
// combined with a newer root release by a consumer's module graph, so they
// import only exported packages of the root module, never its internal
// packages. Changes to this package follow semantic versioning of the root
// module.
package reflectx
