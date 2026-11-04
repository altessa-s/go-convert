// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package tspb_test

import (
	"fmt"
	"time"

	"github.com/altessa-s/go-convert"
	"github.com/altessa-s/go-convert/proto/codec/tspb"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func ExampleNew() {
	type Event struct{ At time.Time }

	type EventPB struct{ At *timestamppb.Timestamp }

	at := time.Date(2026, 1, 2, 3, 4, 5, 6, time.UTC)

	pb := convert.Convert(Event{At: at}, &EventPB{}, convert.WithCodecs(tspb.New()))
	fmt.Println(pb.At.GetSeconds(), pb.At.GetNanos())

	event := convert.Convert(pb, &Event{}, convert.WithCodecs(tspb.New()))
	fmt.Println(event.At.UTC())
	// Output:
	// 1767323045 6
	// 2026-01-02 03:04:05.000000006 +0000 UTC
}

func ExampleWithMilliseconds() {
	type EventPB struct{ At *timestamppb.Timestamp }

	type EventRow struct{ At int64 }

	pb := EventPB{At: timestamppb.New(time.UnixMilli(1767323045123))}

	row := convert.Convert(pb, &EventRow{}, convert.WithCodecs(tspb.New(tspb.WithMilliseconds())))
	fmt.Println(row.At)
	// Output: 1767323045123
}
