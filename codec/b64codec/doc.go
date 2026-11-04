// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

// Package b64codec provides an error codec for bidirectional conversion
// between byte slices and base64 strings.
//
// Encoding uses base64.StdEncoding by default; [WithEncoding] selects
// base64.URLEncoding, a raw (unpadded) variant or a custom encoding. Any type
// whose kind is a byte slice and any string kind are supported, through
// pointers as well.
//
// Invalid base64 input is returned as an [*Error], so the codec is an error
// codec ([convcodec.CodecE]) registered with convert.WithErrorCodecs;
// convert.ConvertE returns the error unchanged. An empty string decodes to
// a non-nil empty slice, keeping []byte{} apart from nil; a nil source
// pointer sets the destination to its zero value (a nil slice, "" or a nil
// pointer).
//
// The codec claims every byte slice to string conversion, so register a more
// specific codec, such as one for net.IP, before it.
//
// Example:
//
//	convert.ConvertE(src, &dst, convert.WithErrorCodecs(b64codec.New(b64codec.WithEncoding(base64.RawURLEncoding))))
package b64codec
