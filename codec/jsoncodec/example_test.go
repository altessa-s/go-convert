// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package jsoncodec_test

import (
	"fmt"

	"github.com/altessa-s/go-convert"
	"github.com/altessa-s/go-convert/codec/jsoncodec"

	stdjson "encoding/json"
)

func ExampleNew() {
	type Order struct {
		ID    int
		Items stdjson.RawMessage
	}

	type Item struct {
		SKU string `json:"sku"`
		Qty int    `json:"qty"`
	}

	type OrderModel struct {
		ID    int
		Items []Item
	}

	codec := jsoncodec.New()

	model, err := convert.ConvertE(Order{ID: 7, Items: stdjson.RawMessage(`[{"sku":"A1","qty":2}]`)}, &OrderModel{}, convert.WithErrorCodecs(codec))
	fmt.Println(model.Items, err)

	order, err := convert.ConvertE(*model, &Order{}, convert.WithErrorCodecs(codec))
	fmt.Println(string(order.Items), err)

	_, err = convert.ConvertE(Order{Items: stdjson.RawMessage(`{"sku":"A1"}`)}, &OrderModel{}, convert.WithErrorCodecs(codec))
	fmt.Println(err != nil)
	// Output:
	// [{A1 2}] <nil>
	// [{"sku":"A1","qty":2}] <nil>
	// true
}

func ExampleWithStrings() {
	type Settings struct {
		Theme string `json:"theme"`
		Size  int    `json:"size"`
	}

	type Profile struct{ Settings Settings }

	type ProfileRow struct{ Settings string }

	opt := convert.WithErrorCodecs(jsoncodec.New(jsoncodec.WithStrings()))

	row, err := convert.ConvertE(Profile{Settings: Settings{Theme: "dark", Size: 12}}, &ProfileRow{}, opt)
	fmt.Println(row.Settings, err)

	profile, err := convert.ConvertE(ProfileRow{Settings: `{"theme":"light","size":14}`}, &Profile{}, opt)
	fmt.Println(profile.Settings, err)
	// Output:
	// {"theme":"dark","size":12} <nil>
	// {light 14} <nil>
}
