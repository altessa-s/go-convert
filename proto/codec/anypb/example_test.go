// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package anypb_test

import (
	"errors"
	"fmt"
	"reflect"
	"time"

	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/altessa-s/go-convert"
	"github.com/altessa-s/go-convert/proto/codec/anypb"
	"github.com/altessa-s/go-convert/proto/codec/tspb"

	pbany "google.golang.org/protobuf/types/known/anypb"
)

func ExampleNew() {
	type Audit struct {
		At     time.Time
		Detail any
	}

	type AuditMessage struct {
		At     *pbany.Any
		Detail *pbany.Any
	}

	// time.Time packs as a google.protobuf.Timestamp; tspb converts the
	// message and the domain value for the codec.
	registry := anypb.NewRegistry().MustRegister(&timestamppb.Timestamp{}, reflect.TypeFor[time.Time]())
	opts := []convert.Option{
		convert.WithErrorCodecs(anypb.New(anypb.WithRegistry(registry))),
		convert.WithCodecs(tspb.New()),
	}

	audit := Audit{At: time.Date(2026, time.March, 14, 9, 0, 0, 0, time.UTC), Detail: wrapperspb.String("login")}
	msg, err := convert.ConvertE(audit, &AuditMessage{}, opts...)
	fmt.Println(err, msg.At.GetTypeUrl(), msg.Detail.GetTypeUrl())

	back, err := convert.ConvertE(msg, &Audit{}, opts...)
	fmt.Println(err, back.At.UTC().Format(time.RFC3339), back.Detail.(*wrapperspb.StringValue).GetValue())

	var into struct{ Detail *durationpb.Duration }

	_, err = convert.ConvertE(msg, &into, opts...)

	var mismatch anypb.TypeMismatchError
	fmt.Println(errors.As(err, &mismatch), mismatch.Want)
	// Output:
	// <nil> type.googleapis.com/google.protobuf.Timestamp type.googleapis.com/google.protobuf.StringValue
	// <nil> 2026-03-14T09:00:00Z login
	// true google.protobuf.Duration
}
