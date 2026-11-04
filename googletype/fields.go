// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package googletype

import (
	"errors"
	"fmt"
	"time"

	"github.com/altessa-s/go-atlas-core/time/civil"

	"google.golang.org/genproto/googleapis/type/date"
	"google.golang.org/genproto/googleapis/type/dayofweek"
	"google.golang.org/protobuf/types/known/durationpb"

	coreerrs "github.com/altessa-s/go-atlas-core/errors"
)

// Field-error reasons of the decoders that take an API field path. They are
// part of the API contract: a client sees them in the BadRequest violation.
const (
	ReasonImpossibleDate  = "impossible date"
	ReasonInvalidDuration = "invalid duration"
	ReasonUnknownWeekday  = "unknown or unspecified day of the week"
)

// DateFromProtoAt is [DateFromProto] for the API field path field: an
// impossible or partial date is a [coreerrs.FieldError] on field with the
// reason "impossible date YYYY-MM-DD", keeping [ErrImpossibleDate] in its
// chain.
func DateFromProtoAt(field string, d *date.Date) (civil.Date, error) {
	cd, err := DateFromProto(d)
	if err != nil {
		reason := fmt.Sprintf("%s %04d-%02d-%02d", ReasonImpossibleDate, d.GetYear(), d.GetMonth(), d.GetDay())
		return civil.Date{}, &coreerrs.FieldError{Field: field, Reason: reason, Err: err}
	}
	return cd, nil
}

// DurationFromProtoAt is [DurationFromProto] for the API field path field: an
// invalid duration is a [coreerrs.FieldError] on field ("invalid duration"),
// keeping [ErrInvalidDuration] in its chain.
func DurationFromProtoAt(field string, d *durationpb.Duration) (time.Duration, error) {
	v, err := DurationFromProto(d)
	if err != nil {
		return 0, &coreerrs.FieldError{Field: field, Reason: ReasonInvalidDuration, Err: err}
	}
	return v, nil
}

// WeekdaysFromProtoAt is [WeekdaysFromProto] for the repeated API field
// field: an UNSPECIFIED or undefined element is a [coreerrs.FieldError] on
// "field[i]", keeping the [*IndexError] and [ErrUnknownWeekday] in its chain.
func WeekdaysFromProtoAt(field string, days []dayofweek.DayOfWeek) ([]time.Weekday, error) {
	out, err := WeekdaysFromProto(days)
	if err != nil {
		path := field
		if ie, ok := errors.AsType[*IndexError](err); ok {
			path = fmt.Sprintf("%s[%d]", field, ie.Index)
		}
		return nil, &coreerrs.FieldError{Field: path, Reason: ReasonUnknownWeekday, Err: err}
	}
	return out, nil
}
