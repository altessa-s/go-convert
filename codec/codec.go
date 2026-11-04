// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package codec

import (
	"reflect"
	"sync"
)

// Codec is a middleware-style conversion function. It receives the current
// field name, source and destination [reflect.Value] objects, and a next
// [CodecHandler] that represents the rest of the chain. A Codec should either
// handle the conversion (set dst) and return, or delegate to next.
//
// Example:
//
//	codec := func(fieldName string, src, dst reflect.Value, next CodecHandler) {
//	    if shouldHandle(src, dst) {
//	        dst.Set(transformValue(src))
//	    } else {
//	        next(fieldName, src, dst)
//	    }
//	}
type Codec func(fieldName string, src, dst reflect.Value, next CodecHandler)

// CodecHandler is the terminal callback in a codec chain. It performs the
// default conversion logic when no [Codec] in the chain handles the field.
type CodecHandler func(fieldName string, src, dst reflect.Value)

// Set represents an ordered collection of codecs that form a conversion pipeline.
// Codecs execute in registration order; each codec can handle the conversion or
// delegate to the next via the CodecHandler callback. Not safe for concurrent
// modification after construction; concurrent [Set.Run] calls are safe.
type Set struct {
	customCodecs []Codec
	errorCodecs  []CodecE
}

// NewCodecsSet creates and returns a new codec Set initialized with the provided codecs.
// The codecs will be executed in the order they are provided when the Set is used for conversions.
//
// Example:
//
//	codecSet := NewCodecsSet(customCodec1, customCodec2, customCodec3)
func NewCodecsSet(c ...Codec) *Set {
	return (&Set{}).Add(c...)
}

// Add appends one or more codecs to the existing codec Set.
// The newly added codecs will be executed after the existing ones in the conversion chain.
// Returns the Set itself to enable method chaining.
//
// Example:
//
//	codecSet.Add(codec1).Add(codec2, codec3)
func (c *Set) Add(codecs ...Codec) *Set {
	c.customCodecs = append(c.customCodecs, codecs...)
	return c
}

// Run executes the codec chain for the given field and values.
// It processes all codecs in order, with each having the opportunity to handle the conversion.
// The finishHandler is called at the end if no codec handles the conversion.
//
// Example:
//
//	codecSet.Run("name", srcValue, dstValue, defaultHandler)
func (c *Set) Run(fieldName string, src, dst reflect.Value, finishHandler CodecHandler) {
	switch len(c.customCodecs) {
	case 0:
		finishHandler(fieldName, src, dst)
	case 1:
		c.customCodecs[0](fieldName, src, dst, finishHandler)
	default:
		ctx, ok := codecCtxPool.Get().(*codecContext)
		if !ok {
			ctx = &codecContext{}
			ctx.nextFn = ctx.next
		}

		ctx.codecs = c.customCodecs
		ctx.final = finishHandler
		ctx.index = 1

		c.customCodecs[0](fieldName, src, dst, ctx.nextFn)

		ctx.codecs = nil
		ctx.final = nil
		codecCtxPool.Put(ctx)
	}
}

// HasCodecs reports whether the Set contains any codecs registered with
// [Set.Add].
func (c *Set) HasCodecs() bool {
	return len(c.customCodecs) > 0
}

// codecContext is a pooled execution context for multi-codec chains.
// Each context pre-binds its next method as a CodecHandler to avoid
// allocating a new closure on every Run call.
type codecContext struct {
	codecs []Codec
	final  CodecHandler
	index  int
	nextFn CodecHandler // pre-bound method value, allocated once per pool entry
}

// codecCtxPool reuses codecContext instances across Run calls.
var codecCtxPool = sync.Pool{
	New: func() any {
		ctx := &codecContext{}
		ctx.nextFn = ctx.next

		return ctx
	},
}

// next advances the chain to the next codec, or calls the final handler
// when all codecs have been visited.
func (ctx *codecContext) next(fieldName string, src, dst reflect.Value) {
	i := ctx.index
	if i >= len(ctx.codecs) {
		ctx.final(fieldName, src, dst)
		return
	}

	ctx.index = i + 1
	ctx.codecs[i](fieldName, src, dst, ctx.nextFn)
}

// CodecE is an error-returning codec receiving the full source field path.
// It handles the conversion or delegates to next, propagating its error.
type CodecE func(fieldPath string, src, dst reflect.Value, next CodecHandlerE) error

// CodecHandlerE is the error-returning continuation of a CodecE chain.
type CodecHandlerE func(fieldPath string, src, dst reflect.Value) error

// AddError appends error codecs in execution order and returns the Set.
func (c *Set) AddError(codecs ...CodecE) *Set {
	c.errorCodecs = append(c.errorCodecs, codecs...)
	return c
}

// HasErrorCodecs reports whether any error codecs are registered.
func (c *Set) HasErrorCodecs() bool { return len(c.errorCodecs) > 0 }

// RunError runs error codecs, then the legacy Codec chain, then finishHandler.
// Error codecs and finishHandler receive the full path; legacy codecs receive
// the leaf field name. An error stops the chain and is returned unchanged.
func (c *Set) RunError(fieldPath, fieldName string, src, dst reflect.Value, finishHandler CodecHandlerE) error {
	ctx, ok := errorCtxPool.Get().(*errorContext)
	if !ok {
		ctx = newErrorContext()
	}

	ctx.set, ctx.fieldName, ctx.finish, ctx.index = c, fieldName, finishHandler, 0
	err := ctx.next(fieldPath, src, dst)
	*ctx = errorContext{nextFn: ctx.nextFn, legacyFn: ctx.legacyFn}
	errorCtxPool.Put(ctx)

	return err
}

// errorContext is a pooled execution context for error codec chains. Its
// continuations are bound once, so running a chain does not allocate.
type errorContext struct {
	set       *Set
	fieldName string
	finish    CodecHandlerE
	index     int
	path      string
	err       error
	nextFn    CodecHandlerE
	legacyFn  CodecHandler
}

var errorCtxPool = sync.Pool{New: func() any { return newErrorContext() }}

// newErrorContext returns a context with its continuations bound.
func newErrorContext() *errorContext {
	ctx := &errorContext{}
	ctx.nextFn = ctx.next
	ctx.legacyFn = ctx.legacy

	return ctx
}

// next runs the error codec at the current index, or the legacy chain once
// all error codecs have run. The index is restored afterwards so a codec
// may call next more than once.
func (ctx *errorContext) next(path string, src, dst reflect.Value) error {
	i := ctx.index
	if i < len(ctx.set.errorCodecs) {
		ctx.index = i + 1
		err := ctx.set.errorCodecs[i](path, src, dst, ctx.nextFn)
		ctx.index = i

		return err
	}

	prevPath, prevErr := ctx.path, ctx.err
	ctx.path, ctx.err = path, nil
	ctx.set.Run(ctx.fieldName, src, dst, ctx.legacyFn)
	err := ctx.err
	ctx.path, ctx.err = prevPath, prevErr

	return err
}

// legacy is the terminal of the legacy chain: it runs finish with the full
// path the error codecs settled on.
func (ctx *errorContext) legacy(_ string, src, dst reflect.Value) {
	ctx.err = ctx.finish(ctx.path, src, dst)
}
