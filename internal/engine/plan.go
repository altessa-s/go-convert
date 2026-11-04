// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package engine

import (
	"maps"
	"reflect"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/altessa-s/go-convert/internal/primitive"
	"github.com/altessa-s/go-convert/internal/typeinfo"
	"github.com/altessa-s/go-convert/reflectx"
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
	// stepStruct converts between distinct struct types, or pointers to
	// them, through [walker.convertNestedStruct] with the nested plan
	// resolved once.
	stepStruct
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
	// nested is the plan reference of a stepStruct.
	nested *nestedPlan
	// dstType is the destination struct type of a stepStruct.
	dstType reflect.Type
	// leaf is the kind of a stepGeneric reached only because of ignore
	// paths: stepAssign or stepPrimitive (with leafConvert) where the
	// field's path is neither ignored nor above an ignored path, and
	// stepGeneric otherwise. See [walker.isUnfilteredLeaf].
	leaf        stepKind
	leafConvert primitive.Func
	// paths memoizes the field's path when paths are tracked.
	paths *pathMemo
}

// pathMemo remembers the path of a field under the first parent path it was
// reached at. A struct type is usually reached at one path, so its fields
// reuse their paths instead of joining them on every conversion.
type pathMemo struct {
	first atomic.Pointer[pathEntry]
}

// pathEntry is a parent path and the field path below it.
type pathEntry struct {
	parent, path string
}

// join returns the path of the field name below parent.
func (m *pathMemo) join(parent, name string) string {
	if entry := m.first.Load(); entry != nil && entry.parent == parent {
		return entry.path
	}

	path := joinPath(parent, name)
	if m.first.Load() == nil {
		m.first.CompareAndSwap(nil, &pathEntry{parent: parent, path: path})
	}

	return path
}

// nestedPlan lazily resolves the plan of a nested struct conversion. Plans
// are resolved on first use rather than when the parent plan is built, so
// recursive types do not recurse at build time.
type nestedPlan struct {
	src, dst reflect.Type
	plan     atomic.Pointer[structPlan]
}

// get returns the plan converting n.src into n.dst under e's configuration.
// Every engine sharing the parent plan derives the same nested plan, so
// whichever resolves it first may store it.
func (n *nestedPlan) get(e *Engine) *structPlan {
	if plan := n.plan.Load(); plan != nil {
		return plan
	}

	plan := e.planFor(n.src, n.dst)
	n.plan.Store(plan)

	return plan
}

// structPlan is the field-by-field conversion of one source struct type into
// one destination struct type under the engine's configuration.
type structPlan struct {
	steps []fieldStep
	// copyAll reports that the plan assigns every field of a struct type to
	// the same field of that type and skips none for being nil, so with
	// the nil and zero modes off it amounts to assigning the whole struct.
	copyAll bool
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

// planCache caches struct plans by source type and then, in a [planList], by
// destination type and flags. A single type hashes far faster than a
// composite key, whose hash visits every field.
type planCache = typeinfo.Keyed[reflect.Type, *planList]

// maxPlansPerSource bounds the plans a [planList] keeps.
const maxPlansPerSource = typeinfo.MaxRelations

// planList holds the plans of one source type. Lookups read an immutable
// snapshot without locking; additions copy it under mu.
type planList struct {
	entries atomic.Pointer[[]planEntry]
	mu      sync.Mutex
}

// planEntry is one cached plan of a [planList].
type planEntry struct {
	dst   reflect.Type
	flags planFlags
	plan  *structPlan
}

// newPlanList returns an empty plan list.
func newPlanList() *planList { return &planList{} }

// find returns the cached plan for dst and flags, or nil.
func (l *planList) find(dst reflect.Type, flags planFlags) *structPlan {
	entries := l.entries.Load()
	if entries == nil {
		return nil
	}

	for i := range *entries {
		if entry := &(*entries)[i]; entry.dst == dst && entry.flags == flags {
			return entry.plan
		}
	}

	return nil
}

// get returns the plan for dst and flags, building it on a miss. build may
// run more than once for the same key under contention; the first stored
// plan wins. Once full, the list drops its oldest plan.
func (l *planList) get(dst reflect.Type, flags planFlags, build func() *structPlan) *structPlan {
	if plan := l.find(dst, flags); plan != nil {
		return plan
	}

	plan := build()

	l.mu.Lock()
	defer l.mu.Unlock()

	if stored := l.find(dst, flags); stored != nil {
		return stored
	}

	var old []planEntry
	if entries := l.entries.Load(); entries != nil {
		old = *entries
	}

	if len(old) >= maxPlansPerSource {
		old = old[1:]
	}

	entries := make([]planEntry, len(old), len(old)+1)
	copy(entries, old)
	entries = append(entries, planEntry{dst: dst, flags: flags, plan: plan})
	l.entries.Store(&entries)

	return plan
}

// sharedPlans holds the plans of engines without ignored field names and
// field mappings, whose plans depend on the types and planFlags alone. It
// lets short-lived engines with such configurations reuse their plans.
var sharedPlans = typeinfo.NewKeyed[reflect.Type, *planList](typeinfo.MaxTypes)

// maxConfigPlanCaches bounds the plan caches kept for configurations with
// ignored field names or field mappings.
const maxConfigPlanCaches = 64

// configPlans holds the plan caches of configurations with ignored field
// names or field mappings, keyed by [planCacheKey]. Engines built from equal
// configurations, such as the one-off engines of package-level calls with
// the same options, share their plans instead of rebuilding them.
var configPlans = typeinfo.NewKeyed[string, *planCache](maxConfigPlanCaches)

// planCacheKey encodes the ignored field names and the field mappings of cfg,
// the configuration plans depend on besides planFlags, canonically: sorted
// and length-prefixed, so distinct configurations never share a key.
func planCacheKey(cfg *Config) string {
	key := make([]byte, 0, 64)
	key = appendKeyPart(key, 'f', len(cfg.IgnoreFields))

	for _, field := range slices.Sorted(maps.Keys(cfg.IgnoreFields)) {
		key = appendKeyPart(key, 's', len(field))
		key = append(key, field...)
	}

	key = appendKeyPart(key, 'm', len(cfg.FieldMappings))

	for _, source := range slices.Sorted(maps.Keys(cfg.FieldMappings)) {
		destination := cfg.FieldMappings[source]
		key = appendKeyPart(key, 's', len(source))
		key = append(key, source...)
		key = appendKeyPart(key, 'd', len(destination))
		key = append(key, destination...)
	}

	return string(key)
}

// appendKeyPart appends a tagged length to a [planCacheKey].
func appendKeyPart(key []byte, tag byte, n int) []byte {
	key = append(key, tag)
	key = strconv.AppendInt(key, int64(n), 10)

	return append(key, ':')
}

// newPlanCache returns the plan cache for cfg and the flags keying its plans.
func newPlanCache(cfg *Config) (*planCache, planFlags) {
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

	return configPlans.Get(planCacheKey(cfg), newConfigPlanCache), flags
}

// newConfigPlanCache returns an empty plan cache for [configPlans].
func newConfigPlanCache() *planCache {
	return typeinfo.NewKeyed[reflect.Type, *planList](typeinfo.MaxTypes)
}

// planFor returns the cached plan converting src into dst, both struct types.
func (e *Engine) planFor(src, dst reflect.Type) *structPlan {
	return e.plans.Get(src, newPlanList).get(dst, e.planFlags, func() *structPlan { return e.buildPlan(src, dst) })
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

		if len(e.cfg.IgnorePaths) > 0 || e.cfg.Codecs.HasErrorCodecs() {
			step.paths = &pathMemo{}
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

		dstType := dst.Field(dstIndex).Type

		step.kind, step.convert = e.fieldStepKind(srcType, dstType)
		if step.kind == stepGeneric && len(e.cfg.IgnorePaths) > 0 && !e.cfg.Codecs.HasErrorCodecs() {
			if leaf, convert := e.staticStepKind(srcType, dstType); leaf == stepAssign || leaf == stepPrimitive {
				step.leaf, step.leafConvert = leaf, convert
			}
		}

		if step.kind == stepStruct {
			step.dstType = reflectx.IndirectType(dstType)
			step.nested = &nestedPlan{src: reflectx.IndirectType(srcType), dst: step.dstType}
		}

		steps = append(steps, step)
	}

	return &structPlan{steps: steps, copyAll: src == dst && assignsEveryField(steps, srcInfo)}
}

// assignsEveryField reports whether steps assign each field described by
// info, all exported and none a pointer, to the field of the same index.
func assignsEveryField(steps []fieldStep, info *typeinfo.Info) bool {
	if len(steps) == 0 || len(steps) != len(info.Fields) {
		return false
	}

	for i := range steps {
		if step := &steps[i]; step.kind != stepAssign || step.srcPointer || step.src != i || step.dst != i {
			return false
		}
	}

	return true
}

// fieldStepKind decides statically how a field of type src converts into a
// field of type dst, mirroring the decision order of [walker.convertValueAt].
func (e *Engine) fieldStepKind(src, dst reflect.Type) (stepKind, primitive.Func) {
	// Ignore paths make skipping and walking depend on the path, and error
	// codecs see every value.
	if len(e.cfg.IgnorePaths) > 0 || e.cfg.Codecs.HasErrorCodecs() {
		return stepGeneric, nil
	}

	return e.staticStepKind(src, dst)
}

// staticStepKind is [Engine.fieldStepKind] for a field whose conversion
// does not depend on its path.
func (e *Engine) staticStepKind(src, dst reflect.Type) (stepKind, primitive.Func) {
	// Codecs see every field, assignable ones included.
	if e.cfg.Codecs.HasCodecs() {
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

	if isStructPair(src, dst) {
		return stepStruct, nil
	}

	if src.Kind() == reflect.Pointer || dst.Kind() == reflect.Pointer || !reflectx.IsPrimitive(src.Kind()) {
		return stepGeneric, nil
	}

	if fn, ok := primitive.Lookup(src.Kind(), dst.Kind()); ok {
		return stepPrimitive, fn
	}

	return stepGeneric, nil
}

// isStructPair reports whether src and dst are both structs or pointers to
// structs.
func isStructPair(src, dst reflect.Type) bool {
	return reflectx.IndirectType(src).Kind() == reflect.Struct && reflectx.IndirectType(dst).Kind() == reflect.Struct
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
	if reflectx.IndirectType(t).Kind() != reflect.Struct {
		return false
	}

	return len(types.Get(t).ByName) > 0
}

// convertStructFields converts every exported, non-nil source field into the
// destination field of the same (or mapped) name following plan.
func (w *walker) convertStructFields(path string, src, dst, dstValue reflect.Value, plan *structPlan) {
	if w.copyWhole(path, src, dstValue, plan) {
		return
	}

	// Child paths are only built when tracked; see [walker.childPath].
	tracked := path != noIgnorePath && (len(w.cfg.IgnorePaths) > 0 || w.cfg.Codecs.HasErrorCodecs())

	untrackedPath := ""
	if path == noIgnorePath {
		untrackedPath = noIgnorePath
	}

	for k := range plan.steps {
		step := &plan.steps[k]

		srcField := src.Field(step.src)
		if step.srcPointer && srcField.IsNil() {
			continue
		}

		if step.kind == stepEmbedded {
			w.convertEmbeddedStep(path, step, srcField, dst, dstValue)
			continue
		}

		dstField := dstValue.Field(step.dst)
		if !dstField.CanSet() {
			continue
		}

		childPath := untrackedPath
		if tracked {
			childPath = step.paths.join(path, step.name)
		}

		kind, convert := step.kind, step.convert
		if step.leaf != stepGeneric && w.isUnfilteredLeaf(childPath) {
			kind, convert = step.leaf, step.leafConvert
		}

		switch kind {
		case stepAssign:
			if w.enterLeaf(childPath, step.name, srcField) {
				dstField.Set(srcField)
			}
		case stepPrimitive:
			if !srcField.CanAddr() && !srcField.CanInterface() {
				// Let the generic path report the unexported value.
				w.convertValueAt(childPath, step.name, srcField, dstField)
				continue
			}

			if w.enterLeaf(childPath, step.name, srcField) {
				w.convertPrimitive(step.name, convert, srcField, dstField)
			}
		case stepStruct:
			w.convertNestedStruct(childPath, step.name, srcField, dstField, step.dstType, step.nested.get(w.Engine))
		default:
			w.convertValueAt(childPath, step.name, srcField, dstField)
		}
	}
}

// convertNestedStruct converts the (pointer to) struct src into the (pointer
// to) struct dst of struct type dstType following plan. It is
// [walker.convertValueAt] reduced to the steps that apply when the types are
// not assignable and neither codecs nor ignore paths are configured.
func (w *walker) convertNestedStruct(path, name string, src, dst reflect.Value, dstType reflect.Type, plan *structPlan) {
	if w.skipValue(path, name, src) {
		return
	}

	// A panic aborts the whole conversion, as no codec can recover it, so
	// the level is left without a deferred call.
	w.descend(path)
	w.convertStructToStruct(path, src, dst, dstType, plan)
	w.depth--
}

// convertEmbeddedStep converts the embedded source field srcField of step
// through [walker.convertEmbeddedSource].
func (w *walker) convertEmbeddedStep(path string, step *fieldStep, srcField, dst, dstValue reflect.Value) {
	var dstField reflect.Value
	if step.dst >= 0 {
		dstField = dstValue.Field(step.dst)
	}

	w.convertEmbeddedSource(path, step.name, srcField, dst, dstField, step.dst >= 0)
}

// copyWhole assigns src to dstValue as a whole when plan allows it (see
// [structPlan.copyAll]) and reports whether it did.
func (w *walker) copyWhole(path string, src, dstValue reflect.Value, plan *structPlan) bool {
	if !plan.copyAll || w.cfg.IgnoreNilValues || w.cfg.IgnoreZeroValues || !src.CanInterface() || !dstValue.CanSet() {
		return false
	}

	// Every field is a leaf at the same depth, so the first one would
	// exceed the limit if any did.
	if w.depth+1 > w.cfg.MaxDepth {
		w.descend(w.childPath(path, plan.steps[0].name))
	}

	dstValue.Set(src)

	return true
}

// isUnfilteredLeaf reports whether no ignore path matches path or lies below
// it, so a value at path converts as without ignore paths.
func (w *walker) isUnfilteredLeaf(path string) bool {
	if _, ignored := w.cfg.IgnorePaths[path]; ignored {
		return false
	}

	return !w.hasIgnorePathBelow(path)
}

// enterLeaf applies the skip rules and the depth limit of
// [walker.convertValueAt] to a planned leaf that does not recurse. It
// reports whether the value must be converted.
func (w *walker) enterLeaf(path, name string, src reflect.Value) bool {
	if w.skipsValues && w.skipValue(path, name, src) {
		return false
	}

	if w.depth >= w.cfg.MaxDepth {
		w.descend(path)
	}

	return true
}

// convertPrimitive writes the primitive src of the field name into dst,
// honoring the overflow check. An addressable src is converted with the
// planned conversion function convert. Any other src is converted like
// [reflect.Value.Convert], which is what the generic path,
// [walker.convertConvertible], does for it.
func (w *walker) convertPrimitive(name string, convert primitive.Func, src, dst reflect.Value) {
	if w.cfg.OverflowCheck && primitive.WouldOverflow(src, dst) {
		panic(newOverflowError(name, src, dst.Kind()))
	}

	if !src.CanAddr() {
		setScalar(src, dst)
		return
	}

	convert(unsafe.Pointer(src.UnsafeAddr()), unsafe.Pointer(dst.UnsafeAddr())) // #nosec G103 -- kinds fixed by the plan
}
