// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package textcodec_test

import (
	"errors"
	"fmt"
	"net/netip"
	"uuid"

	converter "github.com/altessa-s/go-convert"
	"github.com/altessa-s/go-convert/codec/textcodec"
)

func ExampleNew() {
	type Device struct {
		ID   uuid.UUID
		Addr netip.Addr
	}
	type DeviceDTO struct {
		ID   string
		Addr string
	}
	opt := converter.WithErrorCodecs(textcodec.New())

	dto, err := converter.ConvertE(Device{
		ID:   uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8"),
		Addr: netip.MustParseAddr("192.0.2.1"),
	}, &DeviceDTO{}, opt)
	fmt.Println(dto.ID, dto.Addr, err)

	_, err = converter.ConvertE(DeviceDTO{Addr: "300.0.0.1"}, &Device{}, opt)
	var textErr *textcodec.Error
	fmt.Println(errors.As(err, &textErr), textErr.Type)
	// Output:
	// 6ba7b810-9dad-11d1-80b4-00c04fd430c8 192.0.2.1 <nil>
	// true netip.Addr
}
