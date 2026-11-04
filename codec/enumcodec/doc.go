// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

// Package enumcodec provides error codecs for bidirectional conversion between
// a Go enum type and strings or integers through a user-supplied table.
//
// [New] maps an enum type E to strings and [NewInt] maps it to integers; both
// return a convcodec.CodecE for converter.WithErrorCodecs. A codec claims a
// field pair only when one side is exactly E or *E and the other has string
// (or integer) kind; every other pair is delegated to the rest of the chain.
//
// Values missing from the table are reported as [UnknownValueError]. Use
// [WithUnspecified] to map the zero enum value to "" or 0 and back,
// [WithUnknownAsZero] to map unknown values to zero instead of failing, and
// [WithCaseInsensitive] to parse strings ignoring case. Both pointer and
// non-pointer variants are supported.
//
// Example:
//
//	codec := enumcodec.New(map[Color]string{Red: "red", Green: "green"})
//	_, err := converter.ConvertE(src, &dst, converter.WithErrorCodecs(codec))
package enumcodec
