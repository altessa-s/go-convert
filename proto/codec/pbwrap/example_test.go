// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package pbwrap_test

import (
	"fmt"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/altessa-s/go-convert"
	"github.com/altessa-s/go-convert/proto/codec/pbwrap"
)

func ExampleNew() {
	type User struct {
		Nickname string
		Age      int64
	}

	type UserPB struct {
		Nickname *wrapperspb.StringValue
		Age      *wrapperspb.Int64Value
	}

	pb := convert.Convert(User{Nickname: "ada", Age: 36}, &UserPB{}, convert.WithCodecs(pbwrap.New()))
	fmt.Println(pb.Nickname.GetValue(), pb.Age.GetValue())

	user := convert.Convert(pb, &User{}, convert.WithCodecs(pbwrap.New()))
	fmt.Println(user.Nickname, user.Age)
	// Output:
	// ada 36
	// ada 36
}

func ExampleWithIgnoreZeroValues() {
	type User struct{ Nickname string }

	type UserPB struct{ Nickname *wrapperspb.StringValue }

	pb := convert.Convert(User{}, &UserPB{}, convert.WithCodecs(pbwrap.New(pbwrap.WithIgnoreZeroValues())))
	fmt.Println(pb.Nickname == nil)
	// Output: true
}
