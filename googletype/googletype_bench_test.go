// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package googletype_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/altessa-s/go-convert/googletype"

	"google.golang.org/genproto/googleapis/type/date"
	"google.golang.org/genproto/googleapis/type/latlng"
	"google.golang.org/genproto/googleapis/type/money"
	"google.golang.org/genproto/googleapis/type/timeofday"
)

func benchHandler(string, reflect.Value, reflect.Value) error { return nil }

func BenchmarkNewMoney_ToInt64(b *testing.B) {
	codec := googletype.NewMoney()
	src := reflect.ValueOf(&money.Money{CurrencyCode: "USD", Units: 12, Nanos: 340_000_000}).Elem()

	for b.Loop() {
		var dst int64

		_ = codec("Total", src, reflect.ValueOf(&dst).Elem(), benchHandler)
	}
}

func BenchmarkNewMoney_FromInt64(b *testing.B) {
	codec := googletype.NewMoney(googletype.WithCurrency("USD"))
	src := reflect.ValueOf(int64(1234))

	for b.Loop() {
		var dst money.Money

		_ = codec("Total", src, reflect.ValueOf(&dst).Elem(), benchHandler)
	}
}

func BenchmarkNewMoney_ToString(b *testing.B) {
	codec := googletype.NewMoney()
	src := reflect.ValueOf(&money.Money{CurrencyCode: "USD", Units: 12, Nanos: 340_000_000}).Elem()

	for b.Loop() {
		var dst string

		_ = codec("Total", src, reflect.ValueOf(&dst).Elem(), benchHandler)
	}
}

func BenchmarkNewMoney_FromString(b *testing.B) {
	codec := googletype.NewMoney(googletype.WithCurrency("USD"))
	src := reflect.ValueOf("12.34")

	for b.Loop() {
		var dst money.Money

		_ = codec("Total", src, reflect.ValueOf(&dst).Elem(), benchHandler)
	}
}

func BenchmarkNewDate_ToTime(b *testing.B) {
	codec := googletype.NewDate()
	src := reflect.ValueOf(&date.Date{Year: 2025, Month: 6, Day: 15}).Elem()

	for b.Loop() {
		var dst time.Time

		_ = codec("Day", src, reflect.ValueOf(&dst).Elem(), benchHandler)
	}
}

func BenchmarkNewDate_FromString(b *testing.B) {
	codec := googletype.NewDate()
	src := reflect.ValueOf("2025-06-15")

	for b.Loop() {
		var dst date.Date

		_ = codec("Day", src, reflect.ValueOf(&dst).Elem(), benchHandler)
	}
}

func BenchmarkNewLatLng_ToPair(b *testing.B) {
	codec := googletype.NewLatLng()
	src := reflect.ValueOf(&latlng.LatLng{Latitude: 48.8566, Longitude: 2.3522}).Elem()

	for b.Loop() {
		var dst [2]float64

		_ = codec("Location", src, reflect.ValueOf(&dst).Elem(), benchHandler)
	}
}

func BenchmarkNewTimeOfDay_ToDuration(b *testing.B) {
	codec := googletype.NewTimeOfDay()
	src := reflect.ValueOf(&timeofday.TimeOfDay{Hours: 8, Minutes: 30}).Elem()

	for b.Loop() {
		var dst time.Duration

		_ = codec("Opens", src, reflect.ValueOf(&dst).Elem(), benchHandler)
	}
}

func BenchmarkNewTimeOfDay_FromString(b *testing.B) {
	codec := googletype.NewTimeOfDay()
	src := reflect.ValueOf("08:30:05.25")

	for b.Loop() {
		var dst timeofday.TimeOfDay

		_ = codec("Opens", src, reflect.ValueOf(&dst).Elem(), benchHandler)
	}
}

func BenchmarkNew_PassThrough(b *testing.B) {
	codec := googletype.New()
	src := reflect.ValueOf("hello")

	for b.Loop() {
		var dst string

		_ = codec("Name", src, reflect.ValueOf(&dst).Elem(), benchHandler)
	}
}
