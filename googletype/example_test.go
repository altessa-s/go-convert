// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package googletype_test

import (
	"errors"
	"fmt"
	"time"

	"github.com/altessa-s/go-convert/googletype"

	"google.golang.org/genproto/googleapis/type/date"
	"google.golang.org/genproto/googleapis/type/money"

	converter "github.com/altessa-s/go-convert"
)

func Example() {
	type Invoice struct {
		Total int64 // minor units
		DueOn time.Time
	}

	type InvoiceMessage struct {
		Total *money.Money
		DueOn *date.Date
	}

	opts := []converter.Option{
		converter.WithErrorCodecs(
			googletype.NewMoney(googletype.WithCurrency("EUR")),
			googletype.NewDate(),
		),
	}

	msg, err := converter.ConvertE(Invoice{Total: 19_990, DueOn: time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)}, &InvoiceMessage{}, opts...)
	if err != nil {
		panic(err)
	}

	fmt.Println(msg.Total.GetCurrencyCode(), msg.Total.GetUnits(), msg.Total.GetNanos())
	fmt.Println(msg.DueOn.GetYear(), msg.DueOn.GetMonth(), msg.DueOn.GetDay())

	_, err = converter.ConvertE(InvoiceMessage{Total: &money.Money{CurrencyCode: "EUR", Units: 1, Nanos: -5}}, &Invoice{}, opts...)
	fmt.Println(errors.Is(err, googletype.ErrNanosSign))
	fmt.Println(err)

	// Output:
	// EUR 199 900000000
	// 2026 1 31
	// true
	// googletype: field "total": units and nanos have opposite signs: units 1, nanos -5
}
