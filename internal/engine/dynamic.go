// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package engine

import (
	"reflect"
	"strings"

	"github.com/altessa-s/go-convert/internal/typeinfo"
)

// dynamicIdentity guards only the active recursion stack. Repeated siblings
// are copied independently; back edges with ignored descendants leave the
// destination unchanged. Unfiltered back edges are handled by the assignable
// leaf path before the guard and retain the source reference there.
type dynamicIdentity struct {
	typ reflect.Type
	ptr uintptr
}

// convertDynamicValue runs codecs once for a nested field or element, keeping
// the active recursion stack in the terminal rather than starting a new walk.
func (w *walker) convertDynamicValue(path, name string, src, dst reflect.Value, active map[dynamicIdentity]bool) {
	if w.skipValue(path, name, src) {
		return
	}
	// A root dynamic map key can collide with noIgnorePath; treat that value
	// as map-key context so error codecs never receive the internal sentinel.
	if path != noIgnorePath && w.cfg.Codecs.HasErrorCodecs() {
		err := w.cfg.Codecs.RunError(path, name, src, dst, func(p string, s, d reflect.Value) error {
			w.convertDynamic(p, name, s, d, active)
			return nil
		})
		if err != nil {
			panic(codecError{err})
		}

		return
	}

	if !w.cfg.Codecs.HasCodecs() {
		// An empty chain calls its terminal directly; skip the closure.
		w.convertDynamic(path, name, src, dst, active)
		return
	}

	w.cfg.Codecs.Run(name, src, dst, func(n string, s, d reflect.Value) {
		w.convertDynamic(path, n, s, d, active)
	})
}

// convertDynamicLeaf applies built-in conversion without repeating codecs after
// unwrapping a value. The nil/zero options still apply to the concrete value.
func (w *walker) convertDynamicLeaf(path, name string, src, dst reflect.Value) {
	if !w.skipValue(path, name, src) {
		w.convertAfterCodecs(path, name, src, dst)
	}
}

// convertDynamic walks values reached through an interface. Map string keys
// in this traversal are object paths, unlike statically typed map keys.
func (w *walker) convertDynamic(path, name string, src, dst reflect.Value, active map[dynamicIdentity]bool) {
	w.descend(path)
	defer func() { w.depth-- }()

	if w.isIgnoredPath(path) {
		return
	}

	if _, ignored := w.cfg.IgnoreFields[name]; ignored {
		return
	}

	if src.Kind() == reflect.Interface {
		if src.IsNil() {
			if dst.Kind() == reflect.Interface {
				dst.SetZero()
			}

			return
		}

		src = src.Elem()
	}

	if dst.Kind() == reflect.Interface {
		w.convertDynamicIntoInterface(path, name, src, dst, active)
		return
	}

	if src.Type() != dst.Type() || !w.hasIgnorePathBelow(path) {
		w.convertDynamicLeaf(path, name, src, dst)
		return
	}

	switch src.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Slice:
		key, entered := w.enterDynamicReference(src, dst, active)
		if !entered {
			return
		}
		defer delete(active, key)
	default:
		// Only reference kinds need identity tracking to break cycles.
	}

	switch src.Kind() {
	case reflect.Pointer:
		w.convertDynamicPointer(path, name, src, dst, active)
	case reflect.Struct:
		w.convertDynamicStruct(path, src, dst, active)
	case reflect.Slice, reflect.Array:
		w.convertDynamicSequence(path, name, src, dst, active)
	case reflect.Map:
		w.convertDynamicMap(path, name, src, dst, active)
	default:
		w.convertDynamicLeaf(path, name, src, dst)
	}
}

// convertDynamicIntoInterface converts src into a fresh value of its own
// concrete type, seeded from the destination when that holds the same type,
// and stores it in the interface destination when assignable.
func (w *walker) convertDynamicIntoInterface(path, name string, src, dst reflect.Value, active map[dynamicIdentity]bool) {
	out := reflect.New(src.Type()).Elem()
	if !dst.IsNil() && dst.Elem().Type() == src.Type() {
		out.Set(dst.Elem())
	}

	w.convertDynamic(path, name, src, out, active)

	if out.Type().AssignableTo(dst.Type()) {
		dst.Set(out)
	}
}

// enterDynamicReference handles a nil reference and registers a non-nil one
// on the active recursion stack. It reports false when the walk must stop:
// the source is nil or the reference is already being walked (a cycle). The
// caller removes the returned key once the value has been walked.
func (w *walker) enterDynamicReference(src, dst reflect.Value, active map[dynamicIdentity]bool) (dynamicIdentity, bool) {
	if src.IsNil() {
		if !w.cfg.IgnoreNilValues {
			dst.Set(src)
		}

		return dynamicIdentity{}, false
	}

	key := dynamicIdentity{src.Type(), src.Pointer()}
	if active[key] {
		return key, false
	}

	active[key] = true

	return key, true
}

// convertDynamicPointer walks a non-nil pointer whose pointee may contain
// ignored descendants; scalar pointees are converted as a leaf.
func (w *walker) convertDynamicPointer(path, name string, src, dst reflect.Value, active map[dynamicIdentity]bool) {
	if w.cfg.SparseMerge && src.Elem().Kind() == reflect.Struct && isSparseStructEmpty(src.Elem()) {
		dst.SetZero()
		return
	}
	// Keep scalar pointers intact so zero checks apply to their presence,
	// not to the dereferenced value. Composite pointees still need walking.
	switch src.Elem().Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Struct, reflect.Map, reflect.Slice, reflect.Array:
		// These pointees may contain ignored descendants.
	default:
		w.convertDynamicLeaf(path, name, src, dst)
		detachPointer(dst)

		return
	}

	out := reflect.New(src.Type().Elem())
	if !dst.IsNil() {
		out.Elem().Set(dst.Elem())
	}

	w.convertDynamic(path, name, src.Elem(), out.Elem(), active)
	dst.Set(out)
}

// convertDynamicStruct walks a struct of the destination's type field by
// field, honoring field mappings, so ignored descendants keep their values.
func (w *walker) convertDynamicStruct(path string, src, dst reflect.Value, active map[dynamicIdentity]bool) {
	srcInfo := w.types.Get(src.Type())

	dstInfo := w.types.Get(dst.Type())
	if !w.cfg.SparseMerge && !w.cfg.IgnoreZeroValues && !w.cfg.IgnoreNilValues {
		w.copyDynamicPrivateState(src, dst, dstInfo)
	}

	for i := range src.NumField() {
		f := srcInfo.Fields[i]
		if !f.Exported {
			continue
		}

		target, ok := w.mappedFieldIndex(f.Lower, i, dstInfo)
		if !ok {
			continue
		}

		if src.Field(i).Kind() == reflect.Pointer && src.Field(i).IsNil() {
			continue
		}

		w.convertDynamicValue(w.childPath(path, f.Lower), f.Lower, src.Field(i), dst.Field(target), active)
	}
}

// copyDynamicPrivateState copies the source's unexported fields into dst while
// keeping every exported destination field for the mapping-aware walk.
func (w *walker) copyDynamicPrivateState(src, dst reflect.Value, dstInfo *typeinfo.Info) {
	old := reflect.New(dst.Type()).Elem()
	old.Set(dst)
	dst.Set(src)

	if w.cfg.CopySlices {
		cloneSlicesInPlace(dst)
	}
	// Restore all exported fields before applying source mappings.
	for i := range dst.NumField() {
		if dstInfo.Fields[i].Exported {
			dst.Field(i).Set(old.Field(i))
		}
	}
}

// mappedFieldIndex resolves the destination field index for the source field
// at index i. Without a mapping the index is unchanged (same struct type); a
// mapping to a missing destination field reports false.
func (w *walker) mappedFieldIndex(fieldName string, i int, dstInfo *typeinfo.Info) (int, bool) {
	mapped, ok := w.cfg.FieldMappings[fieldName]
	if !ok {
		return i, true
	}

	target, exists := dstInfo.ByName[mapped]

	return target, exists
}

// convertDynamicSequence walks slice and array elements; arrays keep the
// destination elements as the starting point for each slot.
func (w *walker) convertDynamicSequence(path, name string, src, dst reflect.Value, active map[dynamicIdentity]bool) {
	out := reflect.New(src.Type()).Elem()
	if src.Kind() == reflect.Slice {
		out = reflect.MakeSlice(src.Type(), src.Len(), src.Len())
	}

	for i := range src.Len() {
		if src.Kind() == reflect.Array && i < dst.Len() {
			out.Index(i).Set(dst.Index(i))
		}

		w.convertDynamicValue(path, name, src.Index(i), out.Index(i), active)
	}

	dst.Set(out)
}

// convertDynamicMap rebuilds a map, treating string keys as path segments:
// ignored keys keep their destination entries and the rest are walked.
func (w *walker) convertDynamicMap(path, name string, src, dst reflect.Value, active map[dynamicIdentity]bool) {
	out := reflect.MakeMapWithSize(src.Type(), src.Len())

	stringKeys := src.Type().Key().Kind() == reflect.String
	if stringKeys {
		w.keepIgnoredDynamicKeys(path, src, dst, out)
	}

	it := src.MapRange()
	for it.Next() {
		child, childName := path, name
		if stringKeys {
			childName = strings.ToLower(it.Key().String())
			child = w.childPath(path, childName)
		}

		old := dst.MapIndex(it.Key())
		if w.isIgnoredPath(child) {
			if old.IsValid() {
				out.SetMapIndex(it.Key(), old)
			}

			continue
		}

		value := reflect.New(src.Type().Elem()).Elem()
		if old.IsValid() {
			value.Set(old)
		}

		w.convertDynamicValue(child, childName, it.Value(), value, active)
		out.SetMapIndex(it.Key(), value)
	}

	dst.Set(out)
}

// keepIgnoredDynamicKeys preserves ignored destination keys even if the
// source omits them.
func (w *walker) keepIgnoredDynamicKeys(path string, src, dst, out reflect.Value) {
	it := dst.MapRange()
	for it.Next() {
		if src.MapIndex(it.Key()).IsValid() {
			continue
		}

		keyName := strings.ToLower(it.Key().String())
		if w.isIgnoredPath(w.childPath(path, keyName)) {
			out.SetMapIndex(it.Key(), it.Value())
		}
	}
}
