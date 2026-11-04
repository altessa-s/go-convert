// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

// Package fieldmaskpb converts fieldmaskpb.FieldMask values and applies them
// to partial updates.
//
// [New] creates a codec for bidirectional conversion between
// fieldmaskpb.FieldMask and a slice of strings holding its paths. Empty masks
// can be skipped with [WithIgnoreEmpty], and [WithNormalize] sorts the paths
// and drops redundant ones. [NewE] is the error codec variant: it validates
// the paths, by syntax or against a message with [WithMessage], and reports
// invalid ones as a [PathError].
//
// [IgnorePaths] turns an update mask into converter ignore paths. A mask lists
// the fields to update while the converter excludes fields, so it returns the
// complement of the mask over the message descriptor, translating protobuf
// field names (display_name) to the lower-cased Go field names the converter
// matches (displayname), and validates every mask path. The package cannot
// return converter options itself: protobuf codec packages may not import the
// root converter package. Pass the result to converter.WithIgnorePaths.
//
// Example:
//
//	paths, err := fieldmaskpb.IgnorePaths(req.GetUpdateMask(), req.GetUser())
//	if err != nil {
//		return err
//	}
//	_, err = converter.ConvertE(req.GetUser(), &user, converter.WithIgnorePaths(paths...))
package fieldmaskpb
