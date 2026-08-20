package app

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/instagram"
	"social.craftsky/appview/internal/integrations/instagrammeta"
	"social.craftsky/appview/internal/middleware"
)

type instagramRuntimeDependencies struct {
	verification   *instagram.VerificationService
	webhook        http.Handler
	webhookWorker  *instagram.WebhookWorker
	imports        *instagram.ImportService
	account        *instagram.AccountStore
	reconciliation *instagram.ReconciliationWorker
	suggestions    *instagram.SuggestionService
	retention      *instagram.RetentionService
}

// newInstagramRuntimeDependencies owns optional Meta transport and every
// background capability built on the private Instagram storage foundation.
func newInstagramRuntimeDependencies(
	ctx context.Context,
	pool *pgxpool.Pool,
	storage *instagramStorageDependencies,
	owners *ownerDependencies,
	pdsEffects *pdsEffectDependencies,
	cfg Config,
	logger *slog.Logger,
) (*instagramRuntimeDependencies, error) {
	runtime := &instagramRuntimeDependencies{
		retention: instagram.NewRetentionService(
			pool,
			time.Now,
			instagram.RetentionServiceOptions{
				Restoration:        storage.restoration,
				ModerationReceipts: storage.moderationStore,
			},
		),
	}
	verificationStore := instagram.NewVerificationStore(pool)
	var challengeCodec *instagram.ChallengeCodec
	var err error
	if cfg.InstagramData.Available() {
		challengeCodec, err = instagram.NewChallengeCodec(rand.Reader, cfg.InstagramData.HMACKey())
		if err != nil {
			return nil, fmt.Errorf("instagram challenge codec: %w", err)
		}
	}
	runtime.verification, err = instagram.NewVerificationService(instagram.VerificationServiceOptions{
		Store: verificationStore, Codec: challengeCodec,
		TTL: cfg.InstagramLimits.ChallengeTTL, DMURL: cfg.InstagramMeta.DMURL(),
		HMACKey:   cfg.InstagramData.HMACKey(),
		Available: cfg.InstagramMeta.Enabled() && cfg.InstagramMeta.Configured(),
	})
	if err != nil {
		return nil, fmt.Errorf("instagram verification service: %w", err)
	}
	if cfg.InstagramMeta.Enabled() && cfg.InstagramMeta.Configured() {
		if storage.rateLimiter == nil {
			return nil, errors.New("instagram Meta integration requires the persistent rate limiter")
		}
		digests, err := instagrammeta.NewDigestCodec(
			cfg.InstagramData.HMACKey(),
			instagram.CanonicalizeChallenge,
		)
		if err != nil {
			return nil, fmt.Errorf("instagram webhook digest codec: %w", err)
		}
		reducer, err := instagrammeta.NewPayloadReducer(
			cfg.InstagramMeta.InstagramAccountID(),
			digests,
		)
		if err != nil {
			return nil, fmt.Errorf("instagram webhook reducer: %w", err)
		}
		webhookRateLimiter, err := middleware.NewInstagramWebhookRateLimiter(
			storage.rateLimiter,
			cfg.InstagramDeployment.TrustedProxyCIDRs(),
			cfg.InstagramLimits.WebhookIPPerMinute,
			cfg.InstagramLimits.WebhookGlobalPerMinute,
			cfg.InstagramLimits.InvalidIPPer15Minutes,
		)
		if err != nil {
			return nil, fmt.Errorf("instagram webhook limiter: %w", err)
		}
		retryPolicy := instagram.WebhookRetryPolicy{
			MaxAttempts:      cfg.InstagramLimits.WorkerMaxAttempts,
			InitialBackoff:   cfg.InstagramLimits.WorkerBackoffInitial,
			MaxBackoff:       cfg.InstagramLimits.WorkerBackoffMax,
			MaxProcessingAge: cfg.InstagramLimits.WorkerMaxProcessingAge,
		}
		webhookStore, err := instagram.NewWebhookStoreWithOptions(pool, instagram.WebhookStoreOptions{
			LeaseDuration: cfg.InstagramLimits.WorkerLeaseDuration,
			RetryPolicy:   retryPolicy,
		})
		if err != nil {
			return nil, fmt.Errorf("instagram webhook store: %w", err)
		}
		runtime.webhook, err = instagrammeta.NewWebhookHandler(instagrammeta.WebhookHandlerConfig{
			AppSecret:   []byte(cfg.InstagramMeta.AppSecret()),
			VerifyToken: cfg.InstagramMeta.VerifyToken(),
			Reducer:     reducer, Sink: webhookStore, Limiter: webhookRateLimiter,
			BodyLimitBytes: cfg.InstagramLimits.WebhookBodyLimitBytes,
			MaxEvents:      cfg.InstagramLimits.WebhookMaxEvents, Now: time.Now,
			Logger: logger, UnsafeDebugLogs: cfg.UnsafeLogInstagramWebhookBodies,
		})
		if err != nil {
			return nil, fmt.Errorf("instagram webhook handler: %w", err)
		}
		baseURL := cfg.InstagramMeta.APIBaseURL()
		metaClient, err := instagrammeta.NewHTTPClient(instagrammeta.HTTPClientConfig{
			HTTPClient: &http.Client{}, BaseURL: baseURL.String(),
			APIVersion:        cfg.InstagramMeta.APIVersion(),
			AccessToken:       cfg.InstagramMeta.AccessToken(),
			OfficialAccountID: cfg.InstagramMeta.InstagramAccountID(),
			RequestTimeout:    cfg.InstagramLimits.MetaHTTPTimeout,
			ResponseLimit:     cfg.InstagramLimits.MetaResponseLimitBytes,
			MaxConcurrent:     cfg.InstagramLimits.MetaLookupConcurrency,
		})
		if err != nil {
			return nil, fmt.Errorf("instagram Meta client: %w", err)
		}
		redeemer, err := instagram.NewVerificationWebhookRedeemer(verificationStore)
		if err != nil {
			return nil, fmt.Errorf("instagram webhook redeemer: %w", err)
		}
		replyText := ""
		if cfg.InstagramMeta.RepliesEnabled() {
			replyText = "CraftSky received your verification message. Return to CraftSky to confirm your Instagram username."
		}
		runtime.webhookWorker, err = instagram.NewWebhookWorker(
			webhookStore,
			redeemer,
			storage.membership,
			metaClient,
			instagram.WebhookWorkerOptions{
				BatchSize: 1, Now: time.Now, ReplyText: replyText,
				ReplyWindow:                cfg.InstagramLimits.DMReplyWindow,
				RateLimiter:                storage.rateLimiter,
				InvalidIGSIDPer15Minutes:   cfg.InstagramLimits.InvalidIGSIDPer15Minutes,
				MetaLookupsPerIGSIDPerHour: cfg.InstagramLimits.MetaLookupsPerIGSIDPerHour,
				MembershipInactivator:      storage.privateData,
				RetryPolicy:                retryPolicy,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("instagram webhook worker: %w", err)
		}
	}
	runtime.imports, err = instagram.NewImportService(instagram.ImportServiceOptions{
		Repository: instagram.NewImportStore(pool),
		Matcher: instagram.NewPrivateSuggestionMatcher(
			pool, storage.privateSuggestions, storage.suggestionPolicy, time.Now,
		),
		MaxEntries:      cfg.InstagramLimits.ImportMaxEntries,
		DefaultPageSize: cfg.InstagramLimits.PageDefault,
		MaxPageSize:     cfg.InstagramLimits.PageMax,
	})
	if err != nil {
		return nil, fmt.Errorf("instagram import service: %w", err)
	}
	runtime.account = instagram.NewAccountStore(pool, time.Now)
	runtime.reconciliation, err = instagram.NewReconciliationWorker(instagram.ReconciliationWorkerOptions{
		Pool: pool, PrivateSuggestions: storage.privateSuggestions,
		Policy: storage.suggestionPolicy, Membership: storage.membership,
		MembershipInactivator: storage.privateData, Now: time.Now,
		LeaseDuration:         cfg.InstagramLimits.WorkerLeaseDuration,
		MaxAttempts:           cfg.InstagramLimits.WorkerMaxAttempts,
		ModerationRestoration: storage.moderationRestoration,
	})
	if err != nil {
		return nil, fmt.Errorf("instagram reconciliation worker: %w", err)
	}
	runtime.suggestions, err = instagram.NewSuggestionService(
		storage.privateSuggestions,
		owners.lifecycles,
		storage.suggestionPolicy,
		instagramSuggestionEffectCoordinator{factory: pdsEffects.guarded},
	)
	if err != nil {
		return nil, fmt.Errorf("instagram suggestion service: %w", err)
	}
	return runtime, nil
}
