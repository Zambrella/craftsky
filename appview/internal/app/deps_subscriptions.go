package app

import (
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/integrations/revenuecat"
	"social.craftsky/appview/internal/observability"
	"social.craftsky/appview/internal/subscriptions"
)

type subscriptionDependencies struct {
	store      *subscriptions.Store
	webhook    http.Handler
	reconciler *subscriptions.Reconciler
}

func newSubscriptionDependencies(pool *pgxpool.Pool, config RevenueCatConfig, httpClient *http.Client, observer *observability.Observer) (*subscriptionDependencies, error) {
	dependencies := &subscriptionDependencies{store: subscriptions.NewStore(pool, observer)}
	if !config.Enabled() || !config.Configured() {
		return dependencies, nil
	}
	catalog, err := subscriptions.NewCatalog(subscriptions.CatalogConfig{
		ProjectID: config.projectID,
		AppIDs:    append([]string(nil), config.appIDs...),
		Products:  config.products,
	})
	if err != nil {
		return nil, fmt.Errorf("RevenueCat catalog: %w", err)
	}
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: config.apiTimeout,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
	}
	client, err := revenuecat.NewClient(revenuecat.ClientConfig{
		BaseURL: config.apiBaseURL, APIKey: config.apiKey.Reveal(), ProjectID: config.projectID,
		PageLimit: config.pageLimit, MaxPages: config.maxPages, MaxResponseBytes: config.maxResponseBytes,
	}, httpClient)
	if err != nil {
		return nil, fmt.Errorf("RevenueCat client: %w", err)
	}
	dependencies.reconciler = subscriptions.NewReconciler(dependencies.store, client, catalog, config.leaseDuration, time.Now, observer)
	dependencies.webhook, err = revenuecat.NewWebhookHandler(revenuecat.WebhookConfig{
		Authorization: config.webhookAuthorization.Reveal(), SigningSecret: config.webhookSigningSecret.Reveal(),
		BodyLimit: config.webhookBodyLimit, IngressDeadline: config.ingressDeadline,
		SignatureTolerance: config.signatureTolerance, Observer: observer,
	}, dependencies.store, time.Now)
	if err != nil {
		return nil, fmt.Errorf("RevenueCat webhook: %w", err)
	}
	return dependencies, nil
}
