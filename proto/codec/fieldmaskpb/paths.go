// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package fieldmaskpb

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	reflectutils "github.com/altessa-s/go-convert/internal/reflect"
	fmpb "google.golang.org/protobuf/types/known/fieldmaskpb"
)

// wildcard is the AIP-134 mask path that selects every field.
const wildcard = "*"

// PathError reports a field mask path that does not select a field of the
// message, or, for [NewE] without [WithMessage], that is not well-formed.
type PathError struct {
	// Message is the full name of the message the mask applies to; empty
	// when only the path syntax was checked.
	Message protoreflect.FullName
	// Path is the offending mask path.
	Path string
	// Reason describes why the path is invalid.
	Reason string
}

// Error reports the message, the path and why it is invalid.
func (e PathError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("fieldmaskpb: path %q: %s", e.Path, e.Reason)
	}

	return fmt.Sprintf("fieldmaskpb: path %q of %s: %s", e.Path, e.Message, e.Reason)
}

// IgnorePaths returns the converter ignore paths that restrict a conversion
// whose source is msg to the fields listed in mask, for partial updates:
//
//	paths, err := fieldmaskpb.IgnorePaths(req.GetUpdateMask(), req.GetUser())
//	if err != nil {
//		return err // invalid mask: report InvalidArgument
//	}
//	_, err = converter.ConvertE(req.GetUser(), &user, converter.WithIgnorePaths(paths...))
//
// A field mask lists the fields to update, while the converter excludes
// fields, so IgnorePaths returns the complement: every field of msg that
// neither is listed in the mask nor lies on the way to a listed field. A
// listed message field selects its whole subtree; for a listed nested path
// such as "address.city", the siblings of city are ignored inside address.
// Masked fields are written even when they hold the zero value, which clears
// them in the destination, so do not combine the result with
// converter.WithIgnoreZeroValues or converter.WithSparseMerge. The converter
// skips nil struct pointers, so a masked message field left unset in msg does
// not clear the destination; clear such fields explicitly if the API requires
// it.
//
// Mask paths use protobuf field names (snake_case). The returned paths use
// the lower-cased Go field names of the generated message struct, which is
// how the converter names source fields: "display_name" becomes
// "displayname". The names are read from the struct tags, with the
// protoc-gen-go naming rules as the fallback for dynamic messages. The
// conversion source must therefore be msg itself or a struct whose fields are
// named like it, and its paths must not be renamed with
// converter.WithFieldMappings.
//
// A oneof is a single Go field: it is ignored unless one of its members is
// listed, and then converted whole. A path may end at a repeated or map
// field but must not traverse one, nor a oneof member. Invalid paths fail
// with [PathError]. A nil or empty mask, and the wildcard mask "*", select
// every field and yield no ignore paths. msg only provides the descriptor and
// may be a typed nil pointer.
func IgnorePaths(mask *fmpb.FieldMask, msg proto.Message) ([]string, error) {
	if msg == nil {
		return nil, errors.New("fieldmaskpb: IgnorePaths called with a nil message")
	}

	desc := msg.ProtoReflect().Descriptor()

	paths := mask.GetPaths()
	if len(paths) == 0 || (len(paths) == 1 && paths[0] == wildcard) {
		return nil, nil
	}

	root := &maskNode{}
	for _, path := range paths {
		if err := root.insert(desc, path); err != nil {
			return nil, err
		}
	}

	var ignored []string
	collectIgnored(desc, reflect.TypeOf(msg), root, "", &ignored)

	return ignored, nil
}

// maskNode is one message level of the parsed mask. A leaf selects the
// whole field.
type maskNode struct {
	leaf     bool
	children map[protoreflect.Name]*maskNode
}

// insert validates path against desc and adds it below n. A path below a
// selected field is already covered; a selected field covers the paths
// below it.
func (n *maskNode) insert(desc protoreflect.MessageDescriptor, path string) error {
	fields, err := resolve(desc, path)
	if err != nil {
		return err
	}

	for _, field := range fields {
		if n.leaf {
			return nil
		}

		n = n.child(field.Name())
	}

	n.leaf = true
	n.children = nil

	return nil
}

// resolve returns the fields named by the segments of path. Every segment
// but the last must name a singular message field outside a oneof.
func resolve(desc protoreflect.MessageDescriptor, path string) ([]protoreflect.FieldDescriptor, error) {
	invalid := func(reason string, args ...any) error {
		return PathError{Message: desc.FullName(), Path: path, Reason: fmt.Sprintf(reason, args...)}
	}
	if path == wildcard {
		return nil, invalid("the wildcard must be the only path")
	}

	segments := strings.Split(path, ".")
	fields := make([]protoreflect.FieldDescriptor, 0, len(segments))
	current := desc

	for i, segment := range segments {
		if segment == "" {
			return nil, invalid("empty field name")
		}

		if current == nil {
			return nil, invalid("%q is not a message field and must be the last segment", segments[i-1])
		}

		field := current.Fields().ByName(protoreflect.Name(segment))
		if field == nil {
			return nil, invalid("%s has no field %q", current.FullName(), segment)
		}

		fields = append(fields, field)

		if i == len(segments)-1 {
			break
		}

		switch {
		case field.IsList() || field.IsMap():
			return nil, invalid("%q is a repeated field and must be the last segment", segment)
		case isOneofMember(field):
			return nil, invalid("%q is a oneof member and must be the last segment", segment)
		default:
			current = field.Message()
		}
	}

	return fields, nil
}

// child returns the node of the named field, creating it when absent.
func (n *maskNode) child(name protoreflect.Name) *maskNode {
	if n.children == nil {
		n.children = make(map[protoreflect.Name]*maskNode)
	}

	c, ok := n.children[name]
	if !ok {
		c = &maskNode{}
		n.children[name] = c
	}

	return c
}

// collectIgnored appends, in descriptor order, the converter paths of the
// fields of desc that node does not select. goType is the generated Go type
// of the message, or nil when unknown.
func collectIgnored(desc protoreflect.MessageDescriptor, goType reflect.Type, node *maskNode, prefix string, ignored *[]string) {
	names := goNamesOf(goType)

	fields := desc.Fields()
	for i := range fields.Len() {
		field := fields.Get(i)
		if isOneofMember(field) {
			continue
		}

		goField := names.field(field)

		child := node.children[field.Name()]
		switch {
		case child == nil:
			*ignored = append(*ignored, prefix+strings.ToLower(goField.name))
		case !child.leaf:
			collectIgnored(field.Message(), goField.typ, child, prefix+strings.ToLower(goField.name)+".", ignored)
		default:
		}
	}

	oneofs := desc.Oneofs()
	for i := range oneofs.Len() {
		oneof := oneofs.Get(i)
		if oneof.IsSynthetic() || selectsOneof(node, oneof) {
			continue
		}

		*ignored = append(*ignored, prefix+strings.ToLower(names.oneof(oneof)))
	}
}

// selectsOneof reports whether node selects a member of oneof.
func selectsOneof(node *maskNode, oneof protoreflect.OneofDescriptor) bool {
	members := oneof.Fields()
	for i := range members.Len() {
		if _, ok := node.children[members.Get(i).Name()]; ok {
			return true
		}
	}

	return false
}

// isOneofMember reports whether field belongs to a real (non-synthetic) oneof.
func isOneofMember(field protoreflect.FieldDescriptor) bool {
	oneof := field.ContainingOneof()
	return oneof != nil && !oneof.IsSynthetic()
}

// goField is the Go struct field generated for a protobuf field.
type goField struct {
	name string
	typ  reflect.Type // message struct type, nil when unknown
}

// goNames maps protobuf field and oneof names to generated Go fields.
type goNames struct {
	fields map[protoreflect.Name]goField
	oneofs map[protoreflect.Name]string
}

var goNamesCache sync.Map // reflect.Type -> *goNames

// goNamesOf reads the protobuf struct tags of a generated message type. The
// result is empty for types without tags, such as dynamic messages.
func goNamesOf(t reflect.Type) *goNames {
	if t == nil {
		return &goNames{}
	}

	t = reflectutils.IndirectType(t)
	if cached, ok := goNamesCache.Load(t); ok {
		names, _ := cached.(*goNames)
		return names
	}

	names := &goNames{fields: map[protoreflect.Name]goField{}, oneofs: map[protoreflect.Name]string{}}

	if t.Kind() == reflect.Struct {
		for f := range t.Fields() {
			if oneof, ok := f.Tag.Lookup("protobuf_oneof"); ok {
				names.oneofs[protoreflect.Name(oneof)] = f.Name
				continue
			}

			if name, ok := tagName(f.Tag.Get("protobuf")); ok {
				typ := reflectutils.IndirectType(f.Type)
				if typ.Kind() != reflect.Struct {
					typ = nil
				}

				names.fields[name] = goField{name: f.Name, typ: typ}
			}
		}
	}

	goNamesCache.Store(t, names)

	return names
}

// field returns the Go field generated for a protobuf field.
func (n *goNames) field(field protoreflect.FieldDescriptor) goField {
	if f, ok := n.fields[field.Name()]; ok {
		return f
	}

	return goField{name: goCamelCase(string(field.Name()))}
}

// oneof returns the Go interface field generated for a oneof.
func (n *goNames) oneof(oneof protoreflect.OneofDescriptor) string {
	if name, ok := n.oneofs[oneof.Name()]; ok {
		return name
	}

	return goCamelCase(string(oneof.Name()))
}

// tagName extracts the name= item of a protobuf struct tag.
func tagName(tag string) (protoreflect.Name, bool) {
	for item := range strings.SplitSeq(tag, ",") {
		if name, ok := strings.CutPrefix(item, "name="); ok {
			return protoreflect.Name(name), true
		}
	}

	return "", false
}

// goCamelCase converts a protobuf name to its generated Go name, following
// protoc-gen-go: an underscore followed by a lower-case letter is dropped and
// the letter upper-cased, other underscores are kept, and a leading
// underscore becomes X.
func goCamelCase(s string) string {
	b := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '_' && i == 0:
			b = append(b, 'X')
		case c == '_' && i+1 < len(s) && isLower(s[i+1]):
			// Dropped; the next letter is upper-cased below.
		case isDigit(c):
			b = append(b, c)
		default:
			if isLower(c) {
				c -= 'a' - 'A'
			}

			b = append(b, c)
			for ; i+1 < len(s) && isLower(s[i+1]); i++ {
				b = append(b, s[i+1])
			}
		}
	}

	return string(b)
}

// isLower reports whether c is an ASCII lower-case letter.
func isLower(c byte) bool { return c >= 'a' && c <= 'z' }

// isDigit reports whether c is an ASCII digit.
func isDigit(c byte) bool { return c >= '0' && c <= '9' }
