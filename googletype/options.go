// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package googletype

//go:generate optgen generate

// defaultScale is the number of minor-unit digits used for a currency that
// [WithCurrencyScales] does not list, unless [WithScale] sets another one.
const defaultScale = 2

// options holds the configuration shared by every codec of the package. Each
// codec reads only the options that concern its message type.
type options struct {
	// currency ([WithCurrency]) is the ISO 4217 code given to Money values
	// built from int64 minor units or decimal strings, which carry no
	// currency. When set, Money -> int64 and Money -> string also require the
	// source currency to match it, so that the dropped code cannot change
	// silently.
	currency string
	// scale ([WithScale]) is the number of minor-unit digits (0-9) used for
	// currencies that [WithCurrencyScales] does not list; 2 by default, so
	// int64 1234 is 12.34.
	scale int `optgen:"default=defaultScale"`
	// currencyScales ([WithCurrencyScales]) overrides the scale per ISO 4217
	// code, for example {"JPY": 0, "KWD": 3}.
	currencyScales map[string]int
	amountField    string `optgen:"manual"`
	currencyField  string `optgen:"manual"`
	// allowPartialDates ([WithAllowPartialDates]) accepts Date values whose
	// year, month or day is 0. Date -> time.Time replaces a missing month or
	// day with 1 and keeps a missing year as year 0; Date <-> string writes
	// and reads the missing parts as zeros, such as "0000-12-25".
	allowPartialDates bool
	// ignoreZero ([WithIgnoreZero]) skips the conversion, leaving the
	// destination unchanged, when the source is zero: a message with every
	// field zero, the zero time.Time or time.Duration, 0, "" or [0, 0].
	ignoreZero bool
}

// WithMoneyStruct converts Money to and from structs that hold the amount in
// the field named amountField and the ISO 4217 code in the string field named
// currencyField. An integer amount field holds minor units, scaled as for
// int64; a string amount field holds a decimal string. Structs without both
// fields are left to the rest of the codec chain.
func WithMoneyStruct(amountField, currencyField string) Option {
	return func(o *options) {
		o.amountField = amountField
		o.currencyField = currencyField
	}
}
