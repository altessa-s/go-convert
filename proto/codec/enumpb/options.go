// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package enumpb

//go:generate optgen generate

// Case selects the letter case of enum value names written to strings.
type Case int

const (
	// CaseKeep keeps names as declared in the .proto file and parses them
	// case-sensitively.
	CaseKeep Case = iota
	// CaseLower writes lower-case names and parses names case-insensitively.
	CaseLower
	// CaseUpper writes upper-case names and parses names case-insensitively.
	CaseUpper
)

// options holds configuration options for enum conversion behavior.
type options struct {
	// trimPrefix ([WithTrimPrefix]) removes a prefix such as "STATUS_" from
	// enum value names, so STATUS_ACTIVE becomes ACTIVE. Names without the
	// prefix, or equal to it, are kept whole.
	trimPrefix string
	// trimEnumPrefix ([WithTrimEnumPrefix]) removes the prefix derived from
	// the enum's own name in UPPER_SNAKE_CASE, such as "FIELD_PRESENCE_" for
	// FieldPresence. It takes precedence over [WithTrimPrefix].
	trimEnumPrefix bool
	// nameCase ([WithCase]) sets the letter case of the names written to
	// strings; [CaseLower] and [CaseUpper] parse names case-insensitively.
	nameCase Case `opt:"Case"`
	// unspecifiedAsEmpty ([WithUnspecifiedAsEmpty]) maps the zero enum
	// number, the *_UNSPECIFIED value by convention, to the zero value of the
	// other side: "" or a nil pointer, and back.
	unspecifiedAsEmpty bool
}
