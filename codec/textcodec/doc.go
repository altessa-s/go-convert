// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

// Package textcodec provides an error codec that converts between types
// implementing [encoding.TextMarshaler] or [encoding.TextUnmarshaler] and
// their text form.
//
// A source whose type implements TextMarshaler (on the value or the pointer)
// is marshaled into a string or []byte destination. A string or []byte
// source is unmarshaled into a destination whose pointer type implements
// TextUnmarshaler. When the source is a TextMarshaler and the destination a
// TextUnmarshaler of a different type, the text is passed from one to the
// other. This covers uuid.UUID, netip.Addr, netip.Prefix, big.Int,
// time.Time, slog.Level and user types alike.
//
// Failures from MarshalText or UnmarshalText are returned as an [*Error], so
// the codec is an error codec ([convcodec.CodecE]) registered with
// convert.WithErrorCodecs; convert.ConvertE returns the error unchanged.
//
// Empty text sets the destination to its zero value (a nil pointer for
// pointer destinations) without calling UnmarshalText; [WithParseEmpty]
// passes it to UnmarshalText instead. A nil source pointer also sets the
// destination to its zero value. Pointer variants are supported on both
// sides.
//
// MarshalText with a pointer receiver is called on a copy of the source, so a
// method that caches into its receiver never modifies the source. The claim
// decision of each source and destination type pair is cached, so the
// interface checks run once per pair.
//
// Example:
//
//	convert.ConvertE(src, &dst, convert.WithErrorCodecs(textcodec.New()))
package textcodec
