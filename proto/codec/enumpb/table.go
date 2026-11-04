// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package enumpb

import (
	"google.golang.org/protobuf/reflect/protoreflect"

	convcodec "github.com/altessa-s/go-convert/codec"
)

// Table converts between the protobuf enum type P and a Go type G through a
// one-to-one table, both directly ([Table.ToGo], [Table.ToEnum]) and as an
// error codec ([Table.Codec]). Use it where handlers convert single values
// outside converter.ConvertE. A Table is immutable and safe for concurrent
// use.
type Table[P Enum, G comparable] struct {
	c *mappingCodec[P, G]
}

// NewTable creates the Table of table. The table, the options and the
// [ConfigError] are those of [NewMapping]: with [WithUnspecifiedAsEmpty] the
// zero enum number and the zero Go value map to each other unless the table
// lists them.
//
// Example:
//
//	statuses, err := enumpb.NewTable(map[userv1.Status]user.Status{
//		userv1.Status_STATUS_ACTIVE:  user.Active,
//		userv1.Status_STATUS_BLOCKED: user.Blocked,
//	}, enumpb.WithUnspecifiedAsEmpty())
//
//	status, err := statuses.ToGo("user.status", req.GetUser().GetStatus())
func NewTable[P Enum, G comparable](table map[P]G, opts ...Option) (*Table[P, G], error) {
	c, err := newMappingCodec("NewTable", table, opts)
	if err != nil {
		return nil, err
	}

	return &Table[P, G]{c: c}, nil
}

// MustNewTable is [NewTable] for tables known to be valid, such as
// package-level literals; it panics with the [ConfigError] instead of
// returning it.
func MustNewTable[P Enum, G comparable](table map[P]G, opts ...Option) *Table[P, G] {
	t, err := NewTable(table, opts...)
	if err != nil {
		panic(err)
	}

	return t
}

// ToGo returns the Go value mapped from enum. An enum missing from the table
// fails with an [UnknownValueError] on path, the field path reported to the
// caller.
func (t *Table[P, G]) ToGo(path string, enum P) (G, error) {
	if value, ok := t.c.goValues[enum]; ok {
		return value, nil
	}

	var zero G
	if enum == 0 && t.c.opts.unspecifiedAsEmpty {
		return zero, nil
	}

	return zero, UnknownValueError{Enum: t.c.enumName, Path: path, Value: protoreflect.EnumNumber(enum)}
}

// ToEnum returns the enum mapped from value. A value missing from the table
// fails with an [UnknownValueError] on path and returns the zero enum number.
func (t *Table[P, G]) ToEnum(path string, value G) (P, error) {
	if enum, ok := t.c.toEnum[value]; ok {
		return enum, nil
	}
	if value == t.c.zeroGo && t.c.opts.unspecifiedAsEmpty {
		return 0, nil
	}

	return 0, UnknownValueError{Enum: t.c.enumName, Path: path, Value: value}
}

// Codec returns the table as an error codec for converter.WithErrorCodecs,
// equivalent to [NewMapping] with the same table and options.
func (t *Table[P, G]) Codec() convcodec.CodecE {
	return t.c.convert
}
