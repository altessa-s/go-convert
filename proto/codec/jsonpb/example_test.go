// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package jsonpb_test

import (
	"errors"
	"fmt"
	"math"

	"github.com/altessa-s/go-convert/proto/codec/jsonpb"

	"google.golang.org/protobuf/types/known/structpb"

	converter "github.com/altessa-s/go-convert"
)

func ExampleNewE() {
	type Event struct {
		Name  string
		Attrs map[string]any
		Tags  []any
	}
	type EventMessage struct {
		Name  string
		Attrs *structpb.Struct
		Tags  *structpb.ListValue
	}

	codec := converter.WithErrorCodecs(jsonpb.NewE())

	event := Event{Name: "signup", Attrs: map[string]any{"plan": "pro", "seats": 3}, Tags: []any{"web", nil}}
	msg, err := converter.ConvertE(event, &EventMessage{}, codec)
	fmt.Println(err, msg.Attrs.GetFields()["seats"].GetNumberValue(), msg.Tags.GetValues()[1].GetNullValue())

	back, err := converter.ConvertE(msg, &Event{}, codec)
	fmt.Println(err, back.Attrs["plan"], back.Attrs["seats"], back.Tags)

	_, err = converter.ConvertE(Event{Attrs: map[string]any{"ratio": math.NaN()}}, &EventMessage{}, codec)
	var valueErr jsonpb.ValueError
	fmt.Println(errors.As(err, &valueErr), valueErr.Path)
	// Output:
	// <nil> 3 NULL_VALUE
	// <nil> pro 3 [web <nil>]
	// true attrs.ratio
}
