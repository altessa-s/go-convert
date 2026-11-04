// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package mapslice_test

import (
	"fmt"
	"slices"

	"github.com/altessa-s/go-convert"
	"github.com/altessa-s/go-convert/codec/mapslice"
)

func ExampleValues() {
	type Team struct{ Members map[string]string }

	type TeamDTO struct{ Members []string }

	dto := convert.Convert(Team{Members: map[string]string{"a": "Ada", "b": "Bob"}}, &TeamDTO{}, convert.WithCodecs(mapslice.Values))

	slices.Sort(dto.Members) // map iteration order is unspecified
	fmt.Println(dto.Members)
	// Output: [Ada Bob]
}

func ExampleKeys() {
	type Team struct{ Members map[string]string }

	type TeamDTO struct{ Members []string }

	dto := convert.Convert(Team{Members: map[string]string{"a": "Ada", "b": "Bob"}}, &TeamDTO{}, convert.WithCodecs(mapslice.Keys))

	slices.Sort(dto.Members)
	fmt.Println(dto.Members)
	// Output: [a b]
}
