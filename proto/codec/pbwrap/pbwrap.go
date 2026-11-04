// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package pbwrap

import (
	"reflect"
	"strings"

	convcodec "github.com/altessa-s/go-convert/codec"
	reflectutils "github.com/altessa-s/go-convert/internal/reflect"
)

const (
	// WrapperPrefix is the reflect.Type.String() prefix shared by all protobuf
	// wrapper types (e.g. "wrapperspb.StringValue").
	WrapperPrefix = "wrapperspb."
	// BytesValue is excluded from conversion because its []byte nature requires
	// special handling that the generic wrapper codec does not support.
	BytesValue = "wrapperspb.BytesValue"
	// ValueFieldName is the standard field name inside every protobuf wrapper
	// struct that holds the primitive payload.
	ValueFieldName = "Value"

	// Protobuf wrapper type name constants used for documentation and testing.
	StringValue = "wrapperspb.StringValue"
	Int32Value  = "wrapperspb.Int32Value"
	Int64Value  = "wrapperspb.Int64Value"
	UInt32Value = "wrapperspb.UInt32Value"
	UInt64Value = "wrapperspb.UInt64Value"
	BoolValue   = "wrapperspb.BoolValue"
	FloatValue  = "wrapperspb.FloatValue"
	DoubleValue = "wrapperspb.DoubleValue"
)

// Registry performs protobuf wrapper conversions.
// It is intentionally stateless; conversion is done via type inspection.
type Registry struct{}

// NewRegistry creates a new protobuf conversion registry.
// The registry is pre-populated with all standard wrapper type conversions.
func NewRegistry() *Registry {
	return &Registry{}
}

// TryConvert attempts protobuf wrapper conversion.
// Returns true if conversion was handled (including skipped due to options), false otherwise.
func (r *Registry) TryConvert(src, dst reflect.Value, opts *options) bool {
	srcType := reflectutils.IndirectType(src.Type())
	dstType := reflectutils.IndirectType(dst.Type())

	srcValue := indirectSource(src)

	// Check if this is a protobuf wrapper conversion scenario
	if (!reflectutils.IsPrimitive(srcType.Kind()) || !isWrapper(dstType)) &&
		(!reflectutils.IsPrimitive(dstType.Kind()) || !isWrapper(srcType)) {
		return false
	}

	// A nil wrapper pointer has no payload. Treat it as an ignored value so
	// value destinations retain their zero value and no invalid reflect.Value
	// reaches IsZero or unwrapWrapper.
	if !srcValue.IsValid() {
		return isWrapper(srcType)
	}

	// Check for zero value options
	if opts.ignoreZeroValues && srcValue.IsZero() {
		return true
	}

	// Unwrap protobuf wrapper to primitive type
	if isWrapper(srcType) {
		unwrappedValue := unwrapWrapper(srcValue)
		if !unwrappedValue.IsValid() {
			return false
		}
		convertedValue, ok := reflectutils.ConvertValue(unwrappedValue, dstType)
		if !ok {
			return false
		}

		// Check for zero wrapper option BEFORE creating dst
		// Return true to indicate we "handled" it (by ignoring it)
		if opts.ignoreZeroWrappers && unwrappedValue.IsZero() {
			return true
		}

		// Only create dst if we're not ignoring this value
		destination, ok := indirectDestination(dst)
		if !ok {
			return false
		}
		destination.Set(convertedValue)
		return true
	}

	// Wrap primitive type to protobuf wrapper
	if isWrapper(dstType) {
		// Check for zero value option BEFORE creating dst
		// Return true to indicate we "handled" it (by ignoring it)
		if opts.ignoreZeroValues && srcValue.IsZero() {
			return true
		}

		valueField, ok := dstType.FieldByName(ValueFieldName)
		if !ok {
			return false
		}
		wrappedValue, ok := reflectutils.ConvertValue(srcValue, valueField.Type)
		if !ok {
			return false
		}

		// Directly handle wrapping here instead of using converter
		destination, ok := indirectDestination(dst)
		if !ok {
			return false
		}
		destination.FieldByName(ValueFieldName).Set(wrappedValue)
		return true
	}

	return false
}

// indirectSource dereferences all pointer layers, returning an invalid value
// when any layer is nil.
func indirectSource(src reflect.Value) reflect.Value {
	return reflectutils.IndirectValue(src)
}

// indirectDestination allocates all pointer layers and returns the final
// settable value.
func indirectDestination(dst reflect.Value) (reflect.Value, bool) {
	for dst.Kind() == reflect.Pointer {
		if dst.IsNil() {
			if !dst.CanSet() {
				return reflect.Value{}, false
			}
			dst.Set(reflect.New(dst.Type().Elem()))
		}
		dst = dst.Elem()
	}
	return dst, dst.CanSet()
}

// New creates a Codec for bidirectional conversion between protobuf wrapper types
// and primitive Go types. Supports StringValue, Int32Value, Int64Value, UInt32Value, UInt64Value,
// BoolValue, FloatValue, and DoubleValue. BytesValue is excluded as it requires special handling.
//
// Example:
//
//	codec := pbwrap.New()
//	codec := pbwrap.New(pbwrap.WithIgnoreZeroValues(), pbwrap.WithIgnoreZeroWrappers())
func New(opt ...Option) convcodec.Codec {
	registry := NewRegistry()
	opts := newOptions(opt...)
	opts.registry = registry

	return func(fieldName string, src, dst reflect.Value, next convcodec.CodecHandler) {
		// Try protobuf wrapper conversions using the registry approach
		if opts.registry.TryConvert(src, dst, opts) {
			return
		}

		next(fieldName, src, dst)
	}
}

// isWrapper reports whether the given type is a protobuf wrapper type.
// It checks for struct types that have the protobuf wrapper prefix but excludes BytesValue
// which requires different handling due to its []byte nature.
func isWrapper(t reflect.Type) bool {
	return t.Kind() == reflect.Struct &&
		strings.HasPrefix(t.String(), WrapperPrefix) &&
		t.String() != BytesValue
}

// unwrapWrapper extracts the primitive value from a protobuf wrapper.
// It accesses the "Value" field of the wrapper struct and returns its reflect.Value.
func unwrapWrapper(v reflect.Value) reflect.Value {
	if !v.IsValid() {
		return reflect.Value{}
	}
	v = indirectSource(v)
	if !v.IsValid() {
		return reflect.Value{}
	}
	return v.FieldByName(ValueFieldName)
}
