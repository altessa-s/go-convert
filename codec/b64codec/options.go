// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package b64codec

import "encoding/base64"

//go:generate optgen generate

// options holds configuration options for base64 conversion behavior.
type options struct {
	// encoding ([WithEncoding]) is the base64 alphabet and padding used in
	// both directions: base64.StdEncoding (the default), base64.URLEncoding,
	// base64.RawStdEncoding, base64.RawURLEncoding or a custom encoding. A nil
	// encoding is ignored.
	encoding *base64.Encoding `optgen:"default=base64.StdEncoding"`
}
