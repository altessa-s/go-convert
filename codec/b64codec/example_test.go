// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package b64codec_test

import (
	"encoding/base64"
	"fmt"

	"github.com/altessa-s/go-convert"
	"github.com/altessa-s/go-convert/codec/b64codec"
)

func ExampleNew() {
	type Attachment struct {
		Data []byte
	}

	type AttachmentDTO struct {
		Data string
	}

	opt := convert.WithErrorCodecs(b64codec.New(b64codec.WithEncoding(base64.URLEncoding)))

	dto, err := convert.ConvertE(Attachment{Data: []byte("hello?")}, &AttachmentDTO{}, opt)
	fmt.Println(dto.Data, err)

	att, err := convert.ConvertE(AttachmentDTO{Data: "aGVsbG8_"}, &Attachment{}, opt)
	fmt.Println(string(att.Data), err)

	_, err = convert.ConvertE(AttachmentDTO{Data: "aGVsbG8/"}, &Attachment{}, opt)
	fmt.Println(err)
	// Output:
	// aGVsbG8_ <nil>
	// hello? <nil>
	// b64codec: field "data": illegal base64 data at input byte 7
}
