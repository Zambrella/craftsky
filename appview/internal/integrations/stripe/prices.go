package stripe

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"time"

	"social.craftsky/appview/internal/subscriptions"
)

type cachedPrice struct {
	value   subscriptions.WebPrice
	expires time.Time
}

// MonthlyPrice reads the exact price used by Checkout, scoped to this client's
// verified merchant and mode. Short-lived caching bounds catalog reads.
func (c *Client) MonthlyPrice(ctx context.Context, id string) (subscriptions.WebPrice, error) {
	if !regexp.MustCompile(`^price_[A-Za-z0-9]+$`).MatchString(id) {
		return subscriptions.WebPrice{}, errors.New("invalid Stripe price")
	}
	c.pricesMu.Lock()
	cached, ok := c.prices[id]
	c.pricesMu.Unlock()
	if ok && time.Now().Before(cached.expires) {
		return cached.value, nil
	}
	if err := c.verifyAccount(ctx); err != nil {
		return subscriptions.WebPrice{}, err
	}
	var price struct {
		ID            string          `json:"id"`
		Active        bool            `json:"active"`
		LiveMode      *bool           `json:"livemode"`
		Amount        *int64          `json:"unit_amount"`
		Currency      string          `json:"currency"`
		Type          string          `json:"type"`
		BillingScheme string          `json:"billing_scheme"`
		TaxBehavior   string          `json:"tax_behavior"`
		Transform     json.RawMessage `json:"transform_quantity"`
		Recurring     struct {
			Interval string `json:"interval"`
			Count    int    `json:"interval_count"`
			Usage    string `json:"usage_type"`
		} `json:"recurring"`
	}
	if err := c.request(ctx, http.MethodGet, "/prices/"+id, nil, "", &price); err != nil {
		return subscriptions.WebPrice{}, err
	}
	if price.ID != id || !price.Active || price.LiveMode == nil || *price.LiveMode != (c.config.Environment == "production") || price.Amount == nil || *price.Amount <= 0 || *price.Amount > 9007199254740991 || !regexp.MustCompile(`^[a-z]{3}$`).MatchString(price.Currency) || price.Type != "recurring" || price.BillingScheme != "per_unit" || price.Recurring.Interval != "month" || price.Recurring.Count != 1 || price.Recurring.Usage != "licensed" || (len(price.Transform) != 0 && string(price.Transform) != "null") {
		return subscriptions.WebPrice{}, errors.New("Stripe price is not an active monthly catalog price")
	}
	value := subscriptions.WebPrice{Amount: *price.Amount, Currency: price.Currency, TaxBehavior: price.TaxBehavior}
	c.pricesMu.Lock()
	defer c.pricesMu.Unlock()
	if c.prices == nil {
		c.prices = make(map[string]cachedPrice)
	}
	c.prices[id] = cachedPrice{value, time.Now().Add(5 * time.Minute)}
	return value, nil
}
