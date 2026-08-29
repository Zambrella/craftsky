package business

import (
	"errors"
	"strings"

	"golang.org/x/text/currency"
)

var ErrInvalidPrice = errors.New("business: invalid price")

type Price struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

func CurrencyMinorUnits(code string) (int, bool) {
	if len(code) != 3 || code != strings.ToUpper(code) || code == "XXX" || code == "XTS" {
		return 0, false
	}
	unit, err := currency.ParseISO(code)
	if err != nil {
		return 0, false
	}
	scale, _ := currency.Standard.Rounding(unit)
	return scale, true
}

func ValidatePrice(price Price) error {
	scale, ok := CurrencyMinorUnits(price.Currency)
	if !ok {
		return ErrInvalidPrice
	}
	integer, fraction, hasFraction := strings.Cut(price.Amount, ".")
	if !validPriceInteger(integer) {
		return ErrInvalidPrice
	}
	if !hasFraction {
		return nil
	}
	if scale == 0 || len(fraction) == 0 || len(fraction) > scale || fraction[len(fraction)-1] == '0' || !allASCIIDigits(fraction) {
		return ErrInvalidPrice
	}
	return nil
}

func HydrateIndependentPrice(price *Price) *Price {
	if price == nil || ValidatePrice(*price) != nil {
		return nil
	}
	hydrated := *price
	return &hydrated
}

func validPriceInteger(value string) bool {
	if value == "0" {
		return true
	}
	return len(value) >= 1 && len(value) <= 12 && value[0] >= '1' && value[0] <= '9' && allASCIIDigits(value)
}

func allASCIIDigits(value string) bool {
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}
