// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package typeinfo

import (
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
)

const (
	// MaxTypes bounds the number of struct descriptions kept by a [Cache].
	MaxTypes = 1024
	// MaxRelations bounds the number of destination types remembered per
	// source type and per type predicate.
	MaxRelations = 256
)

// Field describes one struct field.
type Field struct {
	// Lower is the lower-cased field name used for case-insensitive matching.
	Lower    string
	Exported bool
	Embedded bool
}

// Info describes a struct type. Non-struct types have no fields.
type Info struct {
	Fields []Field
	// ByName maps the lower-cased name of every exported field to its index.
	ByName map[string]int
	// Embedded maps the lower-cased name of every exported embedded struct
	// field to its index.
	Embedded map[string]int

	assignable  Bounded[bool]
	convertible Bounded[bool]
}

// AssignableTo reports whether src, the type described by info, is
// assignable to dst. Results are memoized per destination type.
func (info *Info) AssignableTo(src, dst reflect.Type) bool {
	if src == dst {
		return true
	}

	return info.assignable.Get(dst, func() bool { return src.AssignableTo(dst) })
}

// ConvertibleTo reports whether src, the type described by info, is
// convertible to dst. Results are memoized per destination type.
func (info *Info) ConvertibleTo(src, dst reflect.Type) bool {
	if src == dst {
		return true
	}

	return info.convertible.Get(dst, func() bool { return src.ConvertibleTo(dst) })
}

// Cache memoizes [Info] per type. It is safe for concurrent use.
type Cache struct {
	infos Bounded[*Info]
}

// NewCache returns an empty cache bounded by [MaxTypes].
func NewCache() *Cache {
	c := &Cache{}
	c.infos.limit = MaxTypes

	return c
}

// Get returns the description of t, building it on first use.
func (c *Cache) Get(t reflect.Type) *Info {
	return c.infos.Get(t, func() *Info { return build(t) })
}

// Len reports the number of cached descriptions.
func (c *Cache) Len() int { return c.infos.Len() }

// build describes the struct behind t; other types get an empty description.
func build(t reflect.Type) *Info {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return &Info{}
	}

	n := t.NumField()

	info := &Info{
		Fields:   make([]Field, n),
		ByName:   make(map[string]int, n),
		Embedded: make(map[string]int),
	}
	for i := range n {
		field := t.Field(i)
		f := Field{
			Lower:    strings.ToLower(field.Name),
			Exported: field.IsExported(),
			Embedded: field.Anonymous,
		}

		info.Fields[i] = f
		if !f.Exported {
			continue
		}

		info.ByName[f.Lower] = i
		if f.Embedded && indirect(field.Type).Kind() == reflect.Struct {
			info.Embedded[f.Lower] = i
		}
	}

	return info
}

// indirect strips all pointer levels from t.
func indirect(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	return t
}

// Bounded is a concurrency-safe memo keyed by type. It is a [Keyed] memo
// whose zero value is ready to use with a limit of [MaxRelations].
type Bounded[V any] = Keyed[reflect.Type, V]

// Keyed is a concurrency-safe memo keyed by any comparable key. Reads are
// lock-free; once the limit is reached, inserting a new entry evicts an
// arbitrary one, so the memo never grows past its limit on programs that
// synthesize types at run time. The zero value is ready to use with a limit
// of [MaxRelations].
type Keyed[K comparable, V any] struct {
	entries sync.Map // K -> V
	size    atomic.Int64
	limit   int
}

// NewKeyed returns an empty memo bounded by limit; a non-positive limit
// selects [MaxRelations].
func NewKeyed[K comparable, V any](limit int) *Keyed[K, V] {
	return &Keyed[K, V]{limit: limit}
}

// Get returns the memoized value for key, computing and storing it on a
// miss. compute may run more than once for the same key under contention.
func (b *Keyed[K, V]) Get(key K, compute func() V) V {
	if v, ok := b.entries.Load(key); ok {
		if value, ok := v.(V); ok {
			return value
		}
	}

	value := compute()

	actual, loaded := b.entries.LoadOrStore(key, value)
	if loaded {
		if stored, ok := actual.(V); ok {
			return stored
		}

		return value
	}

	if b.size.Add(1) > int64(b.max()) {
		b.evictOne(key)
	}

	return value
}

// Len reports the number of memoized entries.
func (b *Keyed[K, V]) Len() int { return int(b.size.Load()) }

// max returns the configured limit or MaxRelations.
func (b *Keyed[K, V]) max() int {
	if b.limit > 0 {
		return b.limit
	}

	return MaxRelations
}

// evictOne removes one entry other than keep.
func (b *Keyed[K, V]) evictOne(keep K) {
	b.entries.Range(func(key, _ any) bool {
		if k, ok := key.(K); ok && k == keep {
			return true
		}

		if _, deleted := b.entries.LoadAndDelete(key); deleted {
			b.size.Add(-1)
			return false
		}

		return true
	})
}
