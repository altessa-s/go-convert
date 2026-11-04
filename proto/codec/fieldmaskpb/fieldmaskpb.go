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
// or a nil slice) leaves the destination unchanged; a FieldMask without paths
// converts to a nil slice and an empty non-nil slice to a FieldMask without
// paths. Pointer variants of either side are supported.
//
// Example:
//
//	codec := fieldmaskpb.New(fieldmaskpb.WithNormalize())
func New(opts ...Option) convcodec.Codec {
	o := newOptions(opts...)

	return func(fieldName string, src, dst reflect.Value, next convcodec.CodecHandler) {
		if tryConvert(src, dst, o) {
			return
		}

		next(fieldName, src, dst)
	}
}

// tryConvert reports whether it handled the conversion, including skipping it.
func tryConvert(src, dst reflect.Value, opts *options) bool {
	srcType := reflectutils.IndirectType(src.Type())
	dstType := reflectutils.IndirectType(dst.Type())

	switch {
	case srcType == maskType && isStringSlice(dstType):
		return convertMaskToPaths(reflectutils.IndirectValue(src), dst, dstType, opts)
	case isStringSlice(srcType) && dstType == maskType:
		return convertPathsToMask(reflectutils.IndirectValue(src), dst, dstType, opts)
	default:
		return false
	}
}

// convertMaskToPaths handles FieldMask -> []string conversion.
func convertMaskToPaths(srcValue, dst reflect.Value, dstType reflect.Type, opts *options) bool {
	if !srcValue.IsValid() {
		return true
	}

	paths, ok := reflect.TypeAssert[[]string](srcValue.Field(pathsIndex))
	if !ok {
		return false
	}
	paths = normalize(slices.Clone(paths), opts)
	if opts.ignoreEmpty && len(paths) == 0 {
		return true
	}

	reflectutils.MakeDst(&dst, dstType)
	if paths == nil {
		dst.Set(reflect.Zero(dstType))
		return true
	}
	if dstType.Elem() == stringType {
		dst.Set(reflect.ValueOf(paths).Convert(dstType))
		return true
	}
	out := reflect.MakeSlice(dstType, len(paths), len(paths))
	for i, path := range paths {
		out.Index(i).SetString(path)
	}
	dst.Set(out)
	return true
}

// convertPathsToMask handles []string -> FieldMask conversion.
func convertPathsToMask(srcValue, dst reflect.Value, dstType reflect.Type, opts *options) bool {
	if !srcValue.IsValid() || srcValue.IsNil() {
		return true
	}
	if opts.ignoreEmpty && srcValue.Len() == 0 {
		return true
	}

	paths := make([]string, srcValue.Len())
	for i := range paths {
		paths[i] = srcValue.Index(i).String()
	}

	reflectutils.MakeDst(&dst, dstType)
	dst.Field(pathsIndex).Set(reflect.ValueOf(normalize(paths, opts)))
	return true
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
