// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package enumpb_test

import (
	"fmt"

	"google.golang.org/protobuf/types/known/typepb"

	"github.com/altessa-s/go-convert"
	"github.com/altessa-s/go-convert/proto/codec/enumpb"
)

func Example() {
	type Column struct {
		Name        string
		Kind        string
		Cardinality string
	}

	opts := []convert.Option{convert.WithErrorCodecs(enumpb.MustNew(
		enumpb.WithTrimEnumPrefix(), // CARDINALITY_REPEATED -> REPEATED
		enumpb.WithCase(enumpb.CaseLower),
		enumpb.WithUnspecifiedAsEmpty(), // TYPE_UNKNOWN <-> ""
	))}

	field := &typepb.Field{Name: "tags", Cardinality: typepb.Field_CARDINALITY_REPEATED}
	column, err := convert.ConvertE(field, &Column{}, opts...)
	fmt.Printf("%+v %v\n", *column, err)

	_, err = convert.ConvertE(&Column{Cardinality: "sometimes"}, &typepb.Field{}, opts...)
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

	codec := enumpb.MustNewMapping(map[typepb.Field_Cardinality]Cardinality{
		typepb.Field_CARDINALITY_OPTIONAL: Optional,
		typepb.Field_CARDINALITY_REPEATED: Repeated,
	})

	field, err := convert.ConvertE(&Column{Name: "tags", Cardinality: Repeated}, &typepb.Field{}, convert.WithErrorCodecs(codec))
	fmt.Println(field.GetCardinality(), err)
	// Output:
	// CARDINALITY_REPEATED <nil>
}
