// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package convert_test

import (
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"
	"time"

	converter "github.com/altessa-s/go-convert"
	convcodec "github.com/altessa-s/go-convert/codec"
)

type exampleAddress struct {
	City string
	Zip  string
}

type exampleUser struct {
	ID       int64
	Name     string
	Password string
	Tags     []string
	Address  *exampleAddress
}

type exampleAddressDTO struct {
	City string
}

type exampleUserDTO struct {
	ID      int64
	Name    string
	Tags    []string
	Address *exampleAddressDTO
}

func Example() {
	user := exampleUser{
		ID:       7,
		Name:     "Ann",
		Password: "secret",
		Tags:     []string{"admin", "ops"},
		Address:  &exampleAddress{City: "Berlin", Zip: "10115"},
	}

	dto := converter.Convert(user, &exampleUserDTO{})

	fmt.Println(dto.ID, dto.Name, dto.Tags, dto.Address.City)
	// Output: 7 Ann [admin ops] Berlin
}

func ExampleNew() {
	toDTO := converter.New[*exampleUser, *exampleUserDTO](converter.WithIgnoreFields("tags"))

	for _, user := range []*exampleUser{
		{ID: 1, Name: "Ann", Tags: []string{"admin"}},
		{ID: 2, Name: "Bob", Tags: []string{"ops"}},
	} {
		var dto exampleUserDTO
		if err := toDTO.ConvertE(user, &dto); err != nil {
			fmt.Println("error:", err)
			continue
		}
		fmt.Println(dto.ID, dto.Name, dto.Tags == nil)
	}
	// Output:
	// 1 Ann true
	// 2 Bob true
}

func ExampleWithSparseMerge() {
	type Address struct {
		City *string
		Zip  *string
	}
	type Profile struct {
		Name    *string
		Age     *int
		Address *Address
	}
	ptr := func(s string) *string { return &s }
	age := 30

	stored := Profile{
		Name:    ptr("Ann"),
		Age:     &age,
		Address: &Address{City: ptr("Berlin"), Zip: ptr("10115")},
	}

	// Only the set fields are applied; nested structs are merged.
	patch := Profile{Name: ptr("Anna"), Address: &Address{City: ptr("Paris")}}
	converter.Convert(&patch, &stored, converter.WithSparseMerge())
	fmt.Println(*stored.Name, *stored.Age, *stored.Address.City, *stored.Address.Zip)

	// A nested struct with no set fields clears the destination field.
	converter.Convert(&Profile{Address: &Address{}}, &stored, converter.WithSparseMerge())
	fmt.Println(*stored.Name, stored.Address == nil)
	// Output:
	// Anna 30 Paris 10115
	// Anna true
}

func ExampleWithFieldMappings() {
	type Account struct {
		Login string
		Email string
	}
	type AccountDTO struct {
		Username string
		Email    string
	}

	dto := converter.Convert(Account{Login: "ann", Email: "ann@example.com"}, &AccountDTO{},
		converter.WithFieldMappings(map[string]string{"Login": "Username"}))

	fmt.Println(dto.Username, dto.Email)
	// Output: ann ann@example.com
}

func ExampleWithCodecs() {
	type Event struct {
		Title string
		At    time.Time
	}
	type EventDTO struct {
		Title string
		At    string
	}

	// timeToString formats time.Time fields into string destinations and
	// delegates everything else to the next codec.
	timeToString := func(field string, src, dst reflect.Value, next convcodec.CodecHandler) {
		if t, ok := src.Interface().(time.Time); ok && dst.Kind() == reflect.String {
			dst.SetString(t.Format(time.DateOnly))
			return
		}
		next(field, src, dst)
	}

	event := Event{Title: "launch", At: time.Date(2026, time.March, 14, 9, 0, 0, 0, time.UTC)}
	dto := converter.Convert(event, &EventDTO{}, converter.WithCodecs(timeToString))

	fmt.Println(dto.Title, dto.At)
	// Output: launch 2026-03-14
}

func ExampleWithErrorCodecs() {
	type Signup struct {
		Name  string
		Email string
	}

	errInvalidEmail := errors.New("invalid email")
	validate := func(path string, src, dst reflect.Value, next convcodec.CodecHandlerE) error {
		if path == "email" && !strings.Contains(src.String(), "@") {
			return fmt.Errorf("%s: %w", path, errInvalidEmail)
		}
		return next(path, src, dst)
	}

	_, err := converter.ConvertE(Signup{Name: "Ann", Email: "nope"}, &Signup{}, converter.WithErrorCodecs(validate))

	fmt.Println(err)
	fmt.Println(errors.Is(err, errInvalidEmail))
	// Output:
	// email: invalid email
	// true
}

func ExampleConvertSeq() {
	users := []exampleUser{
		{ID: 1, Name: "Ann"},
		{ID: 2, Name: "Bob"},
	}

	for dto := range converter.ConvertSeq[exampleUserDTO](users) {
		fmt.Println(dto.ID, dto.Name)
	}
	// Output:
	// 1 Ann
	// 2 Bob
}

func ExampleConvertMapSeq() {
	usersByID := map[int64]exampleUser{
		2: {ID: 2, Name: "Bob"},
		1: {ID: 1, Name: "Ann"},
	}

	// Keys are converted too; iteration order is unspecified, so collect and sort.
	dtos := maps.Collect(converter.ConvertMapSeq[string, exampleUserDTO](usersByID))
	for _, key := range slices.Sorted(maps.Keys(dtos)) {
		fmt.Printf("%q %s\n", key, dtos[key].Name)
	}
	// Output:
	// "1" Ann
	// "2" Bob
}

func ExampleWithMaxDepth() {
	type Node struct {
		Value int
		Next  *Node
	}
	type NodeDTO struct {
		Value int
		Next  *NodeDTO
	}

	ring := &Node{Value: 1}
	ring.Next = &Node{Value: 2, Next: ring}

	// A distinct destination type makes the converter walk the cycle instead
	// of assigning the pointer as a whole.
	_, err := converter.ConvertE(ring, &NodeDTO{}, converter.WithMaxDepth(16))

	if depthErr, ok := errors.AsType[converter.DepthError](err); ok {
		fmt.Println("depth limit:", depthErr.Limit)
	}
	// Output: depth limit: 16
}
