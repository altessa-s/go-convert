// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package googletype

import (
	"math"
	"reflect"
	"strconv"
	"strings"

	"github.com/altessa-s/go-convert/reflectx"

	"google.golang.org/genproto/googleapis/type/money"

	convcodec "github.com/altessa-s/go-convert/codec"
)

const (
	// decimalBase is the radix of every decimal representation.
	decimalBase = 10
	// maxFractionDigits is the number of nanos digits in a unit.
	maxFractionDigits = 9
	// maxNanos is the largest magnitude of Money.Nanos.
	maxNanos = 999_999_999
	// currencyCodeLen is the length of an ISO 4217 currency code.
	currencyCodeLen = 3
	// maxInt64Magnitude is the magnitude of math.MinInt64.
	maxInt64Magnitude = uint64(math.MaxInt64) + 1
)

var moneyType = reflect.TypeFor[money.Money]()

// pow10 holds 10^0 through 10^9.
var pow10 = [maxFractionDigits + 1]int64{1, 1e1, 1e2, 1e3, 1e4, 1e5, 1e6, 1e7, 1e8, 1e9}

// NewMoney creates a CodecE for money.Money.
//
// Supported conversions:
//   - money.Money <-> int64 minor units: with scale 2, 1234 is 12.34. The
//     scale comes from [WithCurrencyScales] for the currency, else
//     [WithScale] (2 by default). Money -> int64 fails with [ErrPrecision]
//     when the amount has more fraction digits than the scale, and with
//     [ErrOverflow] when it does not fit int64.
//   - money.Money <-> string decimal such as "12.34" or "-0.5": at least as
//     many fraction digits as the scale, more when the amount has them. The
//     input accepts an optional sign and 1-9 fraction digits, nothing else.
//   - money.Money <-> a struct registered with [WithMoneyStruct], which keeps
//     the currency.
//   - Pointer variants of all of the above.
//
// int64 and string carry no currency: converting them to Money uses
// [WithCurrency] and fails with [ErrInvalidCurrency] when it is not set.
// Every Money source is validated: its currency must be three upper-case
// ASCII letters, |nanos| at most 999,999,999 and units and nanos must not
// have opposite signs ([ErrNanosSign]).
//
// Example:
//
//	codec := googletype.NewMoney(googletype.WithCurrency("USD"))
//	codec := googletype.NewMoney(googletype.WithCurrency("JPY"), googletype.WithScale(0))
func NewMoney(opts ...Option) convcodec.CodecE {
	return newCodec(buildOptions(opts), convertMoney)
}

// convertMoney handles the pairs with money.Money on one side.
func convertMoney(p pair, o *options) (bool, error) {
	switch {
	case p.srcType == moneyType && p.dstType != moneyType:
		amount, currency, ok := o.moneyShape(p.dstType)
		if !ok {
			return false, nil
		}

		if !p.src.IsValid() {
			clearNil(p, o)

			return true, nil
		}

		m := messagePtr[money.Money](p.src)
		if o.ignoreZero && m.GetCurrencyCode() == "" && m.GetUnits() == 0 && m.GetNanos() == 0 {
			return true, nil
		}

		return true, moneyTo(p, m, amount, currency, o)
	case p.dstType == moneyType && p.srcType != moneyType:
		amount, currency, ok := o.moneyShape(p.srcType)
		if !ok {
			return false, nil
		}

		if !p.src.IsValid() || o.ignoreZero && p.src.IsZero() {
			if !p.src.IsValid() {
				clearNil(p, o)
			}

			return true, nil
		}

		return true, moneyFrom(p, amount, currency, o)
	default:
		return false, nil
	}
}

// moneyStructShape is the cached result of [options.moneyShape] for a
// struct type.
type moneyStructShape struct {
	amount, currency []int
	ok               bool
}

// moneyShape reports whether t converts to and from Money. For a struct
// registered with [WithMoneyStruct] it returns the field indexes, resolved
// once per type and cached by the codec; for int64 and string both are nil.
func (o *options) moneyShape(t reflect.Type) (amount, currency []int, ok bool) {
	if t == int64Type || t == stringType {
		return nil, nil, true
	}

	if o.amountField == "" || t.Kind() != reflect.Struct {
		return nil, nil, false
	}

	if o.moneyShapes == nil {
		return o.resolveMoneyShape(t)
	}

	if cached, found := o.moneyShapes.Load(t); found {
		shape, _ := cached.(moneyStructShape)
		return shape.amount, shape.currency, shape.ok
	}

	amount, currency, ok = o.resolveMoneyShape(t)
	o.moneyShapes.Store(t, moneyStructShape{amount: amount, currency: currency, ok: ok})

	return amount, currency, ok
}

// resolveMoneyShape looks up the [WithMoneyStruct] fields of the struct type
// t.
func (o *options) resolveMoneyShape(t reflect.Type) (amount, currency []int, ok bool) {
	amountField, found := t.FieldByName(o.amountField)
	if !found || !amountField.IsExported() || len(amountField.Index) != 1 || !isAmountKind(amountField.Type.Kind()) {
		return nil, nil, false
	}

	currencyField, found := t.FieldByName(o.currencyField)
	if !found || !currencyField.IsExported() || len(currencyField.Index) != 1 || currencyField.Type.Kind() != reflect.String {
		return nil, nil, false
	}

	return amountField.Index, currencyField.Index, true
}

// isAmountKind reports whether a struct amount field of kind k is supported.
func isAmountKind(k reflect.Kind) bool {
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.String:
		return true
	default:
		return false
	}
}

// moneyTo converts the Money m to the destination of p.
func moneyTo(p pair, m *money.Money, amountIdx, currencyIdx []int, o *options) error {
	currency, units, nanos := m.GetCurrencyCode(), m.GetUnits(), m.GetNanos()
	if err := validateMoney(p.path, currency, units, nanos); err != nil {
		return err
	}

	if amountIdx == nil && o.currency != "" && currency != o.currency {
		return newError(p.path, ErrCurrencyMismatch, "money currency %q, want %q", currency, o.currency)
	}

	scale, err := o.scaleFor(p.path, currency)
	if err != nil {
		return err
	}

	switch p.dstType {
	case int64Type:
		minor, err := toMinorUnits(p.path, units, nanos, scale)
		if err != nil {
			return err
		}

		setValue(p.dst, p.dstType, minor)
	case stringType:
		setValue(p.dst, p.dstType, formatDecimal(units, nanos, scale))
	default:
		return moneyToStruct(p, amountIdx[0], currencyIdx[0], currency, units, nanos, scale)
	}

	return nil
}

// moneyFrom converts the source of p to the Money destination.
func moneyFrom(p pair, amountIdx, currencyIdx []int, o *options) error {
	amount, currency := p.src, o.currency
	if amountIdx != nil {
		amount = p.src.Field(amountIdx[0])
		if c := p.src.Field(currencyIdx[0]).String(); c != "" {
			currency = c
		}
	}

	if currency == "" {
		return newError(p.path, ErrInvalidCurrency, "no currency: set one with WithCurrency")
	}

	if !validCurrency(currency) {
		return newError(p.path, ErrInvalidCurrency, "%q is not a three-letter ISO 4217 code", currency)
	}

	scale, err := o.scaleFor(p.path, currency)
	if err != nil {
		return err
	}

	var (
		units int64
		nanos int32
	)

	switch amount.Kind() {
	case reflect.String:
		if units, nanos, err = parseDecimal(p.path, amount.String()); err != nil {
			return err
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if amount.Uint() > math.MaxInt64 {
			return newError(p.path, ErrOverflow, "amount %d overflows int64", amount.Uint())
		}

		units, nanos = fromMinorUnits(int64(amount.Uint()), scale)
	default:
		units, nanos = fromMinorUnits(amount.Int(), scale)
	}

	m := messageDst[money.Money](p.dst, p.dstType)
	m.CurrencyCode, m.Units, m.Nanos = currency, units, nanos

	return nil
}

// moneyToStruct writes a valid amount into the [WithMoneyStruct] fields of
// the destination of p. Nothing is written when the amount does not fit.
func moneyToStruct(p pair, amountIdx, currencyIdx int, currency string, units int64, nanos int32, scale int) error {
	amountType := p.dstType.Field(amountIdx).Type

	var minor int64

	if amountType.Kind() != reflect.String {
		var err error
		if minor, err = toMinorUnits(p.path, units, nanos, scale); err != nil {
			return err
		}

		if !fitsAmount(amountType, minor) {
			return newError(p.path, ErrOverflow, "%d minor units overflow %s", minor, amountType)
		}
	}

	dst := p.dst
	reflectx.MakeDst(&dst, p.dstType)
	amount := dst.Field(amountIdx)

	switch amountType.Kind() {
	case reflect.String:
		amount.SetString(formatDecimal(units, nanos, scale))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		amount.SetUint(uint64(minor))
	default:
		amount.SetInt(minor)
	}

	dst.Field(currencyIdx).SetString(currency)

	return nil
}

// fitsAmount reports whether minor units fit the integer type t.
func fitsAmount(t reflect.Type, minor int64) bool {
	switch t.Kind() {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return minor >= 0 && !reflect.Zero(t).OverflowUint(uint64(minor))
	default:
		return !reflect.Zero(t).OverflowInt(minor)
	}
}

// scaleFor returns the minor-unit scale of currency.
func (o *options) scaleFor(path, currency string) (int, error) {
	scale, ok := o.currencyScales[currency]
	if !ok {
		scale = o.scale
	}

	if scale < 0 || scale > maxFractionDigits {
		return 0, newError(path, ErrInvalidScale, "scale %d for %s", scale, currency)
	}

	return scale, nil
}

// validCurrency reports whether code is three upper-case ASCII letters.
func validCurrency(code string) bool {
	if len(code) != currencyCodeLen {
		return false
	}

	for i := range len(code) {
		if code[i] < 'A' || code[i] > 'Z' {
			return false
		}
	}

	return true
}

// validateMoney checks the invariants of a Money value.
func validateMoney(path, currency string, units int64, nanos int32) error {
	if !validCurrency(currency) {
		return newError(path, ErrInvalidCurrency, "%q is not a three-letter ISO 4217 code", currency)
	}

	if nanos > maxNanos || nanos < -maxNanos {
		return newError(path, ErrOutOfRange, "nanos %d outside ±999,999,999", nanos)
	}

	if units > 0 && nanos < 0 || units < 0 && nanos > 0 {
		return newError(path, ErrNanosSign, "units %d, nanos %d", units, nanos)
	}

	return nil
}

// toMinorUnits converts a valid amount to minor units at scale.
func toMinorUnits(path string, units int64, nanos int32, scale int) (int64, error) {
	divisor := pow10[maxFractionDigits-scale]
	if int64(nanos)%divisor != 0 {
		return 0, newError(path, ErrPrecision, "nanos %d at scale %d", nanos, scale)
	}

	factor, frac := pow10[scale], int64(nanos)/divisor
	if units > math.MaxInt64/factor || units < math.MinInt64/factor {
		return 0, newError(path, ErrOverflow, "units %d at scale %d overflow int64", units, scale)
	}

	minor := units * factor
	if frac > 0 && minor > math.MaxInt64-frac || frac < 0 && minor < math.MinInt64-frac {
		return 0, newError(path, ErrOverflow, "units %d at scale %d overflow int64", units, scale)
	}

	return minor + frac, nil
}

// fromMinorUnits splits minor units at scale into units and nanos.
func fromMinorUnits(minor int64, scale int) (units int64, nanos int32) {
	factor := pow10[scale]
	return minor / factor, int32(minor % factor * pow10[maxFractionDigits-scale])
}

// formatDecimal formats a valid amount with at least scale fraction digits.
func formatDecimal(units int64, nanos int32, scale int) string {
	var buf [32]byte

	out := buf[:0]
	if units < 0 || nanos < 0 {
		out = append(out, '-')
		nanos = -nanos
	}

	magnitude := uint64(units)
	if units < 0 {
		magnitude = -magnitude
	}

	out = strconv.AppendUint(out, magnitude, decimalBase)

	return string(appendFraction(out, nanos, scale))
}

// parseDecimal parses an optionally signed decimal with 0-9 fraction digits.
func parseDecimal(path, s string) (units int64, nanos int32, err error) {
	digits, negative := s, false
	if digits != "" && (digits[0] == '-' || digits[0] == '+') {
		negative = digits[0] == '-'
		digits = digits[1:]
	}

	whole, fraction, hasDot := strings.Cut(digits, ".")
	if _, ok := parseDigits(whole); !ok {
		return 0, 0, newError(path, ErrSyntax, "%q is not a decimal amount", s)
	}

	if hasDot {
		var ok bool
		if nanos, ok = parseFraction(fraction); !ok {
			return 0, 0, newError(path, ErrSyntax, "%q is not a decimal amount", s)
		}
	}

	magnitude, parseErr := strconv.ParseUint(whole, decimalBase, 64)
	if parseErr != nil || magnitude > maxInt64Magnitude || !negative && magnitude > math.MaxInt64 {
		return 0, 0, newError(path, ErrOverflow, "%q overflows int64 units", s)
	}

	if negative {
		return int64(-magnitude), -nanos, nil
	}

	return int64(magnitude), nanos, nil
}
