// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

// Package sqlnull provides an error codec that bridges database/sql null types
// and their plain counterparts.
//
// [New] recognizes the generic sql.Null[T] and the legacy sql.NullString,
// NullInt64, NullInt32, NullInt16, NullByte, NullFloat64, NullBool and
// NullTime by reflection ([IsNullType]). Valid=false maps to a nil pointer or
// the zero value, and a valid value maps to a pointer to it or the value
// itself. As with codec/optionalcodec, a null type paired with a type the codec
// does not bridge is unwrapped or wrapped and the inner conversion is delegated
// to the rest of the chain, so register it before codecs such as
// codec/unixtime. Use [WithZeroAsNull] to store zero values as NULL.
//
// Example:
//
//	_, err := converter.ConvertE(row, &entity, converter.WithErrorCodecs(sqlnull.New()))
package sqlnull
