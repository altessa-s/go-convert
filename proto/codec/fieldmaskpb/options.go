// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package fieldmaskpb

//go:generate optgen generate

import "google.golang.org/protobuf/proto"

// options holds configuration options for FieldMask conversion behavior.
type options struct {
	// ignoreEmpty ([WithIgnoreEmpty]) skips conversion when the source has no
	// paths: a nil or empty FieldMask, or a nil or empty slice. The
	// destination is left unchanged.
	ignoreEmpty bool
	// normalize ([WithNormalize]) sorts the paths and removes duplicates and
	// paths covered by a parent path, as FieldMask.Normalize does.
	normalize bool
	// message ([WithMessage]) makes [NewE] validate paths against the
	// descriptor of this message instead of checking their syntax only. Only
	// the descriptor is used; a typed nil pointer is fine. [New] ignores it.
	message proto.Message `opt:"Message"`
}
