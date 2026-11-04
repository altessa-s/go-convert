// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

// Package enumcodec provides error codecs for bidirectional conversion between
// a Go enum type and strings or integers through a user-supplied table.
//
// [New] maps an enum type E to strings and [NewInt] maps it to integers; both
// return a convcodec.CodecE for convert.WithErrorCodecs. A codec claims a
// field pair only when one side is exactly E or *E and the other is the
// predeclared string type (or a predeclared integer type, such as int32);
// every other pair, including a pair of two enum types, is delegated to the
// rest of the chain. A nil source zeroes the destination.
//
// Values missing from the table are reported as [UnknownValueError]. Use
// [WithUnspecified] to map the zero enum value to "" or 0 and back,
// [WithUnknownAsZero] to map unknown values to zero instead of failing, and
// [WithCaseInsensitive] to parse strings ignoring case ([NewInt] rejects
// it). Both pointer and non-pointer variants are supported. An invalid table
// is a [ConfigError]; [MustNew] and [MustNewInt] panic with it instead.
//
// Example:
//
//	codec := enumcodec.MustNew(map[Color]string{Red: "red", Green: "green"})
//	_, err := convert.ConvertE(src, &dst, convert.WithErrorCodecs(codec))
package enumcodec
