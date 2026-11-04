// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package mapslice

import (
	"reflect"

	"github.com/altessa-s/go-convert/reflectx"

	convcodec "github.com/altessa-s/go-convert/codec"
)

// optimizer provides optimized type checking for map-to-slice conversions.
type optimizer struct {
	isMapToSlice bool
}

// newOptimizer creates an optimizer for the given types.
// Returns an optimizer that caches type information for fast conversion checks.
func newOptimizer(srcType, dstType reflect.Type) *optimizer {
	return &optimizer{
		isMapToSlice: srcType.Kind() == reflect.Map && dstType.Kind() == reflect.Slice,
	}
}

// canConvert reports whether map elements can be appended without conversion.
func (opt *optimizer) canConvert(srcElem, dstElem reflect.Type) bool {
	return opt.isMapToSlice && srcElem.AssignableTo(dstElem)
}

// Values is a Codec that converts map values to a slice.
// The map value type must match the slice element type, or slice elements must be any.
// If the destination slice is nil, a new slice is created with appropriate capacity.
// The order of values corresponds to the iteration order of map keys (not guaranteed).
//
// Example:
//
//	source := map[string]int{"a": 1, "b": 2}
//	var dst []int
//	// After conversion: dst contains [1, 2] (order may vary)
func Values(fieldName string, src, dst reflect.Value, next convcodec.CodecHandler) {
	sourceMap, ok := mapSource(src)
	if !ok {
		next(fieldName, src, dst)
		return
	}

	srcType := sourceMap.Type()
	dstType := reflectx.IndirectType(dst.Type())

	// Use optimized type checking
	opt := newOptimizer(srcType, dstType)
	if !opt.isMapToSlice {
		next(fieldName, src, dst)
		return
	}

	// skip if the type of slice elements does not match the type of map values
	if !opt.canConvert(srcType.Elem(), dstType.Elem()) {
		next(fieldName, src, dst)
		return
	}

	sliceDst, ok := sliceDestination(dst)
	if !ok {
		next(fieldName, src, dst)
		return
	}

	keys := sourceMap.MapKeys()
	if sliceDst.IsNil() {
		sliceDst.Set(reflect.MakeSlice(dstType, 0, len(keys)))
	}

	for i := range keys {
		sliceDst.Set(reflect.Append(sliceDst, sourceMap.MapIndex(keys[i])))
	}
}

// Keys is a Codec that converts map keys to a slice.
// The map key type must match the slice element type, or slice elements must be any.
// If the destination slice is nil, a new slice is created with appropriate capacity.
// The order of keys corresponds to the iteration order of map keys (not guaranteed).
//
// Example:
//
//	source := map[string]int{"apple": 1, "banana": 2}
//	var dst []string
//	// After conversion: dst contains ["apple", "banana"] (order may vary)
func Keys(fieldName string, src, dst reflect.Value, next convcodec.CodecHandler) {
	sourceMap, ok := mapSource(src)
	if !ok {
		next(fieldName, src, dst)
		return
	}

	srcType := sourceMap.Type()
	dstType := reflectx.IndirectType(dst.Type())

	// Use optimized type checking
	opt := newOptimizer(srcType, dstType)
	if !opt.isMapToSlice {
		next(fieldName, src, dst)
		return
	}

	// skip if the type of slice elements does not match the type of map keys
	if !opt.canConvert(srcType.Key(), dstType.Elem()) {
		next(fieldName, src, dst)
		return
	}

	sliceDst, ok := sliceDestination(dst)
	if !ok {
		next(fieldName, src, dst)
		return
	}

	keys := sourceMap.MapKeys()
	if sliceDst.IsNil() {
		sliceDst.Set(reflect.MakeSlice(dstType, 0, len(keys)))
	}

	for i := range keys {
		sliceDst.Set(reflect.Append(sliceDst, keys[i]))
	}
}

// mapSource dereferences a non-nil pointer chain to a map. Nil pointers are
// delegated to the next codec because they carry no map to extract.
func mapSource(src reflect.Value) (reflect.Value, bool) {
	src = reflectx.IndirectValue(src)
	return src, src.IsValid() && src.Kind() == reflect.Map
}

// sliceDestination dereferences and allocates pointer layers around a slice
// destination, returning the settable slice value.
func sliceDestination(dst reflect.Value) (reflect.Value, bool) {
	if dst.Kind() == reflect.Pointer && dst.IsNil() && !dst.CanSet() {
		return reflect.Value{}, false
	}

	reflectx.MakeDst(&dst, reflectx.IndirectType(dst.Type()))
	dst = reflectx.IndirectValue(dst)

	return dst, dst.Kind() == reflect.Slice && dst.CanSet()
}
