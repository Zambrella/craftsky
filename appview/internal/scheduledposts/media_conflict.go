package scheduledposts

import "errors"

// MediaConflictStage is an allowlisted, privacy-safe location for a scheduled
// media conflict. It must never contain request or media values.
type MediaConflictStage string

const (
	MediaConflictInvalidRequest           MediaConflictStage = "invalid_request"
	MediaConflictDigestConstruction       MediaConflictStage = "digest_construction"
	MediaConflictObjectKeyConstruction    MediaConflictStage = "object_key_construction"
	MediaConflictExistingBlob             MediaConflictStage = "existing_blob"
	MediaConflictExistingOwnerGeneration  MediaConflictStage = "existing_owner_generation"
	MediaConflictExistingUploadGeneration MediaConflictStage = "existing_upload_generation"
	MediaConflictExistingUploadAttempt    MediaConflictStage = "existing_upload_attempt"
	MediaConflictExistingObjectKey        MediaConflictStage = "existing_object_key"
	MediaConflictExistingMIMEType         MediaConflictStage = "existing_mime_type"
	MediaConflictExistingSize             MediaConflictStage = "existing_size"
	MediaConflictExistingDigest           MediaConflictStage = "existing_digest"
	MediaConflictObjectAttemptReservation MediaConflictStage = "object_attempt_reservation"
	MediaConflictAttemptOwner             MediaConflictStage = "object_attempt_owner"
	MediaConflictAttemptGeneration        MediaConflictStage = "object_attempt_generation"
	MediaConflictAttemptMedia             MediaConflictStage = "object_attempt_media"
	MediaConflictAttemptObjectKey         MediaConflictStage = "object_attempt_object_key"
	MediaConflictAttemptDigest            MediaConflictStage = "object_attempt_digest"
	MediaConflictAttemptOutcome           MediaConflictStage = "object_attempt_outcome"
	MediaConflictMediaReservation         MediaConflictStage = "media_reservation"
)

type scheduledMediaConflict struct {
	stage MediaConflictStage
}

func (conflict *scheduledMediaConflict) Error() string {
	return ErrScheduledMediaConflict.Error()
}

func (conflict *scheduledMediaConflict) Is(target error) bool {
	return target == ErrScheduledMediaConflict
}

// NewScheduledMediaConflict creates a conflict carrying only an allowlisted
// stage. It is exported so API-boundary tests can verify safe logging.
func NewScheduledMediaConflict(stage MediaConflictStage) error {
	return &scheduledMediaConflict{stage: stage}
}

// ScheduledMediaConflictStage returns a safe stage for structured logging.
func ScheduledMediaConflictStage(err error) string {
	var conflict *scheduledMediaConflict
	if errors.As(err, &conflict) && validMediaConflictStage(conflict.stage) {
		return string(conflict.stage)
	}
	return "unspecified"
}

func validMediaConflictStage(stage MediaConflictStage) bool {
	switch stage {
	case MediaConflictInvalidRequest,
		MediaConflictDigestConstruction,
		MediaConflictObjectKeyConstruction,
		MediaConflictExistingBlob,
		MediaConflictExistingOwnerGeneration,
		MediaConflictExistingUploadGeneration,
		MediaConflictExistingUploadAttempt,
		MediaConflictExistingObjectKey,
		MediaConflictExistingMIMEType,
		MediaConflictExistingSize,
		MediaConflictExistingDigest,
		MediaConflictObjectAttemptReservation,
		MediaConflictAttemptOwner,
		MediaConflictAttemptGeneration,
		MediaConflictAttemptMedia,
		MediaConflictAttemptObjectKey,
		MediaConflictAttemptDigest,
		MediaConflictAttemptOutcome,
		MediaConflictMediaReservation:
		return true
	default:
		return false
	}
}
