// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package enumcodec_test

import (
	"fmt"

	"github.com/altessa-s/go-convert"
	"github.com/altessa-s/go-convert/codec/enumcodec"
)

type Status int

const (
	StatusUnknown Status = iota
	StatusActive
	StatusBlocked
)

func ExampleNew() {
	type User struct{ Status Status }

	type UserDTO struct{ Status string }

	codec, err := enumcodec.New(map[Status]string{StatusActive: "active", StatusBlocked: "blocked"}, enumcodec.WithCaseInsensitive())
	if err != nil {
		fmt.Println(err)
		return
	}

	opt := convert.WithErrorCodecs(codec)

	dto, err := convert.ConvertE(User{Status: StatusBlocked}, &UserDTO{}, opt)
	fmt.Println(dto.Status, err)

	user, err := convert.ConvertE(UserDTO{Status: "ACTIVE"}, &User{}, opt)
	fmt.Println(user.Status == StatusActive, err)

	_, err = convert.ConvertE(UserDTO{Status: "deleted"}, &User{}, opt)
	fmt.Println(err)
	// Output:
	// blocked <nil>
	// true <nil>
	// enumcodec: field "status": unknown enumcodec_test.Status value "deleted"
}

func ExampleNewInt() {
	type User struct{ Status Status }

	type UserRow struct{ Status int16 }

	opt := convert.WithErrorCodecs(enumcodec.MustNewInt(map[Status]int64{StatusActive: 10, StatusBlocked: 20}))

	row, err := convert.ConvertE(User{Status: StatusBlocked}, &UserRow{}, opt)
	fmt.Println(row.Status, err)
	// Output: 20 <nil>
}
