// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

// Package enumpb provides error codecs for protobuf enums: any generated type
// implementing protoreflect.Enum.
//
// [New] converts enums to and from strings using the value names of the enum
// descriptor. [WithTrimPrefix] and [WithTrimEnumPrefix] strip a common prefix
// such as STATUS_, [WithCase] maps names to lower or upper case, and
// [WithUnspecifiedAsEmpty] maps the zero (*_UNSPECIFIED) value to "" or a nil
// pointer. [NewMapping] converts an enum to and from a Go type, typically a
// hand-written Go enum, through a one-to-one table.
//
// Numbers, names and Go values without a counterpart fail with
// [UnknownValueError], so both codecs are error codecs: register them with
// converter.WithErrorCodecs and call converter.ConvertE, which returns the
// error unchanged. Pointer variants of either side are supported; a nil source
// pointer sets the destination to its zero value (a nil pointer, "" or the
// zero enum or Go value).
//
// Example:
//
//	codec := enumpb.New(enumpb.WithTrimEnumPrefix(), enumpb.WithCase(enumpb.CaseLower))
//	dto, err := converter.ConvertE(msg, &DTO{}, converter.WithErrorCodecs(codec))
package enumpb
