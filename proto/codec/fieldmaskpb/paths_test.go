// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package fieldmaskpb_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-convert/proto/codec/fieldmaskpb"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/apipb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/typepb"

	converter "github.com/altessa-s/go-convert"
	fmpb "google.golang.org/protobuf/types/known/fieldmaskpb"
)

func mask(paths ...string) *fmpb.FieldMask { return &fmpb.FieldMask{Paths: paths} }

func TestIgnorePaths_GeneratedMessages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		mask *fmpb.FieldMask
		msg  proto.Message
		want []string
	}{
		{name: "nil mask selects everything", msg: &typepb.Field{}},
		{name: "empty mask selects everything", mask: mask(), msg: &typepb.Field{}},
		{name: "wildcard selects everything", mask: mask("*"), msg: &typepb.Field{}},
		{
			name: "snake_case names map to lower-cased Go names",
			mask: mask("name", "json_name"),
			msg:  &typepb.Field{},
			want: []string{"kind", "cardinality", "number", "typeurl", "oneofindex", "packed", "options", "defaultvalue"},
		},
		{
			name: "repeated field as the last segment",
			mask: mask("type_url", "kind", "cardinality", "number", "name", "oneof_index", "packed", "json_name", "default_value"),
			msg:  &typepb.Field{},
			want: []string{"options"},
		},
		{
			name: "nested path ignores the siblings inside the parent",
			mask: mask("value.type_url"),
			msg:  &typepb.Option{},
			want: []string{"name", "value.value"},
		},
		{
			name: "a parent path covers its nested paths in any order",
			mask: mask("value.type_url", "value", "value.value"),
			msg:  &typepb.Option{},
			want: []string{"name"},
		},
		{
			name: "duplicates are harmless",
			mask: mask("name", "name"),
			msg:  &typepb.Option{},
			want: []string{"value"},
		},
		{
			name: "deep nested path",
			mask: mask("source_context.file_name", "syntax"),
			msg:  &apipb.Api{},
			want: []string{"name", "methods", "options", "version", "mixins", "edition"},
		},
		{
			name: "oneof member keeps the whole oneof",
			mask: mask("string_value"),
			msg:  &structpb.Value{},
		},
		{
			name: "typed nil message",
			mask: mask("type_url"),
			msg:  (*anypb.Any)(nil),
			want: []string{"value"},
		},
		{
			name: "proto2 names with digits",
			mask: mask("java_package"),
			msg:  &descriptorpb.FileOptions{},
			want: []string{
				"javaouterclassname", "javamultiplefiles", "javagenerateequalsandhash", "javastringcheckutf8", "optimizefor",
				"gopackage", "ccgenericservices", "javagenericservices", "pygenericservices", "deprecated", "ccenablearenas",
				"objcclassprefix", "csharpnamespace", "swiftprefix", "phpclassprefix", "phpnamespace", "phpmetadatanamespace",
				"rubypackage", "features", "uninterpretedoption",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := fieldmaskpb.IgnorePaths(tt.mask, tt.msg)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

// profileDescriptor builds, without .proto generation, a message with a
// oneof, a proto3 optional field, a map, nested and repeated messages and
// names that exercise the protoc-gen-go naming rules.
func profileDescriptor(t *testing.T) protoreflect.MessageDescriptor {
	t.Helper()

	scalar := func(name string, number int32) *descriptorpb.FieldDescriptorProto {
		return &descriptorpb.FieldDescriptorProto{
			Name:   new(name),
			Number: new(number),
			Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
			Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
		}
	}
	inOneof := func(field *descriptorpb.FieldDescriptorProto, index int32) *descriptorpb.FieldDescriptorProto {
		field.OneofIndex = new(index)
		return field
	}
	message := func(name string, number int32, typeName string, label descriptorpb.FieldDescriptorProto_Label) *descriptorpb.FieldDescriptorProto {
		return &descriptorpb.FieldDescriptorProto{
			Name:     new(name),
			Number:   new(number),
			Label:    label.Enum(),
			Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
			TypeName: new(typeName),
		}
	}
	optional := inOneof(scalar("nick_name", 4), 1)
	optional.Proto3Optional = new(true)

	file := &descriptorpb.FileDescriptorProto{
		Name:    new("fieldmaskpb_test.proto"),
		Package: new("fieldmaskpb.test"),
		Syntax:  new("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: new("Address"), Field: []*descriptorpb.FieldDescriptorProto{scalar("city", 1), scalar("zip_code", 2)}},
			{
				Name: new("Profile"),
				Field: []*descriptorpb.FieldDescriptorProto{
					scalar("display_name", 1),
					inOneof(scalar("email", 2), 0),
					inOneof(scalar("phone", 3), 0),
					optional,
					message("labels", 5, ".fieldmaskpb.test.Profile.LabelsEntry", descriptorpb.FieldDescriptorProto_LABEL_REPEATED),
					message("address", 6, ".fieldmaskpb.test.Address", descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
					message("history", 7, ".fieldmaskpb.test.Address", descriptorpb.FieldDescriptorProto_LABEL_REPEATED),
					scalar("line_2", 8),
					scalar("x509_cert", 9),
				},
				OneofDecl: []*descriptorpb.OneofDescriptorProto{{Name: new("contact")}, {Name: new("_nick_name")}},
				NestedType: []*descriptorpb.DescriptorProto{{
					Name:    new("LabelsEntry"),
					Field:   []*descriptorpb.FieldDescriptorProto{scalar("key", 1), scalar("value", 2)},
					Options: &descriptorpb.MessageOptions{MapEntry: new(true)},
				}},
			},
		},
	}
	fd, err := protodesc.NewFile(file, nil)
	require.NoError(t, err)
	return fd.Messages().ByName("Profile")
}

func TestIgnorePaths_DynamicMessage(t *testing.T) {
	t.Parallel()

	profile := dynamicpb.NewMessage(profileDescriptor(t))

	tests := []struct {
		name string
		mask *fmpb.FieldMask
		want []string
	}{
		{
			name: "oneof without a selected member is ignored",
			mask: mask("address.city"),
			want: []string{"displayname", "nickname", "labels", "address.zipcode", "history", "line_2", "x509cert", "contact"},
		},
		{
			name: "oneof member, proto3 optional and map fields",
			mask: mask("phone", "nick_name", "labels", "address", "line_2"),
			want: []string{"displayname", "history", "x509cert"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := fieldmaskpb.IgnorePaths(tt.mask, profile)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestIgnorePaths_DynamicNamesMatchGenerated(t *testing.T) {
	t.Parallel()

	for _, msg := range []proto.Message{
		&typepb.Field{}, &typepb.Type{}, &apipb.Api{}, &descriptorpb.FileOptions{}, &descriptorpb.FieldDescriptorProto{},
		&descriptorpb.FileDescriptorProto{}, &descriptorpb.DescriptorProto{}, &structpb.Value{},
	} {
		desc := msg.ProtoReflect().Descriptor()
		t.Run(string(desc.FullName()), func(t *testing.T) {
			t.Parallel()

			m := mask(string(desc.Fields().Get(0).Name()))
			generated, err := fieldmaskpb.IgnorePaths(m, msg)
			require.NoError(t, err)
			dynamic, err := fieldmaskpb.IgnorePaths(m, dynamicpb.NewMessage(desc))
			require.NoError(t, err)
			require.Equal(t, generated, dynamic)
		})
	}
}

func TestIgnorePaths_InvalidPaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		mask *fmpb.FieldMask
		msg  proto.Message
		want fieldmaskpb.PathError
	}{
		{name: "unknown field", mask: mask("name", "title"), msg: &typepb.Option{},
			want: fieldmaskpb.PathError{Message: "google.protobuf.Option", Path: "title", Reason: `google.protobuf.Option has no field "title"`}},
		{name: "JSON name", mask: mask("typeUrl"), msg: &typepb.Field{},
			want: fieldmaskpb.PathError{Message: "google.protobuf.Field", Path: "typeUrl", Reason: `google.protobuf.Field has no field "typeUrl"`}},
		{name: "unknown nested field", mask: mask("value.type"), msg: &typepb.Option{},
			want: fieldmaskpb.PathError{Message: "google.protobuf.Option", Path: "value.type", Reason: `google.protobuf.Any has no field "type"`}},
		{name: "empty path", mask: mask(""), msg: &typepb.Option{},
			want: fieldmaskpb.PathError{Message: "google.protobuf.Option", Path: "", Reason: "empty field name"}},
		{name: "trailing dot", mask: mask("value."), msg: &typepb.Option{},
			want: fieldmaskpb.PathError{Message: "google.protobuf.Option", Path: "value.", Reason: "empty field name"}},
		{name: "through a scalar", mask: mask("name.length"), msg: &typepb.Option{},
			want: fieldmaskpb.PathError{Message: "google.protobuf.Option", Path: "name.length", Reason: `"name" is not a message field and must be the last segment`}},
		{name: "through a repeated field", mask: mask("fields.name"), msg: &typepb.Type{},
			want: fieldmaskpb.PathError{Message: "google.protobuf.Type", Path: "fields.name", Reason: `"fields" is a repeated field and must be the last segment`}},
		{name: "through a map", mask: mask("fields.key"), msg: &structpb.Struct{},
			want: fieldmaskpb.PathError{Message: "google.protobuf.Struct", Path: "fields.key", Reason: `"fields" is a repeated field and must be the last segment`}},
		{name: "through a oneof member", mask: mask("struct_value.fields"), msg: &structpb.Value{},
			want: fieldmaskpb.PathError{Message: "google.protobuf.Value", Path: "struct_value.fields", Reason: `"struct_value" is a oneof member and must be the last segment`}},
		{name: "wildcard with other paths", mask: mask("name", "*"), msg: &typepb.Option{},
			want: fieldmaskpb.PathError{Message: "google.protobuf.Option", Path: "*", Reason: "the wildcard must be the only path"}},
		{name: "invalid path below a selected field", mask: mask("value", "value.type"), msg: &typepb.Option{},
			want: fieldmaskpb.PathError{Message: "google.protobuf.Option", Path: "value.type", Reason: `google.protobuf.Any has no field "type"`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := fieldmaskpb.IgnorePaths(tt.mask, tt.msg)
			require.Nil(t, got)
			require.Equal(t, error(tt.want), err)
		})
	}
}

func TestIgnorePaths_NilMessage(t *testing.T) {
	t.Parallel()

	got, err := fieldmaskpb.IgnorePaths(mask("name"), nil)
	require.Nil(t, got)
	require.EqualError(t, err, "fieldmaskpb: IgnorePaths called with a nil message")
}

func TestPathError_Error(t *testing.T) {
	t.Parallel()

	err := fieldmaskpb.PathError{Message: "google.protobuf.Option", Path: "title", Reason: "no such field"}
	require.EqualError(t, err, `fieldmaskpb: path "title" of google.protobuf.Option: no such field`)
}

type anyDTO struct {
	TypeUrl string
	Value   []byte
}

type optionDTO struct {
	Name  string
	Value *anyDTO
}

func TestConverter_PartialUpdate(t *testing.T) {
	t.Parallel()

	stored := func() optionDTO {
		return optionDTO{Name: "old", Value: &anyDTO{TypeUrl: "old-url", Value: []byte("old")}}
	}
	update := &typepb.Option{Name: "new", Value: &anypb.Any{TypeUrl: "new-url", Value: []byte("new")}}

	tests := []struct {
		name   string
		mask   *fmpb.FieldMask
		update *typepb.Option
		want   optionDTO
	}{
		{name: "top-level field", mask: mask("name"), update: update,
			want: optionDTO{Name: "new", Value: &anyDTO{TypeUrl: "old-url", Value: []byte("old")}}},
		{name: "nested field", mask: mask("value.type_url"), update: update,
			want: optionDTO{Name: "old", Value: &anyDTO{TypeUrl: "new-url", Value: []byte("old")}}},
		{name: "whole message field", mask: mask("value"), update: update,
			want: optionDTO{Name: "old", Value: &anyDTO{TypeUrl: "new-url", Value: []byte("new")}}},
		{name: "masked zero value clears the field", mask: mask("name", "value.value"), update: &typepb.Option{Value: &anypb.Any{TypeUrl: "ignored"}},
			want: optionDTO{Name: "", Value: &anyDTO{TypeUrl: "old-url"}}},
		// The converter skips nil struct pointers, so an unset message field
		// cannot clear the destination; see the IgnorePaths documentation.
		{name: "unset masked message leaves the field", mask: mask("value"), update: &typepb.Option{Name: "ignored"},
			want: optionDTO{Name: "old", Value: &anyDTO{TypeUrl: "old-url", Value: []byte("old")}}},
		{name: "unset parent of a masked nested field leaves the field", mask: mask("value.type_url"), update: &typepb.Option{Name: "ignored"},
			want: optionDTO{Name: "old", Value: &anyDTO{TypeUrl: "old-url", Value: []byte("old")}}},
		{name: "empty mask updates everything", mask: mask(), update: update,
			want: optionDTO{Name: "new", Value: &anyDTO{TypeUrl: "new-url", Value: []byte("new")}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			paths, err := fieldmaskpb.IgnorePaths(tt.mask, tt.update)
			require.NoError(t, err)

			dst := stored()
			_, err = converter.ConvertE(tt.update, &dst, converter.WithIgnorePaths(paths...))
			require.NoError(t, err)
			require.Equal(t, tt.want, dst)
		})
	}
}
