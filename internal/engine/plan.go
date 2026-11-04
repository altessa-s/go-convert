// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package engine

import (
	"reflect"
	"unsafe"

	"github.com/altessa-s/go-convert/internal/primitive"
	"github.com/altessa-s/go-convert/internal/typeinfo"

	reflectutils "github.com/altessa-s/go-convert/internal/reflect"
)

// stepKind selects how a planned field is converted.
type stepKind uint8

const (
	// stepGeneric converts through [walker.convertValueAt].
	stepGeneric stepKind = iota
	// stepEmbedded converts an embedded source struct through
	// [walker.convertEmbeddedSource].
	stepEmbedded
	// stepAssign assigns the source field to the destination field.
	stepAssign
	// stepPrimitive converts between primitive kinds with a [primitive.Func].
	stepPrimitive
)

// fieldStep is the precomputed conversion of one source field.
type fieldStep struct {
	// name is the lower-cased source field name.
	name string
	src  int
	// dst is the destination field index; -1 when the destination has no
	// such field (embedded sources only).
	dst  int
	kind stepKind
	// srcPointer reports a pointer source field, skipped while nil.
	srcPointer bool
	convert    primitive.Func
}

// structPlan is the field-by-field conversion of one source struct type into
// one destination struct type under the engine's configuration.
type structPlan struct {
	steps []fieldStep
}

// planFlags are the configuration switches a [structPlan] depends on besides
// the ignored field names and the field mappings.
type planFlags uint8

const (
	planIgnorePaths planFlags = 1 << iota
	planErrorCodecs
	planCodecs
	planSparseMerge
	planCopySlices
)

// planKey keys a [structPlan].
type planKey struct {
	src, dst reflect.Type
	flags    planFlags
}

// sharedPlans holds the plans of engines without ignored field names and
// field mappings, whose plans depend on the types and planFlags alone. It
// lets short-lived engines with such configurations reuse their plans.
var sharedPlans = typeinfo.NewKeyed[planKey, *structPlan](typeinfo.MaxTypes)

// newPlanCache returns the plan cache for cfg and the flags keying its plans.
func newPlanCache(cfg *Config) (*typeinfo.Keyed[planKey, *structPlan], planFlags) {
	var flags planFlags

	for _, f := range []struct {
		set  bool
		flag planFlags
	}{
		{len(cfg.IgnorePaths) > 0, planIgnorePaths},
		{cfg.Codecs.HasErrorCodecs(), planErrorCodecs},
		{cfg.Codecs.HasCodecs(), planCodecs},
		{cfg.SparseMerge, planSparseMerge},
		{cfg.CopySlices, planCopySlices},
	} {
		if f.set {
			flags |= f.flag
		}
	}

	if len(cfg.IgnoreFields) == 0 && len(cfg.FieldMappings) == 0 {
		return sharedPlans, flags
	}

	return typeinfo.NewKeyed[planKey, *structPlan](typeinfo.MaxTypes), flags
}

// planFor returns the cached plan converting src into dst, both struct types.
func (e *Engine) planFor(src, dst reflect.Type) *structPlan {
	return e.plans.Get(planKey{src, dst, e.planFlags}, func() *structPlan { return e.buildPlan(src, dst) })
}

// buildPlan precomputes the per-field decisions of [walker.convertStructFields]
// that depend only on the types and the configuration. A step falls back to
// the generic path whenever its outcome would depend on the value or on the
// path at which the struct is reached.
func (e *Engine) buildPlan(src, dst reflect.Type) *structPlan {
	srcInfo := e.types.Get(src)
	dstInfo := e.types.Get(dst)

	steps := make([]fieldStep, 0, len(srcInfo.Fields))
	for i, field := range srcInfo.Fields {
		if !field.Exported {
			continue
		}

		srcType := src.Field(i).Type
		dstIndex, exists := dstInfo.ByName[e.mappedName(field.Lower)]

		step := fieldStep{name: field.Lower, src: i, dst: -1, srcPointer: srcType.Kind() == reflect.Pointer}
		if exists {
			step.dst = dstIndex
		}

		if field.Embedded {
			step.kind = stepEmbedded
			steps = append(steps, step)

			continue
		}

		if !exists {
			continue
		}

		if _, ignored := e.cfg.IgnoreFields[field.Lower]; ignored {
			// skipValue rejects the field name before anything else.
			continue
		}

		step.kind, step.convert = e.fieldStepKind(srcType, dst.Field(dstIndex).Type)
		steps = append(steps, step)
	}

	return &structPlan{steps: steps}
}

// fieldStepKind decides statically how a field of type src converts into a
// field of type dst, mirroring the decision order of [walker.convertValueAt].
func (e *Engine) fieldStepKind(src, dst reflect.Type) (stepKind, primitive.Func) {
	// Ignore paths make skipping and walking depend on the path, and error
	// codecs see every value.
	if len(e.cfg.IgnorePaths) > 0 || e.cfg.Codecs.HasErrorCodecs() {
		return stepGeneric, nil
	}

	if e.types.Get(src).AssignableTo(src, dst) {
		if e.cfg.SparseMerge && isMergeableStructType(e.types, src) {
			return stepGeneric, nil
		}

		if e.cfg.CopySlices && typeCanContainSlice(src) {
			return stepGeneric, nil
		}

		return stepAssign, nil
	}

	if e.cfg.Codecs.HasCodecs() || src.Kind() == reflect.Pointer || dst.Kind() == reflect.Pointer ||
		!reflectutils.IsPrimitive(src.Kind()) {
		return stepGeneric, nil
	}

	if fn, ok := primitive.Lookup(src.Kind(), dst.Kind()); ok {
		return stepPrimitive, fn
	}

	return stepGeneric, nil
}

// mappedName applies the field mappings to a lower-cased source field name.
func (e *Engine) mappedName(name string) string {
	if mapped, ok := e.cfg.FieldMappings[name]; ok {
		return mapped
	}

	return name
}

// isMergeableStructType reports whether t (after pointer indirection) is a
// struct with at least one exported field.
func isMergeableStructType(types *typeinfo.Cache, t reflect.Type) bool {
	if reflectutils.IndirectType(t).Kind() != reflect.Struct {
		return false
	}

	return len(types.Get(t).ByName) > 0
}

// convertStructFields converts every exported, non-nil source field into the
// destination field of the same (or mapped) name following the cached plan.
func (w *walker) convertStructFields(path string, src, dst, dstValue reflect.Value) {
	plan := w.planFor(src.Type(), dstValue.Type())
	for k := range plan.steps {
		step := &plan.steps[k]

		srcField := src.Field(step.src)
		if step.srcPointer && srcField.IsNil() {
			continue
		}

		if step.kind == stepEmbedded {
			var dstField reflect.Value
			if step.dst >= 0 {
				dstField = dstValue.Field(step.dst)
			}

			w.convertEmbeddedSource(path, step.name, srcField, dst, dstField, step.dst >= 0)

			continue
		}

		dstField := dstValue.Field(step.dst)
		if !dstField.CanSet() {
			continue
		}

		childPath := w.childPath(path, step.name)
		switch step.kind {
		case stepAssign:
			if w.enterLeaf(childPath, step.name, srcField) {
				dstField.Set(srcField)
			}
		case stepPrimitive:
			if !srcField.CanAddr() {
				w.convertValueAt(childPath, step.name, srcField, dstField)
				continue
			}

			if w.enterLeaf(childPath, step.name, srcField) {
				w.convertPrimitive(step, srcField, dstField)
			}
		default:
			w.convertValueAt(childPath, step.name, srcField, dstField)
		}
	}
}

// enterLeaf applies the skip rules and the depth limit of
// [walker.convertValueAt] to a planned leaf that does not recurse. It
// reports whether the value must be converted.
func (w *walker) enterLeaf(path, name string, src reflect.Value) bool {
	if (w.cfg.IgnoreNilValues || w.cfg.IgnoreZeroValues) && w.skipValue(path, name, src) {
		return false
	}

	if w.depth+1 > w.cfg.MaxDepth {
		w.descend(path)
	}

	return true
}

// convertPrimitive writes the addressable primitive src into dst with the
// planned conversion function, honoring the overflow check.
func (w *walker) convertPrimitive(step *fieldStep, src, dst reflect.Value) {
	if w.cfg.OverflowCheck && primitive.WouldOverflow(src, dst) {
		panic(newOverflowError(step.name, src, dst.Kind()))
	}

	step.convert(unsafe.Pointer(src.UnsafeAddr()), unsafe.Pointer(dst.UnsafeAddr())) // #nosec G103 -- kinds fixed by the plan
}
