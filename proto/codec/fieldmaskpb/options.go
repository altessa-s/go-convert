// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package fieldmaskpb

//go:generate optgen generate

// options holds configuration options for FieldMask conversion behavior.
type options struct {
	// ignoreEmpty ([WithIgnoreEmpty]) skips conversion when the source has no
	// paths: a FieldMask without paths or an empty slice. The destination is
	// left unchanged.
	ignoreEmpty bool
	// normalize ([WithNormalize]) sorts the paths and removes duplicates and
	// paths covered by a parent path, as FieldMask.Normalize does.
	normalize bool
}
