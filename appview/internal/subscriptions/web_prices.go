package subscriptions

import "context"

// WebPrice is a public catalog amount, never a customer-specific billing quote.
// Amount is in Stripe's currency minor units; monthly plans are validated upstream.
type WebPrice struct {
	Tier        Tier   `json:"tier"`
	Amount      int64  `json:"amount"`
	Currency    string `json:"currency"`
	TaxBehavior string `json:"taxBehavior"`
}

type WebPriceProvider interface {
	MonthlyPrice(context.Context, string) (WebPrice, error)
}

func (s *Store) WebPrices(ctx context.Context) ([]WebPrice, error) {
	provider, ok := s.checkoutProvider.(WebPriceProvider)
	if !ok {
		return nil, ErrWebCheckoutUnavailable
	}
	prices := make([]WebPrice, 0, 2)
	for _, entry := range []struct {
		tier Tier
		id   string
	}{{TierPlus, s.checkoutConfig.PlusPriceID}, {TierBusiness, s.checkoutConfig.BusinessPriceID}} {
		price, err := provider.MonthlyPrice(ctx, entry.id)
		if err != nil {
			return nil, err
		}
		price.Tier = entry.tier
		prices = append(prices, price)
	}
	return prices, nil
}
