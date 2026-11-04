// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package fieldmaskpb

import (
	"reflect"
	"slices"

	convcodec "github.com/altessa-s/go-convert/codec"
	reflectutils "github.com/altessa-s/go-convert/internal/reflect"
	fmpb "google.golang.org/protobuf/types/known/fieldmaskpb"
)

var (
	maskType   = reflect.TypeFor[fmpb.FieldMask]()
	stringType = reflect.TypeFor[string]()
	pathsIndex = func() int {
		f, ok := maskType.FieldByName("Paths")
		if !ok {
			panic("fieldmaskpb: fieldmaskpb.FieldMask has no Paths field")
		}

		return f.Index[0]
	}()
)

// New creates a Codec for bidirectional conversion between
// fieldmaskpb.FieldMask and a slice of any string kind holding its paths.
//
// The paths are copied, never aliased. A nil source (a nil FieldMask pointer
// or a nil slice) sets the destination to its zero value, a nil pointer or
// slice, unless [WithIgnoreEmpty] is set; a FieldMask without paths converts
// to a nil slice (a nil pointer for a slice pointer destination) and an empty
// non-nil slice to a FieldMask without paths. Pointer variants of either side
// are supported.
//
// Example:
//
//	codec := fieldmaskpb.New(fieldmaskpb.WithNormalize())
func New(opts ...Option) convcodec.Codec {
	o := newOptions(opts...)

	return func(fieldName string, src, dst reflect.Value, next convcodec.CodecHandler) {
		if handled, _ := tryConvert(src, dst, o, nil); handled {
			return
		}

		next(fieldName, src, dst)
	}
}

// NewE creates an error codec for bidirectional conversion between
// fieldmaskpb.FieldMask and a slice of any string kind holding its paths,
// converting as [New] does but validating every path first. Register it with
// converter.WithErrorCodecs.
//
// Without [WithMessage] a path must be one or more lower snake_case segments
// separated by dots, such as "address.city"; "*" is accepted as the only
// path. With [WithMessage] every path must instead select a field of the
// message, following the rules of [IgnorePaths]. An invalid path fails with a
// [PathError] and leaves the destination unchanged. Other field pairs are
// passed to next unchanged.
//
// Example:
//
//	codec := fieldmaskpb.NewE(fieldmaskpb.WithMessage(&userpb.User{}))
//	_, err := converter.ConvertE(src, &dst, converter.WithErrorCodecs(codec))
func NewE(opts ...Option) convcodec.CodecE {
	o := newOptions(opts...)
	check := validator(o.message)

	return func(fieldPath string, src, dst reflect.Value, next convcodec.CodecHandlerE) error {
		handled, err := tryConvert(src, dst, o, check)
		if handled || err != nil {
			return err
		}

		return next(fieldPath, src, dst)
	}
}

// tryConvert reports whether it handled the conversion, including skipping
// it. check, when not nil, validates the source paths before anything is
// written.
func tryConvert(src, dst reflect.Value, opts *options, check func([]string) error) (bool, error) {
	srcType := reflectutils.IndirectType(src.Type())
	dstType := reflectutils.IndirectType(dst.Type())

	switch {
	case srcType == maskType && isStringSlice(dstType):
		return convertMaskToPaths(reflectutils.IndirectValue(src), dst, dstType, opts, check)
	case isStringSlice(srcType) && dstType == maskType:
		return convertPathsToMask(reflectutils.IndirectValue(src), dst, dstType, opts, check)
	default:
		return false, nil
	}
}

// convertMaskToPaths handles FieldMask -> []string conversion.
func convertMaskToPaths(srcValue, dst reflect.Value, dstType reflect.Type, opts *options, check func([]string) error) (bool, error) {
	if !srcValue.IsValid() {
		clearDst(dst, opts)
		return true, nil
	}

	paths, ok := reflect.TypeAssert[[]string](srcValue.Field(pathsIndex))
	if !ok {
		return false, nil
	}

	if check != nil {
		if err := check(paths); err != nil {
			return true, err
		}
	}

	paths = normalize(slices.Clone(paths), opts)
	if opts.ignoreEmpty && len(paths) == 0 {
		return true, nil
	}

	if paths == nil {
		// Nothing to write: leave pointer destinations nil.
		clearDst(dst, opts)
		return true, nil
	}

	reflectutils.MakeDst(&dst, dstType)

	if dstType.Elem() == stringType {
		dst.Set(reflect.ValueOf(paths).Convert(dstType))
		return true, nil
	}

	out := reflect.MakeSlice(dstType, len(paths), len(paths))
	for i, path := range paths {
		out.Index(i).SetString(path)
	}

	dst.Set(out)

	return true, nil
}

// convertPathsToMask handles []string -> FieldMask conversion.
func convertPathsToMask(srcValue, dst reflect.Value, dstType reflect.Type, opts *options, check func([]string) error) (bool, error) {
	if !srcValue.IsValid() || srcValue.IsNil() {
		clearDst(dst, opts)
		return true, nil
	}

	if opts.ignoreEmpty && srcValue.Len() == 0 {
		return true, nil
	}

	paths := make([]string, srcValue.Len())
	for i := range paths {
		paths[i] = srcValue.Index(i).String()
	}

	if check != nil {
		if err := check(paths); err != nil {
			return true, err
		}
	}

	reflectutils.MakeDst(&dst, dstType)
	dst.Field(pathsIndex).Set(reflect.ValueOf(normalize(paths, opts)))

	return true, nil
}

// clearDst sets dst to its zero value for a source without paths, unless
// [WithIgnoreEmpty] keeps the destination.
func clearDst(dst reflect.Value, opts *options) {
	if !opts.ignoreEmpty && dst.CanSet() {
		dst.SetZero()
	}
}

// normalize applies [WithNormalize] to paths, which the caller owns.
func normalize(paths []string, opts *options) []string {
	if !opts.normalize || len(paths) == 0 {
		return paths
	}

	mask := &fmpb.FieldMask{Paths: paths}
	mask.Normalize()

	return mask.GetPaths()
}

// isStringSlice reports whether t is a slice of any string kind.
func isStringSlice(t reflect.Type) bool {
	return t.Kind() == reflect.Slice && t.Elem().Kind() == reflect.String
}
