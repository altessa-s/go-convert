// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package engine

import (
	"reflect"
	"strings"

	"github.com/altessa-s/go-convert/internal/typeinfo"

	reflectutils "github.com/altessa-s/go-convert/internal/reflect"
)

// isTypeCachedAssignable reports whether srcType is assignable to dstType,
// answering the trivial cases directly and memoizing the rest: identical
// types are assignable, and types of different kinds are not unless dstType
// is an interface.
func (w *walker) isTypeCachedAssignable(srcType, dstType reflect.Type) bool {
	if srcType == dstType {
		return true
	}

	if dstType.Kind() != reflect.Interface && srcType.Kind() != dstType.Kind() {
		return false
	}

	return w.types.Get(srcType).AssignableTo(srcType, dstType)
}

// isTypeCachedConvertible reports whether srcType is convertible to
// dstType, memoizing the answer. Identical types and scalar kind pairs (see
// [isScalarPair]) always convert.
func (w *walker) isTypeCachedConvertible(srcType, dstType reflect.Type) bool {
	if srcType == dstType || isScalarPair(srcType.Kind(), dstType.Kind()) {
		return true
	}

	return w.types.Get(srcType).ConvertibleTo(srcType, dstType)
}

// isMergeableStruct reports whether t (after pointer indirection) is a struct
// that [WithSparseMerge] can merge field-by-field, i.e. it has at least one
// exported field. Structs without exported fields (time.Time and similar
// opaque types) cannot be merged and must be assigned wholesale.
func (w *walker) isMergeableStruct(t reflect.Type) bool {
	if reflectutils.IndirectType(t).Kind() != reflect.Struct {
		return false
	}

	return len(w.types.Get(t).ByName) > 0
}

// canConvertType reports whether a value of src type can produce a
// meaningful value of dst type. It is used before allocating pointer elements
// so an incompatible struct -> *string pair stays nil instead of pointing at
// the zero value. When codecs are registered they get the benefit of the
// doubt; the codec chain sees the actual values.
func (w *walker) canConvertType(src, dst reflect.Type) bool {
	if w.canConvertTypeStructural(src, dst) {
		return true
	}

	return w.cfg.Codecs.HasCodecs() || w.cfg.Codecs.HasErrorCodecs()
}

// canConvertTypeStructural ignores codecs and reports only the built-in
// structural conversions.
func (w *walker) canConvertTypeStructural(src, dst reflect.Type) bool {
	src = reflectutils.IndirectType(src)

	dst = reflectutils.IndirectType(dst)
	if src.AssignableTo(dst) || src.ConvertibleTo(dst) {
		return true
	}

	switch {
	case src.Kind() == reflect.Struct && dst.Kind() == reflect.Struct:
		return true
	case src.Kind() == reflect.Struct && dst.Kind() == reflect.Slice && reflectutils.IndirectType(dst.Elem()) == src:
		return true
	case src.Kind() == reflect.Slice && dst.Kind() == reflect.Slice:
		return true
	case src.Kind() == reflect.Array && dst.Kind() == reflect.Array:
		return true
	case src.Kind() == reflect.Map && dst.Kind() == reflect.Map:
		return true
	default:
		return false
	}
}

// typeHasMergeableStruct reports whether t contains a struct with exported
// fields below any combination of pointers, slices, arrays and maps.
var typeMergeableCache typeinfo.Bounded[bool]

// typeHasMergeableStruct reports whether t reaches a struct with exported fields through pointers, containers or interfaces.
func typeHasMergeableStruct(t reflect.Type) bool {
	t = reflectutils.IndirectType(t)

	return typeMergeableCache.Get(t, func() bool {
		return typeHasMergeableStructSeen(t, make(map[reflect.Type]bool))
	})
}

// typeHasMergeableStructSeen is typeHasMergeableStruct with a guard against recursive types.
func typeHasMergeableStructSeen(t reflect.Type, seen map[reflect.Type]bool) bool {
	t = reflectutils.IndirectType(t)
	if seen[t] {
		return false
	}

	seen[t] = true
	switch t.Kind() {
	case reflect.Interface:
		return true
	case reflect.Struct:
		return typeHasExportedFields(t)
	case reflect.Slice, reflect.Array, reflect.Map:
		return typeHasMergeableStructSeen(t.Elem(), seen)
	default:
		return false
	}
}

// typeHasExportedFields reports whether the struct type t has an exported field.
func typeHasExportedFields(t reflect.Type) bool {
	for field := range t.Fields() {
		if field.IsExported() {
			return true
		}
	}

	return false
}

// isStructZero returns true if all fields of the struct are zero values.
func isStructZero(v reflect.Value) bool {
	// Indexing avoids the allocation of the Value.Fields iterator.
	n := v.NumField()
	for i := range n {
		if !v.Field(i).IsZero() {
			return false
		}
	}

	return true
}

// isSparseStructEmpty reports whether v carries no update instructions: every
// pointer/slice/map/interface field is nil and every other field is its zero
// value. Used by [WithSparseMerge] to detect a present-but-empty nested update
// struct, which signals "clear the whole field" rather than "merge nothing".
func isSparseStructEmpty(v reflect.Value) bool {
	// Indexing avoids the allocation of the Value.Fields iterator.
	n := v.NumField()
	for i := range n {
		field := v.Field(i)
		switch field.Kind() {
		case reflect.Pointer, reflect.Slice, reflect.Map, reflect.Interface:
			if !field.IsNil() {
				return false
			}
		default:
			if !field.IsZero() {
				return false
			}
		}
	}

	return true
}

// joinPath joins two dotted path segments, skipping empty ones.
func joinPath(parent, field string) string {
	switch {
	case parent == "":
		return field
	case field == "":
		return parent
	default:
		return parent + "." + field
	}
}

// typeCanContainSlice reports whether values of t can contain a slice below
// pointers, exported struct fields, arrays, slices or map values.
var typeSliceCache typeinfo.Bounded[bool]

// typeCanContainSlice reports whether values of t can hold a slice that WithCopySlices must clone.
func typeCanContainSlice(t reflect.Type) bool {
	t = reflectutils.IndirectType(t)

	return typeSliceCache.Get(t, func() bool {
		return typeCanContainSliceSeen(t, make(map[reflect.Type]bool))
	})
}

// typeCanContainSliceSeen is typeCanContainSlice with a guard against recursive types.
func typeCanContainSliceSeen(t reflect.Type, seen map[reflect.Type]bool) bool {
	t = reflectutils.IndirectType(t)
	if seen[t] {
		return false
	}

	seen[t] = true
	switch t.Kind() {
	case reflect.Slice:
		return true
	case reflect.Array, reflect.Map:
		return typeCanContainSliceSeen(t.Elem(), seen)
	case reflect.Struct:
		for field := range t.Fields() {
			if field.IsExported() && typeCanContainSliceSeen(field.Type, seen) {
				return true
			}
		}
	default:
		return false
	}

	return false
}

// cloneSlicesInPlace replaces every reachable slice in v with a new backing
// array. It walks exported struct fields, arrays and map values; pointers and
// interfaces are left aliased because cloning them would change identity.
// The caller has already copied the top-level value, so unexported struct
// fields keep the source's state.
func cloneSlicesInPlace(v reflect.Value) {
	if !v.IsValid() || !v.CanSet() {
		return
	}

	switch v.Kind() {
	case reflect.Slice:
		v.Set(cloneSlice(v))
		cloneElementSlices(v)
	case reflect.Array:
		cloneElementSlices(v)
	case reflect.Struct:
		for i := range v.NumField() {
			field := v.Field(i)
			if field.CanSet() && v.Type().Field(i).IsExported() {
				cloneSlicesInPlace(field)
			}
		}
	case reflect.Map:
		cloneMapValueSlices(v)
	default:
		// Other kinds retain their copied value and identity.
	}
}

// cloneElementSlices clones the slices reachable from each element of the
// slice or array v, when its element type can contain one.
func cloneElementSlices(v reflect.Value) {
	if !typeCanContainSlice(v.Type().Elem()) {
		return
	}

	for i := range v.Len() {
		cloneSlicesInPlace(v.Index(i))
	}
}

// cloneMapValueSlices replaces the map v with a copy whose values have their
// reachable slices cloned (map values are not addressable in place).
func cloneMapValueSlices(v reflect.Value) {
	if v.IsNil() || !typeCanContainSlice(v.Type().Elem()) {
		return
	}

	out := reflect.MakeMapWithSize(v.Type(), v.Len())

	iter := v.MapRange()
	for iter.Next() {
		value := reflect.New(iter.Value().Type()).Elem()
		value.Set(iter.Value())
		cloneSlicesInPlace(value)
		out.SetMapIndex(iter.Key(), value)
	}

	v.Set(out)
}

// cloneSlice returns a shallow copy of src with its own backing array. A nil
// slice stays nil so the caller's nil/empty distinction is preserved.
func cloneSlice(src reflect.Value) reflect.Value {
	if src.IsNil() {
		return reflect.Zero(src.Type())
	}

	dst := reflect.MakeSlice(src.Type(), src.Len(), src.Len())
	reflect.Copy(dst, src)

	return dst
}

// makeFieldName joins the non-empty name segments with dots.
func makeFieldName(n ...string) string {
	switch len(n) {
	case 0:
		return ""
	case 1:
		return n[0]
	}

	parts := make([]string, 0, len(n))
	for _, part := range n {
		if part != "" {
			parts = append(parts, part)
		}
	}

	return strings.Join(parts, ".")
}
