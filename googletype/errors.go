// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package googletype

import "fmt"

// ErrorKind classifies an [Error]. It implements error, so a kind can be
// matched with errors.Is.
type ErrorKind int

const (
	// ErrOverflow reports a value that does not fit the destination type.
	ErrOverflow ErrorKind = iota + 1
	// ErrPrecision reports a Money amount with more fraction digits than the
	// currency scale allows in minor units.
	ErrPrecision
	// ErrInvalidCurrency reports a currency code that is not three ASCII
	// upper-case letters, or a missing [WithCurrency] where one is needed.
	ErrInvalidCurrency
	// ErrCurrencyMismatch reports a Money whose currency differs from the one
	// set with [WithCurrency] when the conversion drops the currency.
	ErrCurrencyMismatch
	// ErrNanosSign reports a Money whose units and nanos have opposite signs.
	ErrNanosSign
	// ErrInvalidScale reports a scale outside 0-9.
	ErrInvalidScale
	// ErrPartialDate reports a Date with a zero year, month or day without
	// [WithAllowPartialDates].
	ErrPartialDate
	// ErrOutOfRange reports a field outside its valid range, such as month 13,
	// a latitude above 90 or nanos above 999,999,999.
	ErrOutOfRange
	// ErrSyntax reports a string that does not match the expected format.
	ErrSyntax
	// ErrImpossibleDate reports a complete google.type.Date that is not a
	// calendar date, such as February 30.
	ErrImpossibleDate
	// ErrUnknownWeekday reports a DayOfWeek that is unspecified or out of
	// range.
	ErrUnknownWeekday
	// ErrInvalidDuration reports an invalid google.protobuf.Duration, such as
	// seconds and nanos of opposite signs. A valid duration beyond the
	// time.Duration range is not an error: it saturates.
	ErrInvalidDuration
)

// Error returns the description of the kind.
func (k ErrorKind) Error() string {
	switch k {
	case ErrOverflow:
		return "value overflows the destination"
	case ErrPrecision:
		return "amount has more digits than the currency scale"
	case ErrInvalidCurrency:
		return "invalid currency code"
	case ErrCurrencyMismatch:
		return "currency mismatch"
	case ErrNanosSign:
		return "units and nanos have opposite signs"
	case ErrInvalidScale:
		return "scale out of range 0-9"
	case ErrPartialDate:
		return "partial date"
	case ErrOutOfRange:
		return "value out of range"
	case ErrSyntax:
		return "invalid syntax"
	case ErrImpossibleDate:
		return "impossible date"
	case ErrUnknownWeekday:
		return "unknown or unspecified day of the week"
	case ErrInvalidDuration:
		return "invalid duration"
	default:
		return fmt.Sprintf("googletype error kind %d", int(k))
	}
}

// Error reports a google.type value that cannot be converted.
type Error struct {
	// Path is the dotted source field path, if known.
	Path string
	// Kind classifies the failure; Unwrap returns it.
	Kind ErrorKind
	// Detail describes the offending value.
	Detail string
}

// Error reports the field path, the kind of failure and its detail.
func (e Error) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("googletype: field %q: %s: %s", e.Path, e.Kind.Error(), e.Detail)
	}

	return fmt.Sprintf("googletype: %s: %s", e.Kind.Error(), e.Detail)
}

// Unwrap returns the [ErrorKind] of the error.
func (e Error) Unwrap() error { return e.Kind }

// newError builds an [Error] with a formatted detail.
func newError(path string, kind ErrorKind, format string, args ...any) error {
	return Error{Path: path, Kind: kind, Detail: fmt.Sprintf(format, args...)}
}
