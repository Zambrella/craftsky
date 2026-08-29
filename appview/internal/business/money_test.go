package business

import (
	"errors"
	"testing"
)

func TestMoneyValidation(t *testing.T) {
	for code, wantScale := range map[string]int{
		"USD": 2,
		"JPY": 0,
		"KWD": 3,
		"CLF": 4,
	} {
		gotScale, ok := CurrencyMinorUnits(code)
		if !ok || gotScale != wantScale {
			t.Errorf("CurrencyMinorUnits(%q) = (%d, %t), want (%d, true)", code, gotScale, ok, wantScale)
		}
	}
	for _, unsupported := range []string{"usd", "ZZZ", "BGN", "XAU"} {
		if _, ok := CurrencyMinorUnits(unsupported); ok {
			t.Errorf("CurrencyMinorUnits(%q) unexpectedly supported", unsupported)
		}
	}

	valid := []Price{
		{Amount: "0", Currency: "USD"},
		{Amount: "1", Currency: "USD"},
		{Amount: "1.2", Currency: "USD"},
		{Amount: "1.23", Currency: "USD"},
		{Amount: "999999999999", Currency: "USD"},
		{Amount: "1", Currency: "JPY"},
		{Amount: "1.234", Currency: "KWD"},
		{Amount: "1.2345", Currency: "CLF"},
	}
	for _, price := range valid {
		if err := ValidatePrice(price); err != nil {
			t.Errorf("ValidatePrice(%+v): %v", price, err)
		}
	}

	invalid := []Price{
		{Amount: "1.20", Currency: "USD"},
		{Amount: "1.0", Currency: "USD"},
		{Amount: "1.1", Currency: "JPY"},
		{Amount: "01", Currency: "USD"},
		{Amount: "-1", Currency: "USD"},
		{Amount: "+1", Currency: "USD"},
		{Amount: "1e2", Currency: "USD"},
		{Amount: "1,000", Currency: "USD"},
		{Amount: "1000000000000", Currency: "USD"},
		{Amount: "1.234", Currency: "USD"},
		{Amount: "1.2345", Currency: "KWD"},
		{Amount: "1", Currency: "usd"},
		{Amount: "1", Currency: "ZZZ"},
		{Amount: "1", Currency: "BGN"},
		{Amount: "1", Currency: "XAU"},
	}
	for _, price := range invalid {
		if err := ValidatePrice(price); !errors.Is(err, ErrInvalidPrice) {
			t.Errorf("ValidatePrice(%+v) error = %v, want ErrInvalidPrice", price, err)
		}
	}
}

func TestIndependentMoneyHydration(t *testing.T) {
	valid := &Price{Amount: "1.2", Currency: "USD"}
	if got := HydrateIndependentPrice(valid); got == nil || *got != *valid {
		t.Fatalf("HydrateIndependentPrice(valid) = %+v, want %+v", got, valid)
	}

	for _, unsupported := range []*Price{
		{Amount: "1.20", Currency: "USD"},
		{Amount: "1.234", Currency: "USD"},
		{Amount: "1", Currency: "ZZZ"},
	} {
		before := *unsupported
		if got := HydrateIndependentPrice(unsupported); got != nil {
			t.Errorf("HydrateIndependentPrice(%+v) = %+v, want nil", unsupported, got)
		}
		if *unsupported != before {
			t.Errorf("source price mutated from %+v to %+v", before, unsupported)
		}
	}
}
