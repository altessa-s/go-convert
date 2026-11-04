// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

// Package engine implements reflection-based value conversion. It is
// configured once through [Config] and is safe for concurrent use; each call
// keeps its traversal state, including the nesting depth, in a pooled walker.
package engine
