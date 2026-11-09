package app

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/observability"
	"social.craftsky/appview/internal/video"
)

type videoDependencies struct {
	uploadAuthorization api.VideoUploadAuthorizationIssuer
	uploadLimits        api.VideoUploadLimitsService
	completionVerifier  api.VideoCompletionVerifier
	playback            api.PlaybackURLBuilder
	bunny               *video.Bunny
	cleanup             *video.BunnyCleanup
	importExisting      func(context.Context, syntax.DID, syntax.CID) (string, error)
	captionFetcher      api.VideoCaptionBlobFetcher
}

type observedCompletionVerifier struct {
	inner    api.VideoCompletionVerifier
	observer *observability.Observer
}

func (verifier observedCompletionVerifier) Verify(ctx context.Context, owner syntax.DID, jobID string, blob video.Blob) (video.Blob, error) {
	started := time.Now()
	verified, err := verifier.inner.Verify(ctx, owner, jobID, blob)
	result, reason := "success", "none"
	if err != nil {
		result, reason = "rejected", "invalid_request"
		var verificationError *video.VerificationError
		if errors.As(err, &verificationError) && verificationError.Kind == video.VerificationUnavailable {
			result, reason = "unavailable", "upstream"
		} else if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			result, reason = "canceled", "upstream"
		}
	}
	verifier.observer.ObserveVideoOperation(ctx, "verification", result, reason, time.Since(started))
	return verified, err
}

type bunnyDependencies struct {
	pool    *pgxpool.Pool
	effects api.BlobEffectFactory
}

func newVideoDependencies(authCapability *authDependencies, federated *federatedClients, cfg Config, observer *observability.Observer, options ...bunnyDependencies) (*videoDependencies, error) {
	if authCapability == nil || authCapability.sessionCoordinator == nil || federated == nil || federated.pdsJSON == nil {
		return nil, errors.New("video dependencies unavailable")
	}
	uploadAuthorization, err := video.NewUploadAuthorizationIssuer(video.UploadAuthorizationIssuerOptions{
		Sessions:  authCapability.sessionCoordinator,
		PDSClient: federated.pdsJSON,
		Now:       time.Now,
		Lifetime:  30 * time.Minute,
	})
	if err != nil {
		return nil, err
	}
	serviceDID, err := syntax.ParseDID("did:web:video.bsky.app")
	if err != nil {
		return nil, err
	}
	uploadLimits, err := video.NewUploadLimitsService(video.UploadLimitsServiceOptions{
		Sessions:    authCapability.sessionCoordinator,
		PDSClient:   federated.pdsJSON,
		VideoClient: federated.pdsJSON,
		ServiceDID:  serviceDID,
		ServiceURL:  cfg.VideoServiceURL,
		Now:         time.Now,
	})
	if err != nil {
		return nil, err
	}
	statusClient, err := video.NewPublicJobStatusClient(cfg.VideoServiceURL, federated.pdsJSON)
	if err != nil {
		return nil, err
	}
	playback, err := video.NewPlaybackURLBuilder(video.PlaybackConfig{
		PlaylistTemplate: cfg.VideoPlaylistURLTemplate, ThumbnailTemplate: cfg.VideoThumbnailURLTemplate,
	})
	if err != nil {
		return nil, err
	}
	captionFetcher, err := video.NewCaptionFetcher(federated.directory, federated.pdsBlob, federated.boundary)
	if err != nil {
		return nil, err
	}
	if cfg.VideoProvider == "bunny" {
		if len(options) != 1 || options[0].effects == nil {
			return nil, errors.New("bunny dependencies unavailable")
		}
		factory := func(ctx context.Context, owner syntax.DID, session string) (video.BunnyBlobEffects, error) {
			return options[0].effects(ctx, owner, session)
		}
		bunny, err := video.NewBunny(options[0].pool, cfg.BunnyStreamLibraryID, string(cfg.BunnyStreamAPIKey), cfg.BunnyStreamCDNHostname, factory, playback, authCapability.sessionCoordinator)
		if err != nil {
			return nil, err
		}
		bunny.SetPDSBlobLimits(video.NewPDSBlobLimits(federated.pdsJSON))
		return &videoDependencies{uploadAuthorization: bunny, uploadLimits: bunny, completionVerifier: bunny, playback: bunny, captionFetcher: captionFetcher, bunny: bunny, cleanup: video.NewBunnyCleanup(bunny, string(cfg.BunnyCDNPurgeAPIKey), observer), importExisting: func(ctx context.Context, owner syntax.DID, cid syntax.CID) (string, error) {
			return bunny.ImportExisting(ctx, owner, cid, federated.directory, federated.boundary)
		}}, nil
	}
	return &videoDependencies{
		uploadAuthorization: uploadAuthorization, uploadLimits: uploadLimits,
		completionVerifier: observedCompletionVerifier{inner: video.NewCompletionVerifier(statusClient), observer: observer}, playback: playback, captionFetcher: captionFetcher,
	}, nil
}
