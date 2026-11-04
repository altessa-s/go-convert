// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package jsonpb

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"math"
	"reflect"
	"strconv"
	"unicode/utf8"

	"github.com/altessa-s/go-convert/reflectx"

	"google.golang.org/protobuf/types/known/structpb"

	convcodec "github.com/altessa-s/go-convert/codec"
)

var (
	valueType     = reflect.TypeFor[structpb.Value]()
	listValueType = reflect.TypeFor[structpb.ListValue]()
	anyType       = reflect.TypeFor[any]()
	anySliceType  = reflect.TypeFor[[]any]()
	anyMapType    = reflect.TypeFor[map[string]any]()
)

const (
	reasonNonFinite   = "non-finite numbers have no JSON representation"
	reasonUnsupported = "unsupported type"
	reasonInvalidUTF8 = "invalid UTF-8"
	reasonCyclic      = "cyclic value"

	// DefaultMaxDepth is the nesting limit of lists and maps used by [NewE]
	// unless [WithMaxDepth] sets another one.
	DefaultMaxDepth = 10000

	// cycleCheckDepth is the nesting depth beyond which the walker starts
	// tracking the containers it is inside of, so that shallow values do not
	// pay for cycle detection.
	cycleCheckDepth = 64
)

// errNotObject is wrapped by the [Error] returned when a json.RawMessage
// converted into a structpb.Struct holds JSON other than an object or null.
var errNotObject = errors.New("JSON value is not an object")

// jsonOptions are the encoding/json/v2 options used to render JSON text:
// object members are written in sorted order, as encoding/json v1 did.
var jsonOptions = json.Deterministic(true)

// converter converts a non-nil source value into the value stored in the
// destination: a message pointer or a value of the destination's type.
type converter func(w *walker, path string, src reflect.Value) (reflect.Value, error)

// visitKey identifies a list or map being converted: the address of its
// data and, for slices, its length.
type visitKey struct {
	ptr uintptr
	n   int
}

// walker carries the state of one conversion: the current nesting depth and,
// past cycleCheckDepth, the containers being converted.
type walker struct {
	visiting map[visitKey]struct{}
	maxDepth int
	depth    int
}

// enter records that w descends into container c, a map, slice or message
// pointer, failing when the nesting limit is exceeded or c is already being
// converted. Containers are tracked only past cycleCheckDepth.
func enter[T any](w *walker, c T) error {
	w.depth++
	if w.depth > w.maxDepth {
		return ValueError{Reason: "nesting exceeds " + strconv.Itoa(w.maxDepth) + " levels"}
	}

	if w.depth <= cycleCheckDepth {
		return nil
	}

	key := keyOf(reflect.ValueOf(c))
	if _, ok := w.visiting[key]; ok {
		return ValueError{Reason: reasonCyclic}
	}

	if w.visiting == nil {
		w.visiting = make(map[visitKey]struct{})
	}

	w.visiting[key] = struct{}{}

	return nil
}

// leave undoes the matching [enter] for container c.
func leave[T any](w *walker, c T) {
	if w.depth > cycleCheckDepth {
		delete(w.visiting, keyOf(reflect.ValueOf(c)))
	}

	w.depth--
}

// keyOf returns the visit key of a pointer, map or slice value.
func keyOf(v reflect.Value) visitKey {
	if v.Kind() == reflect.Slice {
		return visitKey{ptr: v.Pointer(), n: v.Len()}
	}

	return visitKey{ptr: v.Pointer()}
}

// NewE creates an error codec for bidirectional conversion between the
// structpb messages and their plain Go counterparts. Register it with
// convert.WithErrorCodecs.
//
// Supported conversions, all in both directions and for pointer and
// non-pointer fields:
//   - json.RawMessage <-> structpb.Struct (as [New]; invalid JSON is an error)
//   - structpb.Value <-> any: nil, bool, float64, string, []any and
//     map[string]any. On the way in, signed and unsigned integers, float32
//     and named types of these kinds are accepted as numbers, strings and
//     booleans; integers beyond 2^53 lose precision as in JSON.
//   - structpb.ListValue <-> []any
//   - structpb.Struct <-> map[string]any
//
// A structpb.Struct or structpb.ListValue source also converts into an any
// destination, holding a map[string]any or []any. A source field of type any
// converts into a structpb.Value, structpb.ListValue or structpb.Struct
// according to its dynamic value.
//
// A nil source (nil pointer, nil interface, nil slice or map, or a
// json.RawMessage that is empty or holds JSON null) sets the destination to
// its zero value; with [WithIgnoreNil] the destination is left untouched. A
// google.protobuf.Value holding NullValue or no kind converts to nil, and nil
// inside []any or map[string]any converts to NullValue.
//
// JSON text is parsed and rendered with encoding/json/v2, which rejects
// duplicate object names and invalid UTF-8. A json.RawMessage holding JSON
// null converts to a nil *structpb.Struct, the same as an empty one, and
// {} converts to an empty Struct.
//
// Conversion fails with a [ValueError] for non-finite numbers (in either
// direction), strings or keys that are not valid UTF-8, values of any other
// Go type, lists and maps nested deeper than [WithMaxDepth] levels
// ([DefaultMaxDepth] by default) and lists or maps that contain themselves;
// with an [Error] when a json.RawMessage is not valid JSON or holds a JSON
// value other than an object or null. Other field pairs are passed to next
// unchanged.
//
// Example:
//
//	codec := jsonpb.NewE()
//	convert.ConvertE(src, &dst, convert.WithErrorCodecs(codec))
func NewE(opts ...Option) convcodec.CodecE {
	o := newOptions(opts...)
	if o.maxDepth <= 0 {
		o.maxDepth = DefaultMaxDepth
	}

	return func(fieldPath string, src, dst reflect.Value, next convcodec.CodecHandlerE) error {
		convert := selectConverter(src.Type(), dst.Type())
		if convert == nil {
			return next(fieldPath, src, dst)
		}

		value := sourceValue(src)
		if isNilSource(value) {
			if !o.ignoreNil && dst.CanSet() {
				dst.SetZero()
			}

			return nil
		}

		w := walker{maxDepth: o.maxDepth}

		out, err := convert(&w, fieldPath, value)
		if err != nil {
			return err
		}

		setDst(dst, out)

		return nil
	}
}

// selectConverter returns the converter for the source and destination
// types, or nil when the pair is not handled by [NewE].
func selectConverter(srcType, dstType reflect.Type) converter {
	srcType = reflectx.IndirectType(srcType)
	dstType = reflectx.IndirectType(dstType)

	switch {
	case srcType == rawMessageType && dstType == structpbType:
		return rawToStruct
	case srcType == structpbType && dstType == rawMessageType:
		return structToRaw
	case srcType == valueType && dstType == anyType:
		return valueToAny
	case srcType == listValueType && (dstType == anySliceType || dstType == anyType):
		return listToSlice
	case srcType == structpbType && (dstType == anyMapType || dstType == anyType):
		return structToMap
	case dstType == valueType && acceptsValue(srcType):
		return anyToValue
	case dstType == listValueType && (srcType == anySliceType || srcType == anyType):
		return sliceToList
	case dstType == structpbType && (srcType == anyMapType || srcType == anyType):
		return mapToStruct
	default:
		return nil
	}
}

// acceptsValue reports whether a source of static type t converts into a
// structpb.Value.
func acceptsValue(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Bool, reflect.String,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	default:
		return t == anyType || t == anySliceType || t == anyMapType
	}
}

// sourceValue dereferences pointers and unwraps an interface, returning an
// invalid value when the source is nil.
func sourceValue(src reflect.Value) reflect.Value {
	value := reflectx.IndirectValue(src)
	if value.IsValid() && value.Kind() == reflect.Interface {
		if value.IsNil() {
			return reflect.Value{}
		}

		value = value.Elem()
	}

	return value
}

// isNilSource reports whether value holds no data to convert.
func isNilSource(value reflect.Value) bool {
	if !value.IsValid() {
		return true
	}

	switch value.Kind() {
	case reflect.Slice:
		return value.IsNil() || (value.Type() == rawMessageType && isJSONNull(value.Bytes()))
	case reflect.Map:
		return value.IsNil()
	default:
		return false
	}
}

// isJSONNull reports whether text is empty or the JSON literal null, ignoring
// surrounding JSON whitespace.
func isJSONNull(text []byte) bool {
	text = bytes.Trim(text, " \t\r\n")

	return len(text) == 0 || string(text) == "null"
}

// setDst stores out into dst, allocating pointer layers. out is either a
// message pointer or a value assignable to the innermost destination.
func setDst(dst, out reflect.Value) {
	for dst.Kind() == reflect.Pointer {
		if dst.Type() == out.Type() {
			dst.Set(out)
			return
		}

		if dst.IsNil() {
			dst.Set(reflect.New(dst.Type().Elem()))
		}

		dst = dst.Elem()
	}

	if out.Kind() == reflect.Pointer && out.Type().Elem() == dst.Type() {
		dst.Set(out.Elem())
		return
	}

	dst.Set(out)
}

// messagePtr returns a pointer to the message held by v, copying it when v
// is not addressable.
func messagePtr[T any](v reflect.Value) *T {
	if !v.CanAddr() {
		tmp := reflect.New(v.Type())
		tmp.Elem().Set(v)
		v = tmp.Elem()
	}

	p, _ := reflect.TypeAssert[*T](v.Addr())

	return p
}

// rawToStruct parses JSON object text into a Struct.
func rawToStruct(w *walker, path string, src reflect.Value) (reflect.Value, error) {
	var x any
	if err := json.Unmarshal(src.Bytes(), &x); err != nil {
		return reflect.Value{}, Error{Path: path, Err: err}
	}

	m, ok := x.(map[string]any)
	if !ok {
		return reflect.Value{}, Error{Path: path, Err: errNotObject}
	}

	pb, err := w.toStruct(m)
	if err != nil {
		return reflect.Value{}, at(err, path)
	}

	return reflect.ValueOf(pb), nil
}

// structToRaw renders a Struct as JSON text.
func structToRaw(w *walker, path string, src reflect.Value) (reflect.Value, error) {
	m, err := w.fromStruct(messagePtr[structpb.Struct](src))
	if err != nil {
		return reflect.Value{}, at(err, path)
	}

	data, err := json.Marshal(m, jsonOptions)
	if err != nil {
		return reflect.Value{}, Error{Path: path, Err: err}
	}

	return reflect.ValueOf(jsontext.Value(data)), nil
}

// valueToAny converts a Value into a Go value.
func valueToAny(w *walker, path string, src reflect.Value) (reflect.Value, error) {
	x, err := w.fromValue(messagePtr[structpb.Value](src))
	if err != nil {
		return reflect.Value{}, at(err, path)
	}
	// Address x so that a nil result is still a valid (zero) any value.
	return reflect.ValueOf(&x).Elem(), nil
}

// listToSlice converts a ListValue into a []any.
func listToSlice(w *walker, path string, src reflect.Value) (reflect.Value, error) {
	s, err := w.fromList(messagePtr[structpb.ListValue](src))
	if err != nil {
		return reflect.Value{}, at(err, path)
	}

	return reflect.ValueOf(s), nil
}

// structToMap converts a Struct into a map[string]any.
func structToMap(w *walker, path string, src reflect.Value) (reflect.Value, error) {
	m, err := w.fromStruct(messagePtr[structpb.Struct](src))
	if err != nil {
		return reflect.Value{}, at(err, path)
	}

	return reflect.ValueOf(m), nil
}

// anyToValue converts a Go value into a Value.
func anyToValue(w *walker, path string, src reflect.Value) (reflect.Value, error) {
	pb, err := w.toValue(src.Interface())
	if err != nil {
		return reflect.Value{}, at(err, path)
	}

	return reflect.ValueOf(pb), nil
}

// sliceToList converts a []any into a ListValue.
func sliceToList(w *walker, path string, src reflect.Value) (reflect.Value, error) {
	s, ok := reflect.TypeAssert[[]any](src)
	if !ok {
		return reflect.Value{}, ValueError{Path: path, Value: src.Interface(), Reason: reasonUnsupported}
	}

	pb, err := w.toList(s)
	if err != nil {
		return reflect.Value{}, at(err, path)
	}

	return reflect.ValueOf(pb), nil
}

// mapToStruct converts a map[string]any into a Struct.
func mapToStruct(w *walker, path string, src reflect.Value) (reflect.Value, error) {
	m, ok := reflect.TypeAssert[map[string]any](src)
	if !ok {
		return reflect.Value{}, ValueError{Path: path, Value: src.Interface(), Reason: reasonUnsupported}
	}

	pb, err := w.toStruct(m)
	if err != nil {
		return reflect.Value{}, at(err, path)
	}

	return reflect.ValueOf(pb), nil
}

// at prefixes the path of a [ValueError] with prefix; other errors are
// returned unchanged.
func at(err error, prefix string) error {
	if ve, ok := err.(ValueError); ok {
		ve.Path = prefix + ve.Path
		return ve
	}

	return err
}

// toValue converts a Go value into a google.protobuf.Value.
func (w *walker) toValue(x any) (*structpb.Value, error) {
	switch v := x.(type) {
	case nil:
		return structpb.NewNullValue(), nil
	case bool:
		return structpb.NewBoolValue(v), nil
	case string:
		return stringValue(v)
	case float64:
		return numberValue(v)
	case []any:
		if v == nil {
			return structpb.NewNullValue(), nil
		}

		list, err := w.toList(v)
		if err != nil {
			return nil, err
		}

		return structpb.NewListValue(list), nil
	case map[string]any:
		if v == nil {
			return structpb.NewNullValue(), nil
		}

		s, err := w.toStruct(v)
		if err != nil {
			return nil, err
		}

		return structpb.NewStructValue(s), nil
	default:
		return scalarValue(reflect.ValueOf(x))
	}
}

// scalarValue converts booleans, numbers and strings of any named type.
func scalarValue(v reflect.Value) (*structpb.Value, error) {
	switch v.Kind() {
	case reflect.Bool:
		return structpb.NewBoolValue(v.Bool()), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return structpb.NewNumberValue(float64(v.Int())), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return structpb.NewNumberValue(float64(v.Uint())), nil
	case reflect.Float32, reflect.Float64:
		return numberValue(v.Float())
	case reflect.String:
		return stringValue(v.String())
	default:
		return nil, ValueError{Value: v.Interface(), Reason: reasonUnsupported}
	}
}

// numberValue wraps f, rejecting NaN and infinities.
func numberValue(f float64) (*structpb.Value, error) {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return nil, ValueError{Value: f, Reason: reasonNonFinite}
	}

	return structpb.NewNumberValue(f), nil
}

// stringValue wraps s, rejecting invalid UTF-8.
func stringValue(s string) (*structpb.Value, error) {
	if !utf8.ValidString(s) {
		return nil, ValueError{Value: s, Reason: reasonInvalidUTF8}
	}

	return structpb.NewStringValue(s), nil
}

// toList converts the elements of s into a ListValue.
func (w *walker) toList(s []any) (*structpb.ListValue, error) {
	if err := enter(w, s); err != nil {
		return nil, err
	}

	defer leave(w, s)

	values := make([]*structpb.Value, len(s))
	for i, x := range s {
		v, err := w.toValue(x)
		if err != nil {
			return nil, at(err, "["+strconv.Itoa(i)+"]")
		}

		values[i] = v
	}

	return &structpb.ListValue{Values: values}, nil
}

// toStruct converts the entries of m into a Struct.
func (w *walker) toStruct(m map[string]any) (*structpb.Struct, error) {
	if err := enter(w, m); err != nil {
		return nil, err
	}

	defer leave(w, m)

	fields := make(map[string]*structpb.Value, len(m))
	for key, x := range m {
		if !utf8.ValidString(key) {
			return nil, ValueError{Value: key, Reason: reasonInvalidUTF8}
		}

		v, err := w.toValue(x)
		if err != nil {
			return nil, at(err, "."+key)
		}

		fields[key] = v
	}

	return &structpb.Struct{Fields: fields}, nil
}

// fromValue converts a google.protobuf.Value into a Go value. A nil Value, a
// Value without a kind and NullValue all convert to nil.
func (w *walker) fromValue(pb *structpb.Value) (any, error) {
	var out any

	switch kind := pb.GetKind().(type) {
	case *structpb.Value_NumberValue:
		f := kind.NumberValue
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return nil, ValueError{Value: f, Reason: reasonNonFinite}
		}

		out = f
	case *structpb.Value_StringValue:
		out = kind.StringValue
	case *structpb.Value_BoolValue:
		out = kind.BoolValue
	case *structpb.Value_StructValue:
		m, err := w.fromStruct(kind.StructValue)
		if err != nil {
			return nil, err
		}

		out = m
	case *structpb.Value_ListValue:
		s, err := w.fromList(kind.ListValue)
		if err != nil {
			return nil, err
		}

		out = s
	default:
	}

	return out, nil
}

// fromList converts a ListValue into a []any.
func (w *walker) fromList(pb *structpb.ListValue) ([]any, error) {
	if err := enter(w, pb); err != nil {
		return nil, err
	}

	defer leave(w, pb)

	values := pb.GetValues()

	out := make([]any, len(values))
	for i, v := range values {
		x, err := w.fromValue(v)
		if err != nil {
			return nil, at(err, "["+strconv.Itoa(i)+"]")
		}

		out[i] = x
	}

	return out, nil
}

// fromStruct converts a Struct into a map[string]any.
func (w *walker) fromStruct(pb *structpb.Struct) (map[string]any, error) {
	if err := enter(w, pb); err != nil {
		return nil, err
	}

	defer leave(w, pb)

	fields := pb.GetFields()

	out := make(map[string]any, len(fields))
	for key, v := range fields {
		x, err := w.fromValue(v)
		if err != nil {
			return nil, at(err, "."+key)
		}

		out[key] = x
	}

	return out, nil
}
