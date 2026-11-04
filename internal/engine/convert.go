// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package engine

import (
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"

	"github.com/altessa-s/go-convert/internal/primitive"
	"github.com/altessa-s/go-convert/internal/typeinfo"
	"github.com/altessa-s/go-convert/reflectx"

	convcodec "github.com/altessa-s/go-convert/codec"
)

// noIgnorePath is a sentinel path for values that must never be matched by
// ignore paths (map keys).
const noIgnorePath = "\x00key"

// DefaultMaxDepth is the nesting limit applied when [Config.MaxDepth] is zero.
const DefaultMaxDepth = 10_000

// Config is the immutable configuration of an [Engine]. Name sets and
// mappings hold lower-cased keys.
type Config struct {
	Codecs                *convcodec.Set
	IgnoreFields          map[string]struct{}
	IgnorePaths           map[string]struct{}
	FieldMappings         map[string]string
	IgnoreZeroValues      bool
	IgnoreNilValues       bool
	SparseMerge           bool
	CopySlices            bool
	HandleEmbeddedStructs bool
	InitEmbeddedStructs   bool
	OverflowCheck         bool
	MaxDepth              int
}

// Engine converts values by reflection according to its [Config]. It is
// safe for concurrent use; all per-call state lives in a walker.
type Engine struct {
	cfg   Config
	types *typeinfo.Cache
	// namesObserved reports whether anything reads element and entry names,
	// so building them per element can be skipped otherwise.
	namesObserved bool
	// skipsValues reports whether the nil or zero mode skips values.
	skipsValues bool
	// ignoreAncestors holds the proper ancestors of every ignore path.
	ignoreAncestors map[string]struct{}
	// plans caches the struct conversion plans, which depend on cfg; see
	// [newPlanCache].
	plans     *planCache
	planFlags planFlags
	// root remembers the plan of the first root struct pair.
	root rootPlan
}

// rootPlan states.
const (
	rootEmpty uint32 = iota
	rootWriting
	rootReady
)

// rootPlan remembers the plan of the first root struct pair an engine
// converts. An engine usually converts a single pair, which then skips the
// plan cache lookup. The fields are written once, before state becomes
// rootReady, and only read after observing it.
type rootPlan struct {
	state    atomic.Uint32
	src, dst reflect.Type
	plan     *structPlan
}

// get returns the plan converting src into dst for e, both struct types.
func (r *rootPlan) get(e *Engine, src, dst reflect.Type) *structPlan {
	if r.state.Load() == rootReady && r.src == src && r.dst == dst {
		return r.plan
	}

	plan := e.planFor(src, dst)
	if r.state.CompareAndSwap(rootEmpty, rootWriting) {
		r.src, r.dst, r.plan = src, dst, plan
		r.state.Store(rootReady)
	}

	return plan
}

// noCodecs is the codec set of engines configured without codecs. It is
// never modified.
var noCodecs = convcodec.NewCodecsSet()

// sharedTypes is shared by every engine: struct metadata does not depend on
// the configuration.
var sharedTypes = typeinfo.NewCache()

// New returns an engine for cfg. Nil sets and mappings are treated as empty.
func New(cfg Config) *Engine {
	if cfg.Codecs == nil {
		cfg.Codecs = noCodecs
	}

	if cfg.MaxDepth <= 0 {
		cfg.MaxDepth = DefaultMaxDepth
	}

	plans, flags := newPlanCache(&cfg)

	return &Engine{
		cfg:   cfg,
		types: sharedTypes,
		namesObserved: cfg.Codecs.HasCodecs() || cfg.Codecs.HasErrorCodecs() ||
			len(cfg.IgnoreFields) > 0 || cfg.OverflowCheck,
		skipsValues:     cfg.IgnoreNilValues || cfg.IgnoreZeroValues,
		ignoreAncestors: ignoreAncestors(cfg.IgnorePaths),
		plans:           plans,
		planFlags:       flags,
	}
}

// walker carries the state of one conversion call.
type walker struct {
	*Engine
	depth int
	// terminal is the codec chain terminal for engines without ignore
	// paths, bound once so the chain does not allocate a closure per field.
	terminal convcodec.CodecHandler
	// frames are the idle error codec terminals; see [walker.errorFrame].
	frames []*errorFrame
}

// errorFrame is the terminal of one error codec chain. Its finish handler is
// bound once, so running a chain does not allocate a closure per value.
type errorFrame struct {
	w      *walker
	name   string
	finish convcodec.CodecHandlerE
}

// run supplies the built-in conversion once the error codecs delegate.
func (f *errorFrame) run(path string, src, dst reflect.Value) error {
	f.w.convertAfterCodecs(path, f.name, src, dst)
	return nil
}

// errorFrame takes an idle error codec terminal for the value named name.
// A frame stays taken while its chain runs, as a codec may call the
// terminal more than once, and is lost rather than reused when the chain
// panics.
func (w *walker) errorFrame(name string) *errorFrame {
	var f *errorFrame
	if n := len(w.frames); n > 0 {
		f = w.frames[n-1]
		w.frames = w.frames[:n-1]
	} else {
		f = &errorFrame{w: w}
		f.finish = f.run
	}

	f.name = name

	return f
}

// maxIdleFrames bounds the idle frames a walker keeps, which deep input
// would otherwise grow to its nesting depth.
const maxIdleFrames = 64

// releaseFrame makes f idle again.
func (w *walker) releaseFrame(f *errorFrame) {
	if len(w.frames) < maxIdleFrames {
		f.name = ""
		w.frames = append(w.frames, f)
	}
}

var walkers = sync.Pool{New: func() any { return newWalker() }}

// newWalker returns a walker with its codec terminal bound.
func newWalker() *walker {
	w := &walker{}
	w.terminal = w.convertAfterCodecsNoPath

	return w
}

// walker takes a walker for e from the pool.
func (e *Engine) walker() *walker {
	w, ok := walkers.Get().(*walker)
	if !ok {
		w = newWalker()
	}

	w.Engine = e
	w.depth = 0

	return w
}

// release returns w to the pool.
func (w *walker) release() {
	w.Engine = nil
	walkers.Put(w)
}

// descend enters one nesting level and fails once the limit is exceeded, so
// cyclic or adversarially deep input cannot overflow the goroutine stack.
func (w *walker) descend(path string) {
	w.depth++
	if w.depth > w.cfg.MaxDepth {
		panic(DepthError{Limit: w.cfg.MaxDepth, Path: path})
	}
}

// Convert converts a struct or a slice of structs src into dst, which must be
// a non-nil pointer to the same kind. A nil src leaves dst untouched.
//
// It returns an [UnsupportedTypeError] for invalid combinations, an
// [OverflowError] or [DepthError], or the unchanged error of a codec. dst may
// be partially written when an error is returned.
func (e *Engine) Convert(src, dst any) (err error) {
	srcValue := reflectx.IndirectValue(reflect.ValueOf(src))
	if !srcValue.IsValid() {
		return nil
	}
	defer func() {
		if r := recover(); r != nil {
			err = classify(r, src, dst)
		}
	}()

	dstValue := rootDestination(srcValue.Kind(), dst)

	w := e.walker()
	defer w.release()

	if srcValue.Kind() == reflect.Struct {
		w.convertStruct("", srcValue, dstValue, false, e.root.get(e, srcValue.Type(), dstValue.Type()))
		return nil
	}

	w.convertValueAt("", "", srcValue, dstValue)

	return nil
}

// rootDestination validates the root shapes and returns the settable value
// behind dst, allocating nil intermediate pointers.
func rootDestination(srcKind reflect.Kind, dst any) reflect.Value {
	if srcKind != reflect.Struct && srcKind != reflect.Slice {
		panic("src must be a struct or slice of structs")
	}

	dstValue := reflect.ValueOf(dst)
	if !dstValue.IsValid() || dstValue.Kind() != reflect.Pointer || dstValue.IsNil() {
		panic("dst must be a non-nil pointer to a struct or slice of structs")
	}

	for dstValue.Kind() == reflect.Pointer {
		if dstValue.IsNil() {
			dstValue.Set(reflect.New(dstValue.Type().Elem()))
		}

		dstValue = dstValue.Elem()
	}

	if dstValue.Kind() != srcKind {
		panic("dst must be the same kind as src")
	}

	return dstValue
}

// classify turns a recovered conversion panic into the returned error.
func classify(r, src, dst any) error {
	switch err := r.(type) {
	case codecError:
		return err.error
	case OverflowError:
		return err
	case DepthError:
		return err
	case error:
		return UnsupportedTypeError{Source: src, Destination: dst, Reason: err.Error(), Err: err}
	default:
		return UnsupportedTypeError{Source: src, Destination: dst, Reason: fmt.Sprint(r)}
	}
}

// ConvertValue converts src into the settable dst and reports failures like
// [Engine.Convert].
func (e *Engine) ConvertValue(src, dst reflect.Value) error {
	return e.convertEntry("", src, dst)
}

// ConvertKey converts a map key src into the settable dst. Ignore paths never
// apply to map keys.
func (e *Engine) ConvertKey(src, dst reflect.Value) error {
	return e.convertEntry(noIgnorePath, src, dst)
}

// convertEntry converts one value at path and recovers failures as errors.
func (e *Engine) convertEntry(path string, src, dst reflect.Value) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = classify(r, valueOf(src), valueOf(dst))
		}
	}()

	w := e.walker()
	defer w.release()

	w.convertValueAt(path, "", src, dst)

	return nil
}

// valueOf returns the interface value of v, or nil when v cannot provide one.
func valueOf(v reflect.Value) any {
	if !v.IsValid() || !v.CanInterface() {
		return nil
	}

	return v.Interface()
}

// convertStruct converts one struct to another following plan, or the cached
// plan for the two types when plan is nil.
func (w *walker) convertStruct(path string, src reflect.Value, dst reflect.Value, nested bool, plan *structPlan) {
	dstValue := reflectx.IndirectValue(dst)
	if nested && src.Type() == dstValue.Type() && w.hasIgnorePathBelow(path) && w.overwritesValues() {
		src = w.copyPrivateState(src, dstValue)
	}

	if w.cfg.HandleEmbeddedStructs {
		w.fillEmbeddedDestinations(path, src, dstValue)
	}

	if plan == nil {
		plan = w.planFor(src.Type(), dstValue.Type())
	}

	w.convertStructFields(path, src, dst, dstValue, plan)
}

// overwritesValues reports whether conversion writes every source value,
// i.e. none of the sparse-merge, ignore-zero or ignore-nil modes is active.
func (w *walker) overwritesValues() bool {
	return !w.cfg.SparseMerge && !w.cfg.IgnoreZeroValues && !w.cfg.IgnoreNilValues
}

// copyPrivateState copies src (including unexported fields) into dstValue but
// keeps dstValue's exported fields, which only the mapping-aware field loop
// may write. It returns an addressable copy of src to walk instead.
func (w *walker) copyPrivateState(src, dstValue reflect.Value) reflect.Value {
	snapshot := reflect.New(dstValue.Type()).Elem()
	snapshot.Set(dstValue)

	source := reflect.New(src.Type()).Elem()
	source.Set(src)
	dstValue.Set(source)

	if w.cfg.CopySlices {
		cloneSlicesInPlace(dstValue)
	}
	// Only the mapping-aware loop below may write exported fields.
	for i := range dstValue.NumField() {
		if dstValue.Type().Field(i).IsExported() {
			dstValue.Field(i).Set(snapshot.Field(i))
		}
	}

	return source
}

// fillEmbeddedDestinations converts the whole source into every embedded
// struct of the destination (convert.EmbeddedFill).
func (w *walker) fillEmbeddedDestinations(path string, src, dstValue reflect.Value) {
	dstInfo := w.types.Get(dstValue.Type())
	for fieldName, fieldIndex := range dstInfo.Embedded {
		embedded := dstValue.Field(fieldIndex)
		if embedded.Kind() == reflect.Pointer && embedded.IsNil() {
			w.initEmbeddedPointer(path, fieldName, src, embedded)
			continue
		}

		w.convertValueAt(path, fieldName, src, embedded)
	}
}

// initEmbeddedPointer fills a nil embedded struct pointer with the converted
// source: always with InitEmbeddedStructs (convert.EmbeddedFillAlways, even if the
// result is empty), otherwise only when the converted struct has a non-zero
// value.
func (w *walker) initEmbeddedPointer(path, fieldName string, src, embedded reflect.Value) {
	tmpVal := reflect.New(embedded.Type().Elem())
	w.convertValueAt(path, fieldName, src, tmpVal)

	if w.cfg.InitEmbeddedStructs || !isStructZero(tmpVal.Elem()) {
		embedded.Set(tmpVal)
	}
}

// convertEmbeddedSource converts an embedded source struct either into the
// destination's field of the same name or, when the destination has none,
// flattened into the destination itself.
func (w *walker) convertEmbeddedSource(
	path, srcFieldName string, srcFieldValue, dst, dstFieldValue reflect.Value, dstFieldExists bool,
) {
	fieldPath := w.childPath(path, srcFieldName)
	if w.isIgnoredPath(fieldPath) {
		return
	}

	if !dstFieldExists {
		// A source embedded struct flattened into the destination
		// contributes promoted fields at the parent path.
		w.convertStruct(path, reflectx.IndirectValue(srcFieldValue), dst, true, nil)
		return
	}

	reflectx.MakeDst(&dstFieldValue, reflectx.IndirectType(dstFieldValue.Type()))
	// A destination embedded field keeps the embedded segment.
	w.convertStruct(fieldPath, reflectx.IndirectValue(srcFieldValue), dstFieldValue, true, nil)
}

// skipValue applies the field exclusion and nil/zero options before codecs.
func (w *walker) skipValue(path, fieldName string, srcValue reflect.Value) bool {
	if w.cfg.IgnoreNilValues || w.cfg.IgnoreZeroValues {
		srcValueKind := srcValue.Kind()

		isNilValue := (srcValueKind == reflect.Pointer || srcValueKind == reflect.Slice || srcValueKind == reflect.Map) && srcValue.IsNil()
		if (w.cfg.IgnoreNilValues && isNilValue) || (w.cfg.IgnoreZeroValues && srcValue.IsZero()) {
			return true
		}
	}

	// Honor ignore fields early, including fast paths (assignable/primitive).
	if len(w.cfg.IgnoreFields) > 0 {
		if _, ok := w.cfg.IgnoreFields[fieldName]; ok {
			return true
		}
	}

	if len(w.cfg.IgnorePaths) > 0 && path != "" {
		if _, ok := w.cfg.IgnorePaths[path]; ok {
			return true
		}
	}

	return false
}

// convertValueAt converts one value: skip rules, codecs, then assignment,
// primitives and kind dispatch.
func (w *walker) convertValueAt(path, fieldName string, srcValue reflect.Value, dstValue reflect.Value) {
	if w.skipValue(path, fieldName, srcValue) {
		return
	}

	w.descend(path)
	defer func() { w.depth-- }()

	if path != noIgnorePath && w.cfg.Codecs.HasErrorCodecs() {
		frame := w.errorFrame(fieldName)

		err := w.cfg.Codecs.RunError(path, fieldName, srcValue, dstValue, frame.finish)
		w.releaseFrame(frame)

		if err != nil {
			panic(codecError{err})
		}

		return
	}

	// When codecs are registered, every value goes through the codec chain
	// before any built-in conversion, assignment included, so a codec sees
	// values whose type is assignable to the destination too (a string
	// codec on a string field) and codec-handled struct, slice or map types
	// (time.Time) are not shadowed by the field-by-field copy. The terminal
	// assigns or dispatches on the kind; primitives convert through the
	// convertible path there.
	if w.cfg.Codecs.HasCodecs() {
		if len(w.cfg.IgnorePaths) == 0 && path != noIgnorePath {
			w.cfg.Codecs.Run(fieldName, srcValue, dstValue, w.terminal)
		} else {
			w.cfg.Codecs.Run(fieldName, srcValue, dstValue,
				func(name string, src, dst reflect.Value) {
					w.convertAfterCodecs(path, name, src, dst)
				})
		}

		return
	}

	if w.tryAssignValue(path, srcValue, dstValue) {
		return
	}

	// Without codecs, primitives are converted directly; codecs may
	// intercept primitives, so this shortcut only runs when there are none.
	handled, overflow := primitive.Convert(srcValue, dstValue, w.cfg.OverflowCheck)
	if overflow {
		panic(newOverflowError(fieldName, srcValue, reflectx.IndirectType(dstValue.Type()).Kind()))
	}

	if handled {
		return
	}

	w.convertByKind(path, fieldName, srcValue, dstValue)
}

// convertAfterCodecsNoPath is the cached codec terminal for converters
// without ignore paths; it avoids a per-field closure allocation.
func (w *walker) convertAfterCodecsNoPath(fieldName string, srcValue, dstValue reflect.Value) {
	w.convertAfterCodecs("", fieldName, srcValue, dstValue)
}

// convertByKind performs the built-in conversion dispatched on the source and
// destination reflect kinds (struct, slice, map), falling back to a convertible
// scalar conversion. It is used directly on the codec-free path and as the
// terminal handler of the codec chain, so it must not invoke codecs itself.
func (w *walker) convertByKind(path, fieldName string, srcValue reflect.Value, dstValue reflect.Value) {
	if w.hasIgnorePathBelow(path) && (srcValue.Kind() == reflect.Interface || dstValue.Kind() == reflect.Interface) {
		w.convertDynamic(path, fieldName, srcValue, dstValue, make(map[dynamicIdentity]bool))
		return
	}

	// A nil pointer has no value to walk. Codecs get the first chance in
	// convertValueAt; this guard runs only when they decline.
	if srcValue.Kind() == reflect.Pointer && srcValue.IsNil() {
		if dstValue.CanSet() {
			dstValue.Set(reflect.Zero(dstValue.Type()))
		}

		return
	}

	// Detach before writes even when the original destination aliases the source.
	if w.hasIgnorePathBelow(path) && w.overwritesValues() {
		detachPointer(dstValue)
	}

	srcValueType := reflectx.IndirectType(srcValue.Type())
	dstValueType := reflectx.IndirectType(dstValue.Type())
	srcKind := srcValueType.Kind()
	dstKind := dstValueType.Kind()

	if srcKind == reflect.Struct && w.convertFromStruct(path, fieldName, srcValue, dstValue, srcValueType, dstValueType) {
		return
	}

	switch {
	case srcKind == reflect.Array && dstKind == reflect.Array && srcValue.Len() == dstValue.Len():
		// Same-length arrays; reached when ignored values force a walk.
		w.convertArrays(path, fieldName, srcValue, dstValue)
	case srcKind == reflect.Slice && dstKind == reflect.Slice:
		w.convertSlices(path, fieldName, srcValue, dstValue)
	case srcKind == reflect.Map && dstKind == reflect.Map:
		w.convertMaps(path, fieldName, srcValue, dstValue)
	default:
		w.convertConvertible(fieldName, srcValue, dstValue)
	}
}

// convertFromStruct handles a (pointer to) struct source converted into a
// struct or into a slice of the same struct type. It reports false when the
// destination kind needs another conversion.
func (w *walker) convertFromStruct(path, fieldName string, srcValue, dstValue reflect.Value, srcValueType, dstValueType reflect.Type) bool {
	switch dstValueType.Kind() {
	case reflect.Struct:
		w.convertStructToStruct(path, srcValue, dstValue, dstValueType, nil)
		return true
	case reflect.Slice:
		if reflectx.IndirectType(dstValueType.Elem()) != srcValueType {
			return false
		}

		w.convertStructToSlice(path, fieldName, srcValue, dstValue)

		return true
	default:
		return false
	}
}

// convertStructToStruct converts a (pointer to) struct into a (pointer to)
// struct, allocating a nil destination pointer and applying the sparse-merge
// clear-on-empty rule. A non-nil plan is the plan for the two struct types.
func (w *walker) convertStructToStruct(path string, srcValue, dstValue reflect.Value, dstValueType reflect.Type, plan *structPlan) {
	srcIndirect := reflectx.IndirectValue(srcValue)
	srcIsNil := !srcIndirect.IsValid()
	// Sparse merge: a present-but-empty nested struct pointer
	// clears the destination instead of recursing. Only a pointer
	// carries a presence signal — a non-pointer source struct is
	// always "present", so its zero value merges normally instead of
	// destroying the destination field.
	if w.cfg.SparseMerge && srcValue.Kind() == reflect.Pointer && !srcIsNil &&
		isSparseStructEmpty(srcIndirect) {
		dstValue.Set(reflect.Zero(dstValue.Type()))
		return
	}

	if srcIsNil {
		dstValue.Set(reflect.Zero(dstValue.Type()))
		return
	}

	reflectx.MakeDst(&dstValue, dstValueType)
	w.convertStruct(path, srcIndirect, reflectx.IndirectValue(dstValue), true, plan)
}

// convertStructToSlice converts a struct to a slice containing that struct.
func (w *walker) convertStructToSlice(path, fieldName string, srcValue reflect.Value, dstValue reflect.Value) {
	dstValue.Set(reflect.MakeSlice(reflect.SliceOf(reflectx.IndirectType(dstValue.Type()).Elem()), 0, 1))

	dstVal := reflect.New(reflectx.IndirectType(dstValue.Type().Elem()))
	w.convertValueAt(path, fieldName, srcValue, dstVal.Elem())

	if dstValue.Type().Elem().Kind() != reflect.Pointer {
		dstValue.Set(reflect.Append(dstValue, dstVal.Elem()))
	} else {
		dstValue.Set(reflect.Append(dstValue, dstVal))
	}
}

// convertConvertible performs a direct convertible-type conversion (e.g. between
// defined types that share an underlying type) when no codec or structural
// conversion applies. It honors the overflow-check option for narrowing numeric
// conversions. This is the terminal step of the built-in dispatch and never
// invokes codecs.
func (w *walker) convertConvertible(fieldName string, srcValue reflect.Value, dstValue reflect.Value) {
	// Check convertibility before allocating a destination pointer. Allocating
	// first would leave a non-nil pointer to the zero value when the types are
	// not convertible (for example a struct field mapped to *string), which
	// callers cannot distinguish from an intentionally empty value.
	dstType := reflectx.IndirectType(dstValue.Type())
	if !w.isTypeCachedConvertible(reflectx.IndirectType(srcValue.Type()), dstType) {
		return
	}

	// The type relation alone is not enough: a slice shorter than the
	// destination array is convertible by type but not by value.
	srcIndirect := reflectx.IndirectValue(srcValue)
	if !srcIndirect.IsValid() {
		return
	}

	// Scalar pairs always convert; they are written in place below instead
	// of through an allocated intermediate value.
	scalar := isScalarPair(srcIndirect.Kind(), dstType.Kind())

	var converted reflect.Value

	if !scalar {
		var ok bool
		if converted, ok = reflectx.ConvertValue(srcIndirect, dstType); !ok {
			return
		}
	}

	dst := dstValue
	reflectx.MakeDst(&dst, dstType)
	dst = reflectx.IndirectValue(dst)

	if w.cfg.OverflowCheck && w.overflows(srcIndirect, dst) {
		panic(newOverflowError(fieldName, srcIndirect, dst.Kind()))
	}

	if scalar {
		if srcIndirect.CanInterface() && dst.CanSet() {
			setScalar(srcIndirect, dst)
			return
		}

		// Let Set report an unexported source or an unsettable destination.
		converted = srcIndirect.Convert(dstType)
	}

	dst.Set(converted)
}

// isIgnoredPath reports whether path is listed in WithIgnorePaths.
func (w *walker) isIgnoredPath(path string) bool {
	if path == "" || len(w.cfg.IgnorePaths) == 0 {
		return false
	}

	_, ok := w.cfg.IgnorePaths[path]

	return ok
}

// childPath builds the dotted path only when ignore paths are configured;
// otherwise it avoids a concatenation on every nested field.
func (w *walker) childPath(path, field string) string {
	if path == noIgnorePath {
		return noIgnorePath
	}

	if len(w.cfg.IgnorePaths) == 0 && !w.cfg.Codecs.HasErrorCodecs() {
		return ""
	}

	return joinPath(path, field)
}

// mustDescend reports whether an otherwise assignable value has to be walked
// because an ignore path can match below it. Opaque structs (time.Time) are
// never walked field by field.
func (w *walker) mustDescend(path string, t reflect.Type) bool {
	if !w.hasIgnorePathBelow(path) {
		return false
	}

	return typeHasMergeableStruct(t)
}

// convertAfterCodecs is the built-in conversion ending the codec chains:
// assignment, then kind dispatch.
func (w *walker) convertAfterCodecs(path, name string, src, dst reflect.Value) {
	if !w.tryAssignValue(path, src, dst) {
		w.convertByKind(path, name, src, dst)
	}
}

// tryAssignValue owns direct assignment and optional slice copying. Sparse
// structs and ignored descendants require traversal; opaque structs and nil
// containers retain direct assignment because they have no fields to walk.
func (w *walker) tryAssignValue(path string, src, dst reflect.Value) bool {
	nilContainer := (src.Kind() == reflect.Slice || src.Kind() == reflect.Map) && src.IsNil()
	if !w.isTypeCachedAssignable(src.Type(), dst.Type()) ||
		(dst.Kind() == reflect.Interface && w.hasIgnorePathBelow(path)) ||
		(w.mustDescend(path, src.Type()) && !nilContainer) ||
		(w.cfg.SparseMerge && w.isMergeableStruct(src.Type())) {
		return false
	}

	dst.Set(src)

	if w.cfg.CopySlices {
		if src.Kind() == reflect.Pointer && !src.IsNil() && src.Elem().Kind() == reflect.Slice {
			cp := reflect.New(src.Type().Elem())
			cp.Elem().Set(src.Elem())
			cloneSlicesInPlace(cp.Elem())
			dst.Set(cp)
		} else {
			cloneSlicesInPlace(dst)
		}
	}

	return true
}

// hasIgnorePathBelow reports whether an ignore path lies strictly below path.
func (w *walker) hasIgnorePathBelow(path string) bool {
	if len(w.ignoreAncestors) == 0 || path == noIgnorePath {
		return false
	}

	_, ok := w.ignoreAncestors[path]

	return ok
}

// ignoreAncestors returns every proper ancestor of the ignore paths, the
// root "" included, so [walker.hasIgnorePathBelow] is a single lookup.
func ignoreAncestors(paths map[string]struct{}) map[string]struct{} {
	if len(paths) == 0 {
		return nil
	}

	ancestors := map[string]struct{}{"": {}}

	for path := range paths {
		for i := range len(path) {
			if path[i] == '.' {
				ancestors[path[:i]] = struct{}{}
			}
		}
	}

	return ancestors
}

// detachPointer copies one destination pointer level before a path-aware write.
func detachPointer(v reflect.Value) {
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return
	}

	cp := reflect.New(v.Type().Elem())
	cp.Elem().Set(v.Elem())
	v.Set(cp)
}
