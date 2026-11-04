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
	plan, planDst := w.structElemPlan(srcValue.Type().Elem(), dstValue.Type().Elem())
	convert := w.primitiveElemFunc(srcValue.Type().Elem(), dstValue.Type().Elem())

	for i := range n {
		name := w.elementName(fieldName, i)

		srcElem := srcValue.Index(i)
		srcNil := srcElem.Kind() == reflect.Pointer && srcElem.IsNil()

		switch {
		case elemIsPointer && srcNil:
			slots.Index(i).Set(reflect.Zero(dstValue.Type().Elem()))
		case plan != nil && !srcNil:
			w.convertNestedStruct(path, name, srcElem, slots.Index(i), planDst, plan)
		case convert != nil:
			w.convertPrimitiveElem(path, name, convert, srcElem, slots.Index(i))
		default:
			// A pointer slot is converted into itself so a codec or the
			// built-in convertible path decides whether to allocate.
			w.convertValueAt(path, name, srcElem, slots.Index(i))
		}

		dstValue.SetLen(i + 1)
	}
}

// primitiveElemFunc returns the conversion function for slice elements of
// type src into elements of type dst when [walker.convertValueAt] would end
// in [primitive.Convert] with it: distinct, unassignable primitive types
// with a table entry and no codecs. It returns nil otherwise.
func (w *walker) primitiveElemFunc(src, dst reflect.Type) primitive.Func {
	if w.cfg.Codecs.HasCodecs() || w.cfg.Codecs.HasErrorCodecs() ||
		!reflectutils.IsPrimitive(src.Kind()) || w.isTypeCachedAssignable(src, dst) {
		return nil
	}

	convert, _ := primitive.Lookup(src.Kind(), dst.Kind())

	return convert
}

// convertPrimitiveElem converts the slice element src into the slot dst
// with convert; it is [walker.convertValueAt] reduced to the steps that
// apply to the elements [walker.primitiveElemFunc] accepts. Both are
// addressable, as slice elements always are.
func (w *walker) convertPrimitiveElem(path, name string, convert primitive.Func, src, dst reflect.Value) {
	if w.skipValue(path, name, src) {
		return
	}

	// A panic aborts the whole conversion, as no codec can recover it, so
	// the level is left without a deferred call.
	w.descend(path)
	w.convertPrimitive(name, convert, src, dst)
	w.depth--
}

// structElemPlan returns the plan and the destination struct type converting
// container elements of type src into elements of type dst when
// [walker.convertNestedStruct] applies to them: distinct (pointer to) struct
// types, and neither codecs nor ignore paths. It returns a nil plan
// otherwise.
func (w *walker) structElemPlan(src, dst reflect.Type) (*structPlan, reflect.Type) {
	if w.cfg.Codecs.HasCodecs() || w.cfg.Codecs.HasErrorCodecs() || len(w.cfg.IgnorePaths) > 0 ||
		!isStructPair(src, dst) || w.isTypeCachedAssignable(src, dst) {
		return nil, nil
	}

	dstType := reflectutils.IndirectType(dst)

	return w.planFor(reflectutils.IndirectType(src), dstType), dstType
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

	elems := w.mapElems(srcValue.Type().Elem(), dstValue.Type().Elem())
	keys := w.mapKeys(srcValue, dstValue.Type().Key())

	for it := srcValue.MapRange(); it.Next(); {
		key := keys.source(it)
		dstValue.SetMapIndex(w.convertMapKeyOf(fieldName, key, &keys), w.convertMapValue(path, fieldName, key, it.Value(), &elems))
	}
}

// mapKeys describes the conversion of the keys of one map. Keys that are
// assigned or converted as scalars are stored right away, as SetMapIndex
// copies them, so buffers receive them instead of a copy per entry.
type mapKeys struct {
	dstType reflect.Type
	// assignable and scalar relate the source key type to dstType; see
	// [isScalarPair] for scalar.
	assignable, scalar bool
	// buf receives the source keys when valid; dstBuf receives the
	// converted scalar keys.
	buf, dstBuf reflect.Value
}

// mapKeys describes the conversion of the keys of the map src into keys of
// type dst.
func (w *walker) mapKeys(src reflect.Value, dst reflect.Type) mapKeys {
	srcType := src.Type().Key()
	keys := mapKeys{dstType: dst, assignable: w.isTypeCachedAssignable(srcType, dst)}
	keys.scalar = !keys.assignable && isScalarPair(srcType.Kind(), dst.Kind())

	// SetIterKey requires an exported map.
	if (keys.assignable || keys.scalar) && src.CanInterface() {
		keys.buf = reflect.New(srcType).Elem()
	}

	if keys.scalar {
		keys.dstBuf = reflect.New(dst).Elem()
	}

	return keys
}

// source returns the key of the entry it is at.
func (k *mapKeys) source(it *reflect.MapIter) reflect.Value {
	if !k.buf.IsValid() {
		return it.Key()
	}

	k.buf.SetIterKey(it)

	return k.buf
}

// convertMapKeyOf converts key as described by keys.
func (w *walker) convertMapKeyOf(fieldName string, key reflect.Value, keys *mapKeys) reflect.Value {
	switch {
	case keys.assignable:
		return key
	case keys.scalar && key.CanInterface():
		if w.cfg.OverflowCheck && w.overflows(key, keys.dstBuf) {
			panic(OverflowError{Field: makeFieldName(fieldName, "key"), From: key.Kind(), To: keys.dstBuf.Kind(), Value: key.Interface()})
		}

		setScalar(key, keys.dstBuf)

		return keys.dstBuf
	default:
		return w.convertMapKey(fieldName, key, keys.dstType)
	}
}

// mapElems describes the conversion of the values of one map. The fields
// depend on the types alone, as every value of a map has its element type.
type mapElems struct {
	// dstType is the destination map value type.
	dstType reflect.Type
	// assignable, convertible and canConvert relate the source value type
	// to dstType; see [walker.canConvertType] for canConvert. scalar
	// reports a kind pair [setScalar] converts.
	assignable, convertible, canConvert, scalar bool
	// plan and planDst are the result of [walker.structElemPlan].
	plan    *structPlan
	planDst reflect.Type
	// slot receives converted values that are stored right away, as
	// SetMapIndex copies them; see [mapElems.buffer].
	slot reflect.Value
}

// buffer returns the reusable value slot of type dstType.
func (e *mapElems) buffer() reflect.Value {
	if !e.slot.IsValid() {
		e.slot = reflect.New(e.dstType).Elem()
	}

	return e.slot
}

// mapElems describes the conversion of src map values into dst map values.
func (w *walker) mapElems(src, dst reflect.Type) mapElems {
	elems := mapElems{
		dstType:     dst,
		assignable:  w.isTypeCachedAssignable(src, dst),
		convertible: w.isTypeCachedConvertible(src, dst),
		canConvert:  w.canConvertType(src, dst),
		scalar:      isScalarPair(src.Kind(), dst.Kind()),
	}
	elems.plan, elems.planDst = w.structElemPlan(src, dst)

	return elems
}

// convertKnown is [reflectutils.ConvertValue] for a value whose type is known
// to convert to t, so the convertibility is not checked again unless the
// result depends on the value.
func convertKnown(v reflect.Value, t reflect.Type) (reflect.Value, bool) {
	if t.Kind() == reflect.String || v.Kind() == reflect.Slice {
		return reflectutils.ConvertValue(v, t)
	}

	return v.Convert(t), true
}

// convertMapKey converts a map key that is not assignable to dstKeyType.
func (w *walker) convertMapKey(fieldName string, key reflect.Value, dstKeyType reflect.Type) reflect.Value {
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
func (w *walker) convertMapValue(path, fieldName string, key reflect.Value, srcMapValue reflect.Value, elems *mapElems) reflect.Value {
	valueFieldName := w.entryName(fieldName, key)

	// Preserve the original zero-cost alias for default map conversion.
	if len(w.cfg.IgnorePaths) == 0 && !w.cfg.CopySlices && !w.cfg.Codecs.HasErrorCodecs() && !w.cfg.OverflowCheck {
		if elems.assignable {
			return srcMapValue
		}

		if elems.convertible {
			if elems.scalar && srcMapValue.CanInterface() {
				slot := elems.buffer()
				setScalar(srcMapValue, slot)

				return slot
			}

			if converted, ok := convertKnown(srcMapValue, elems.dstType); ok {
				return converted
			}
		}
	}

	// Composite values are walked so ignore paths and copy options apply
	// inside nested slices, arrays and maps; scalars keep the primitive path.
	switch reflectutils.IndirectType(elems.dstType).Kind() {
	case reflect.Struct, reflect.Slice, reflect.Array, reflect.Map:
		return w.convertStructMapValue(path, valueFieldName, srcMapValue, elems)
	default:
		return w.convertPrimitiveMapValue(path, valueFieldName, srcMapValue, elems)
	}
}

// convertStructMapValue handles conversion of struct values in maps.
func (w *walker) convertStructMapValue(path, fieldName string, srcMapValue reflect.Value, elems *mapElems) reflect.Value {
	dstValueType := elems.dstType
	srcNil := srcMapValue.Kind() == reflect.Pointer && srcMapValue.IsNil()

	if dstValueType.Kind() == reflect.Pointer && srcNil {
		return reflect.Zero(dstValueType)
	}

	if !elems.canConvert {
		return reflect.Zero(dstValueType)
	}

	if elems.plan != nil && !srcNil {
		// Without codecs nothing keeps the slot, so one serves every entry.
		slot := elems.buffer()
		slot.SetZero()
		w.convertNestedStruct(path, fieldName, srcMapValue, slot, elems.planDst, elems.plan)

		return slot
	}

	dstMapValue := reflect.New(dstValueType).Elem()
	w.convertValueAt(path, fieldName, srcMapValue, dstMapValue)

	return dstMapValue
}

// convertPrimitiveMapValue handles conversion of primitive values in maps.
func (w *walker) convertPrimitiveMapValue(path, fieldName string, srcMapValue reflect.Value, elems *mapElems) reflect.Value {
	// Allocate the exact destination type: a pointer-valued map needs a
	// pointer here, which convertValue allocates for non-assignable sources.
	if w.cfg.Codecs.HasErrorCodecs() || w.hasIgnorePathBelow(path) {
		dstMapValue := reflect.New(elems.dstType).Elem()
		w.convertValueAt(path, fieldName, srcMapValue, dstMapValue)

		return dstMapValue
	}

	// Values assigned or converted here are stored right away, so they share
	// the slot.
	if elems.assignable {
		slot := elems.buffer()
		slot.Set(srcMapValue)

		return slot
	}

	if elems.convertible {
		if elems.scalar && srcMapValue.CanInterface() {
			slot := elems.buffer()
			if w.cfg.OverflowCheck && w.overflows(srcMapValue, slot) {
				panic(OverflowError{Field: fieldName, From: srcMapValue.Kind(), To: slot.Kind(), Value: srcMapValue.Interface()})
			}

			setScalar(srcMapValue, slot)

			return slot
		}

		if converted, ok := convertKnown(srcMapValue, elems.dstType); ok {
			if w.cfg.OverflowCheck && w.overflows(srcMapValue, converted) {
				panic(OverflowError{Field: fieldName, From: srcMapValue.Kind(), To: converted.Kind(), Value: srcMapValue.Interface()})
			}

			slot := elems.buffer()
			slot.Set(converted)

			return slot
		}
	}

	dstMapValue := reflect.New(elems.dstType).Elem()
	w.convertValueAt(path, fieldName, srcMapValue, dstMapValue)

	return dstMapValue
}
