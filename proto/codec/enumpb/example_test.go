// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package enumpb_test

import (
	"fmt"

	"github.com/altessa-s/go-convert/proto/codec/enumpb"

	"google.golang.org/protobuf/types/known/typepb"

	converter "github.com/altessa-s/go-convert"
)

func Example() {
	type Column struct {
		Name        string
		Kind        string
		Cardinality string
	}

	opts := []converter.Option{converter.WithErrorCodecs(enumpb.New(
		enumpb.WithTrimEnumPrefix(), // CARDINALITY_REPEATED -> REPEATED
		enumpb.WithCase(enumpb.CaseLower),
		enumpb.WithUnspecifiedAsEmpty(), // TYPE_UNKNOWN <-> ""
	))}

	field := &typepb.Field{Name: "tags", Cardinality: typepb.Field_CARDINALITY_REPEATED}
	column, err := converter.ConvertE(field, &Column{}, opts...)
	fmt.Printf("%+v %v\n", *column, err)

	_, err = converter.ConvertE(&Column{Cardinality: "sometimes"}, &typepb.Field{}, opts...)
	fmt.Println(err)
	// Output:
	// {Name:tags Kind: Cardinality:repeated} <nil>
	// enumpb: field "cardinality": google.protobuf.Field.Cardinality has no value for "sometimes"
}

func ExampleNewMapping() {
	type Cardinality int
	const (
		Optional Cardinality = iota + 1
		Repeated
	)
	type Column struct {
		Name        string
		Cardinality Cardinality
	}

	codec := enumpb.NewMapping(map[typepb.Field_Cardinality]Cardinality{
		typepb.Field_CARDINALITY_OPTIONAL: Optional,
		typepb.Field_CARDINALITY_REPEATED: Repeated,
	})

	field, err := converter.ConvertE(&Column{Name: "tags", Cardinality: Repeated}, &typepb.Field{}, converter.WithErrorCodecs(codec))
	fmt.Println(field.GetCardinality(), err)
	// Output:
	// CARDINALITY_REPEATED <nil>
}
