// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

// Package typeinfo caches struct field metadata and type relations keyed by
// [reflect.Type]. Caches are bounded and lock-free on reads.
package typeinfo
