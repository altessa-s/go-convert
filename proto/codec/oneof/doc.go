// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

// Package oneof provides a codec for bidirectional conversion between
// struct-based oneof representations and protobuf oneof interface types.
//
// Conversion directions:
//
//   - Struct with optional pointer fields → protobuf oneof interface wrapper
//   - Protobuf oneof interface wrapper → struct with optional pointer fields
//
// Field values follow Go's conversion rules, except that an integer
// converted to a string is formatted in decimal (65 becomes "65", not "A")
// and a slice shorter than the destination array is not converted (a plain
// reflect conversion would panic). A zero [reflect.Value] or a nil struct
// pointer is passed to the next codec.
//
// Field matching is case-insensitive. Use [NewForField] to scope the codec
// to a specific field rather than applying it globally.
//
// Example:
//
//	codec := oneof.NewForField("Payload",
//	    oneof.WithWrapperRegistry(map[string]any{
//	        "Contractor": &pb.Invitation_Contractor{},
//	    }),
//	)
package oneof
