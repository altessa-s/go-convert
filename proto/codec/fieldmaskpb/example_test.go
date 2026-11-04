// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package fieldmaskpb_test

import (
	"fmt"

	"github.com/altessa-s/go-convert/proto/codec/fieldmaskpb"

	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/typepb"

	converter "github.com/altessa-s/go-convert"
	fmpb "google.golang.org/protobuf/types/known/fieldmaskpb"
)

func ExampleIgnorePaths() {
	type Payload struct {
		TypeUrl string
		Value   []byte
	}
	type Option struct {
		Name  string
		Value *Payload
	}

	stored := Option{Name: "retention", Value: &Payload{TypeUrl: "type.googleapis.com/v1", Value: []byte("30d")}}

	// An update request: the resource and the fields it updates.
	update := &typepb.Option{Name: "ignored", Value: &anypb.Any{TypeUrl: "type.googleapis.com/v2", Value: []byte("ignored")}}
	updateMask := &fmpb.FieldMask{Paths: []string{"value.type_url"}}

	paths, err := fieldmaskpb.IgnorePaths(updateMask, update)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(paths)

	if _, err := converter.ConvertE(update, &stored, converter.WithIgnorePaths(paths...)); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(stored.Name, stored.Value.TypeUrl, string(stored.Value.Value))

	_, err = fieldmaskpb.IgnorePaths(&fmpb.FieldMask{Paths: []string{"value.typeUrl"}}, update)
	fmt.Println(err)
	// Output:
	// [name value.value]
	// retention type.googleapis.com/v2 30d
	// fieldmaskpb: path "value.typeUrl" of google.protobuf.Option: google.protobuf.Any has no field "typeUrl"
}

func ExampleNew() {
	type Request struct {
		UpdateMask []string
	}
	type RequestPB struct {
		UpdateMask *fmpb.FieldMask
	}

	codec := fieldmaskpb.New(fieldmaskpb.WithNormalize())
	pb := converter.Convert(&Request{UpdateMask: []string{"title", "author", "author.name"}}, &RequestPB{}, converter.WithCodecs(codec))
	fmt.Println(pb.UpdateMask.GetPaths())
	// Output:
	// [author title]
}
