// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package googletype

import (
	"reflect"
	"time"

	"google.golang.org/genproto/googleapis/type/date"

	convcodec "github.com/altessa-s/go-convert/codec"
)

const (
	// maxYear is the largest year a Date holds.
	maxYear = 9999
	// monthsPerYear is the largest month number.
	monthsPerYear = 12
	// leapYear is a leap year, used to bound the day of a Date without a year.
	leapYear = 2000
	// dateLayoutLen is the length of "YYYY-MM-DD".
	dateLayoutLen = len("YYYY-MM-DD")
	// yearDigits is the number of digits in the year of "YYYY-MM-DD".
	yearDigits = 4
	// twoDigits is the width of the month, day, hours, minutes and seconds.
	twoDigits = 2
)

var dateType = reflect.TypeFor[date.Date]()

// NewDate creates a CodecE for date.Date.
//
// Supported conversions:
//   - date.Date -> time.Time: midnight UTC of the date.
//   - time.Time -> date.Date: the calendar date of the time in its own
//     location; the clock is dropped.
//   - date.Date <-> string "YYYY-MM-DD", with a four-digit year. A partial
//     date has no such form: writing one to a string fails with
//     [ErrPartialDate], while [WithAllowPartialDates] still reads strings
//     with zero parts, such as "0000-12-25".
//   - Pointer variants of all of the above.
//
// Years run from 1 to 9999 (0 for a partial date), months from 1 to 12 and
// days up to the length of the month; anything else fails with
// [ErrOutOfRange], and a malformed string with [ErrSyntax]. A Date with a zero
// year, month or day fails with [ErrPartialDate] unless
// [WithAllowPartialDates] is set; even then only the partial dates
// google.type.Date defines are accepted: a year alone, a year and month, or a
// month and day.
//
// Example:
//
//	codec := googletype.NewDate()
//	codec := googletype.NewDate(googletype.WithAllowPartialDates())
func NewDate(opts ...Option) convcodec.CodecE {
	return newCodec(buildOptions(opts), convertDate)
}

// convertDate handles the pairs with date.Date on one side.
func convertDate(p pair, o *options) (bool, error) {
	switch {
	case p.srcType == dateType && (p.dstType == timeType || p.dstType == stringType):
		if !p.src.IsValid() {
			clearNil(p, o)

			return true, nil
		}

		d := messagePtr[date.Date](p.src)

		year, month, day := d.GetYear(), d.GetMonth(), d.GetDay()
		if o.ignoreZero && year == 0 && month == 0 && day == 0 {
			return true, nil
		}

		if err := validateDate(p.path, year, month, day, o.allowPartialDates); err != nil {
			return true, err
		}

		if p.dstType == stringType {
			if year == 0 || month == 0 || day == 0 {
				return true, newError(p.path, ErrPartialDate, "partial date %d-%d-%d has no YYYY-MM-DD form", year, month, day)
			}

			setValue(p.dst, p.dstType, formatDate(year, month, day))

			return true, nil
		}

		setValue(p.dst, p.dstType, time.Date(int(year), time.Month(max(month, 1)), int(max(day, 1)), 0, 0, 0, 0, time.UTC))

		return true, nil
	case p.dstType == dateType && (p.srcType == timeType || p.srcType == stringType):
		if !p.src.IsValid() || o.ignoreZero && p.src.IsZero() {
			if !p.src.IsValid() {
				clearNil(p, o)
			}

			return true, nil
		}

		year, month, day, err := readDate(p, o)
		if err != nil {
			return true, err
		}

		d := messageDst[date.Date](p.dst, p.dstType)
		d.Year, d.Month, d.Day = year, month, day

		return true, nil
	default:
		return false, nil
	}
}

// readDate reads and validates the time.Time or string source of p.
func readDate(p pair, o *options) (year, month, day int32, err error) {
	if p.srcType == stringType {
		s := p.src.String()

		var ok bool
		if year, month, day, ok = parseDate(s); !ok {
			return 0, 0, 0, newError(p.path, ErrSyntax, "%q is not a YYYY-MM-DD date", s)
		}

		return year, month, day, validateDate(p.path, year, month, day, o.allowPartialDates)
	}

	y, m, d := readValue[time.Time](p.src).Date()
	if y < 1 || y > maxYear {
		return 0, 0, 0, newError(p.path, ErrOutOfRange, "year %d outside 1-9999", y)
	}

	return int32(y), int32(m), int32(d), nil
}

// validateDate checks the ranges of a Date and, for a partial date, that it
// is allowed and one of the shapes google.type.Date defines.
func validateDate(path string, year, month, day int32, allowPartial bool) error {
	if year < 0 || year > maxYear || month < 0 || month > monthsPerYear || day < 0 {
		return newError(path, ErrOutOfRange, "date %d-%d-%d", year, month, day)
	}

	if year == 0 || month == 0 || day == 0 {
		if !allowPartial {
			return newError(path, ErrPartialDate, "date %d-%d-%d has a zero part", year, month, day)
		}

		if month == 0 && day != 0 || year == 0 && day == 0 {
			return newError(path, ErrPartialDate, "date %d-%d-%d is not a year, year and month, or month and day", year, month, day)
		}
	}

	if month != 0 && day != 0 {
		boundYear := int(year)
		if year == 0 {
			boundYear = leapYear
		}

		if last := time.Date(boundYear, time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC).Day(); int(day) > last {
			return newError(path, ErrOutOfRange, "day %d outside 1-%d", day, last)
		}
	}

	return nil
}

// formatDate formats a validated date as "YYYY-MM-DD".
func formatDate(year, month, day int32) string {
	var buf [dateLayoutLen]byte

	out := appendPadded(buf[:0], int(year), yearDigits)
	out = append(out, '-')
	out = appendPadded(out, int(month), twoDigits)
	out = append(out, '-')

	return string(appendPadded(out, int(day), twoDigits))
}

// parseDate parses the digits of "YYYY-MM-DD" without checking their ranges.
func parseDate(s string) (year, month, day int32, ok bool) {
	if len(s) != dateLayoutLen || s[4] != '-' || s[7] != '-' {
		return 0, 0, 0, false
	}

	y, okYear := parseDigits(s[:4])
	m, okMonth := parseDigits(s[5:7])
	d, okDay := parseDigits(s[8:])

	return int32(y), int32(m), int32(d), okYear && okMonth && okDay
}
