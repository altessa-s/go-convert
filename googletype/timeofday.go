// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package googletype

import (
	"reflect"
	"time"

	"google.golang.org/genproto/googleapis/type/timeofday"

	convcodec "github.com/altessa-s/go-convert/codec"
)

const (
	// hoursPerDay is the hour of the end-of-day value "24:00:00".
	hoursPerDay = 24
	// maxMinuteOrSecond is the largest minute or second.
	maxMinuteOrSecond = 59
	// clockLayoutLen is the length of "HH:MM:SS".
	clockLayoutLen = len("HH:MM:SS")
	// endOfDay is the duration of the end-of-day value "24:00:00".
	endOfDay = hoursPerDay * time.Hour
)

var timeOfDayType = reflect.TypeFor[timeofday.TimeOfDay]()

// NewTimeOfDay creates a CodecE for timeofday.TimeOfDay.
//
// Supported conversions:
//   - timeofday.TimeOfDay <-> time.Duration since midnight, from 0 to 24h.
//   - timeofday.TimeOfDay <-> string "HH:MM:SS" followed, when the nanos are
//     not zero, by a dot and 1-9 fraction digits with trailing zeros removed,
//     such as "08:30:00.25". Parsing accepts 1-9 fraction digits.
//   - Pointer variants of all of the above.
//
// Hours run from 0 to 23, minutes and seconds from 0 to 59 and nanos from 0
// to 999,999,999. "24:00:00", the end of the day, is accepted and maps to
// 24h. Leap seconds (second 60) are rejected. Anything else fails with
// [ErrOutOfRange], and a malformed string with [ErrSyntax].
//
// Example:
//
//	codec := googletype.NewTimeOfDay()
func NewTimeOfDay(opts ...Option) convcodec.CodecE {
	return newCodec(buildOptions(opts), convertTimeOfDay)
}

// clock is the content of a TimeOfDay.
type clock struct {
	hours, minutes, seconds, nanos int32
}

// convertTimeOfDay handles the pairs with timeofday.TimeOfDay on one side.
func convertTimeOfDay(p pair, o *options) (bool, error) {
	switch {
	case p.srcType == timeOfDayType && (p.dstType == durationType || p.dstType == stringType):
		if !p.src.IsValid() {
			clearNil(p, o)

			return true, nil
		}

		t := messagePtr[timeofday.TimeOfDay](p.src)

		c := clock{t.GetHours(), t.GetMinutes(), t.GetSeconds(), t.GetNanos()}
		if o.ignoreZero && c == (clock{}) {
			return true, nil
		}

		if err := validateClock(p.path, c); err != nil {
			return true, err
		}

		if p.dstType == stringType {
			setValue(p.dst, p.dstType, formatClock(c))
			return true, nil
		}

		setValue(p.dst, p.dstType, time.Duration(c.hours)*time.Hour+time.Duration(c.minutes)*time.Minute+
			time.Duration(c.seconds)*time.Second+time.Duration(c.nanos))

		return true, nil
	case p.dstType == timeOfDayType && (p.srcType == durationType || p.srcType == stringType):
		if !p.src.IsValid() || o.ignoreZero && p.src.IsZero() {
			if !p.src.IsValid() {
				clearNil(p, o)
			}

			return true, nil
		}

		c, err := readClock(p)
		if err != nil {
			return true, err
		}

		t := messageDst[timeofday.TimeOfDay](p.dst, p.dstType)
		t.Hours, t.Minutes, t.Seconds, t.Nanos = c.hours, c.minutes, c.seconds, c.nanos

		return true, nil
	default:
		return false, nil
	}
}

// readClock reads and validates the time.Duration or string source of p.
func readClock(p pair) (clock, error) {
	if p.srcType == stringType {
		s := p.src.String()

		c, ok := parseClock(s)
		if !ok {
			return clock{}, newError(p.path, ErrSyntax, "%q is not an HH:MM:SS[.fffffffff] time of day", s)
		}

		return c, validateClock(p.path, c)
	}

	d := readValue[time.Duration](p.src)
	if d < 0 || d > endOfDay {
		return clock{}, newError(p.path, ErrOutOfRange, "duration %v outside 0-24h", d)
	}

	return clock{
		hours:   int32(d / time.Hour),
		minutes: int32(d % time.Hour / time.Minute),
		seconds: int32(d % time.Minute / time.Second),
		nanos:   int32(d % time.Second),
	}, nil
}

// validateClock checks the ranges of a TimeOfDay.
func validateClock(path string, c clock) error {
	if c.hours < 0 || c.hours > hoursPerDay || c.minutes < 0 || c.minutes > maxMinuteOrSecond ||
		c.seconds < 0 || c.seconds > maxMinuteOrSecond || c.nanos < 0 || c.nanos > maxNanos ||
		c.hours == hoursPerDay && (c.minutes != 0 || c.seconds != 0 || c.nanos != 0) {
		return newError(path, ErrOutOfRange, "time of day %d:%d:%d.%09d", c.hours, c.minutes, c.seconds, c.nanos)
	}

	return nil
}

// formatClock formats a validated TimeOfDay as "HH:MM:SS[.fffffffff]".
func formatClock(c clock) string {
	var buf [clockLayoutLen + 1 + maxFractionDigits]byte

	out := appendPadded(buf[:0], int(c.hours), twoDigits)
	out = append(out, ':')
	out = appendPadded(out, int(c.minutes), twoDigits)
	out = append(out, ':')
	out = appendPadded(out, int(c.seconds), twoDigits)

	return string(appendFraction(out, c.nanos, 0))
}

// parseClock parses "HH:MM:SS[.fffffffff]" without checking the ranges.
func parseClock(s string) (clock, bool) {
	if len(s) < clockLayoutLen || s[2] != ':' || s[5] != ':' {
		return clock{}, false
	}

	hours, okHours := parseDigits(s[:2])
	minutes, okMinutes := parseDigits(s[3:5])

	seconds, okSeconds := parseDigits(s[6:8])
	if !okHours || !okMinutes || !okSeconds {
		return clock{}, false
	}

	c := clock{hours: int32(hours), minutes: int32(minutes), seconds: int32(seconds)}

	if rest := s[clockLayoutLen:]; rest != "" {
		if rest[0] != '.' {
			return clock{}, false
		}

		nanos, ok := parseFraction(rest[1:])
		if !ok {
			return clock{}, false
		}

		c.nanos = nanos
	}

	return c, true
}
