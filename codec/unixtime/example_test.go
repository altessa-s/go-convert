// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package unixtime_test

import (
	"fmt"
	"time"

	"github.com/altessa-s/go-convert"
	"github.com/altessa-s/go-convert/codec/unixtime"
)

func ExampleNew() {
	type Event struct{ At time.Time }

	type EventRow struct{ At int64 }

	at := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	row := convert.Convert(Event{At: at}, &EventRow{}, convert.WithCodecs(unixtime.New()))
	fmt.Println(row.At)

	event := convert.Convert(EventRow{At: row.At}, &Event{}, convert.WithCodecs(unixtime.New()))
	fmt.Println(event.At.UTC())

	ms := convert.Convert(Event{At: at}, &EventRow{}, convert.WithCodecs(unixtime.New(unixtime.WithMilliseconds())))
	fmt.Println(ms.At)
	// Output:
	// 1767323045
	// 2026-01-02 03:04:05 +0000 UTC
	// 1767323045000
}
