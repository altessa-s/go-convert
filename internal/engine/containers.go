// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package engine

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/altessa-s/go-convert/internal/primitive"

	reflectutils "github.com/altessa-s/go-convert/internal/reflect"
)

// convertSlices converts source slice to destination slice with element type conversion.
func (w *walker) convertSlices(path, fieldName string, srcValue reflect.Value, dstValue reflect.Value) {
	srcValue = reflectutils.IndirectValue(srcValue)
	if !srcValue.IsValid() {
		return
	}

	reflectutils.MakeDst(&dstValue, reflectutils.IndirectType(dstValue.Type()))

	dstValue = reflectutils.IndirectValue(dstValue)
	if srcValue.IsNil() {
		dstValue.Set(reflect.Zero(dstValue.Type()))
		return
	}

	n := srcValue.Len()
	dstValue.Set(reflect.MakeSlice(dstValue.Type(), 0, n))
	// Convert straight into the backing array; the visible length grows only
	// after each element converts, keeping the completed-prefix behavior.
	slots := dstValue.Slice(0, n)
	elemIsPointer := dstValue.Type().Elem().Kind() == reflect.Pointer

	for i := range n {
		name := w.elementName(fieldName, i)

		srcElem := srcValue.Index(i)
		if elemIsPointer {
			if srcElem.Kind() == reflect.Pointer && srcElem.IsNil() {
				slots.Index(i).Set(reflect.Zero(dstValue.Type().Elem()))
			} else {
				// Convert into the pointer slot itself so a codec or the
				// built-in convertible path decides whether to allocate.
				w.convertValueAt(path, name, srcElem, slots.Index(i))
			}
		} else {
			w.convertValueAt(path, name, srcElem, slots.Index(i))
		}

		dstValue.SetLen(i + 1)
	}
}

// overflows reports whether writing the primitive src into the primitive dst
// loses data.
func (w *walker) overflows(src, dst reflect.Value) bool {
	return reflectutils.IsPrimitive(src.Kind()) && reflectutils.IsPrimitive(dst.Kind()) && primitive.WouldOverflow(src, dst)
}

// elementName returns the codec-visible name of element i. Names are only
// built when something observes them: codecs, ignored field names or
// overflow errors.
func (w *walker) elementName(fieldName string, i int) string {
	if !w.namesObserved {
		return fieldName
	}

	return makeFieldName(fieldName, strconv.Itoa(i))
}

// entryName returns the codec-visible name of the map entry with key.
func (w *walker) entryName(fieldName string, key reflect.Value) string {
	if !w.namesObserved {
		return fieldName
	}

	return makeFieldName(fieldName, fmt.Sprintf("[%v]", key.Interface()))
}

// convertArrays converts source arrays to destination arrays element by element.
// The destination length bounds the loop so a malformed descriptor cannot
// write past it; for same-type arrays the lengths are equal.
func (w *walker) convertArrays(path, fieldName string, srcValue reflect.Value, dstValue reflect.Value) {
	srcValue = reflectutils.IndirectValue(srcValue)
	if !srcValue.IsValid() {
		return
	}

	reflectutils.MakeDst(&dstValue, reflectutils.IndirectType(dstValue.Type()))
	dstValue = reflectutils.IndirectValue(dstValue)

	n := min(srcValue.Len(), dstValue.Len())
	for i := range n {
		w.convertValueAt(path, w.elementName(fieldName, i), srcValue.Index(i), dstValue.Index(i))
	}
}

// convertMaps converts source map to destination map with key/value type conversion.
func (w *walker) convertMaps(path, fieldName string, srcValue reflect.Value, dstValue reflect.Value) {
	srcValue = reflectutils.IndirectValue(srcValue)
	if !srcValue.IsValid() {
		return
	}

	reflectutils.MakeDst(&dstValue, reflectutils.IndirectType(dstValue.Type()))

	dstValue = reflectutils.IndirectValue(dstValue)
	if srcValue.IsNil() {
		dstValue.Set(reflect.Zero(dstValue.Type()))
		return
	}

	// Initialize destination map if needed
	if srcValue.Len() > 0 {
		dstValue.Set(reflect.MakeMapWithSize(dstValue.Type(), srcValue.Len()))
	} else {
		dstValue.Set(reflect.MakeMap(dstValue.Type()))
	}

	dstKeyType := dstValue.Type().Key()
	dstValueType := dstValue.Type().Elem()

	for _, key := range srcValue.MapKeys() {
		srcMapValue := srcValue.MapIndex(key)

		// Convert key using helper function
		dstKey := w.convertMapKey(fieldName, key, dstKeyType)

		// Convert value using helper function
		dstMapValue := w.convertMapValue(path, fieldName, key, srcMapValue, dstValueType)

		// Set the converted key-value pair in destination map
		dstValue.SetMapIndex(dstKey, dstMapValue)
	}
}

// convertMapKey converts map key with fallback chain.
func (w *walker) convertMapKey(fieldName string, key reflect.Value, dstKeyType reflect.Type) reflect.Value {
	if key.Type().AssignableTo(dstKeyType) {
		return key
	}

	if converted, ok := reflectutils.ConvertValue(key, dstKeyType); ok {
		if w.cfg.OverflowCheck && w.overflows(key, converted) {
			panic(OverflowError{Field: makeFieldName(fieldName, "key"), From: key.Kind(), To: converted.Kind(), Value: key.Interface()})
		}

		return converted
	}

	dstKey := reflect.New(dstKeyType).Elem()
	// Map keys are part of the map's identity, not of the value
	// namespace: ignore paths never apply to them.
	w.convertValueAt(noIgnorePath, makeFieldName(fieldName, "key"), key, dstKey)

	return dstKey
}

// convertMapValue converts map value handling both struct and primitive types.
func (w *walker) convertMapValue(
	path, fieldName string, key reflect.Value, srcMapValue reflect.Value, dstValueType reflect.Type,
) reflect.Value {
	valueFieldName := w.entryName(fieldName, key)

	// Preserve the original zero-cost alias for default map conversion.
	if len(w.cfg.IgnorePaths) == 0 && !w.cfg.CopySlices && !w.cfg.Codecs.HasErrorCodecs() && !w.cfg.OverflowCheck {
		if srcMapValue.Type().AssignableTo(dstValueType) {
			return srcMapValue
		}

		if converted, ok := reflectutils.ConvertValue(srcMapValue, dstValueType); ok {
			return converted
		}
	}

	// Composite values are walked so ignore paths and copy options apply
	// inside nested slices, arrays and maps; scalars keep the primitive path.
	switch reflectutils.IndirectType(dstValueType).Kind() {
	case reflect.Struct, reflect.Slice, reflect.Array, reflect.Map:
		return w.convertStructMapValue(path, valueFieldName, srcMapValue, dstValueType)
	default:
		return w.convertPrimitiveMapValue(path, valueFieldName, srcMapValue, dstValueType)
	}
}

// convertStructMapValue handles conversion of struct values in maps.
func (w *walker) convertStructMapValue(path, fieldName string, srcMapValue reflect.Value, dstValueType reflect.Type) reflect.Value {
	if dstValueType.Kind() == reflect.Pointer && srcMapValue.Kind() == reflect.Pointer && srcMapValue.IsNil() {
		return reflect.Zero(dstValueType)
	}

	if !w.canConvertType(srcMapValue.Type(), dstValueType) {
		return reflect.Zero(dstValueType)
	}

	if dstValueType.Kind() == reflect.Pointer {
		slot := reflect.New(dstValueType).Elem()
		w.convertValueAt(path, fieldName, srcMapValue, slot)

		return slot
	}

	dstMapValue := reflect.New(dstValueType).Elem()
	w.convertValueAt(path, fieldName, srcMapValue, dstMapValue)

	return dstMapValue
}

// convertPrimitiveMapValue handles conversion of primitive values in maps.
func (w *walker) convertPrimitiveMapValue(path, fieldName string, srcMapValue reflect.Value, dstValueType reflect.Type) reflect.Value {
	// Allocate the exact destination type: a pointer-valued map needs a
	// pointer here, which convertValue allocates for non-assignable sources.
	dstMapValue := reflect.New(dstValueType).Elem()
	if w.cfg.Codecs.HasErrorCodecs() || w.hasIgnorePathBelow(path) {
		w.convertValueAt(path, fieldName, srcMapValue, dstMapValue)
		return dstMapValue
	}

	if srcMapValue.Type().AssignableTo(dstValueType) {
		dstMapValue.Set(srcMapValue)
	} else if converted, ok := reflectutils.ConvertValue(srcMapValue, dstValueType); ok {
		if w.cfg.OverflowCheck && w.overflows(srcMapValue, converted) {
			panic(OverflowError{Field: fieldName, From: srcMapValue.Kind(), To: converted.Kind(), Value: srcMapValue.Interface()})
		}

		dstMapValue.Set(converted)
	} else {
		w.convertValueAt(path, fieldName, srcMapValue, dstMapValue)
	}

	return dstMapValue
}
