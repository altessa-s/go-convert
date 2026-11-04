// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package googletype_test

import (
	"testing"

	"github.com/altessa-s/go-convert/googletype"

	"google.golang.org/genproto/googleapis/type/date"
	"google.golang.org/genproto/googleapis/type/dayofweek"
)

func BenchmarkDateFromProto(b *testing.B) {
	d := &date.Date{Year: 2026, Month: 10, Day: 18}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := googletype.DateFromProto(d); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWeekdaysFromProto(b *testing.B) {
	days := []dayofweek.DayOfWeek{dayofweek.DayOfWeek_MONDAY, dayofweek.DayOfWeek_WEDNESDAY, dayofweek.DayOfWeek_FRIDAY}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := googletype.WeekdaysFromProto(days); err != nil {
			b.Fatal(err)
		}
	}
}
