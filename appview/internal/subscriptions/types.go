package subscriptions

import (
	"context"
	"errors"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
)

// BillingObserver records bounded outcomes from subscription operations without exposing billing identifiers.
type BillingObserver interface {
	ObserveSubscriptionAssignment(context.Context, string, string)
	ObserveSubscriptionAnomalies(context.Context, int)
	ObserveBillingClosure(context.Context, string)
}

var ErrBillingAccountNotFound = errors.New("billing account not found")
var ErrStaleSnapshot = errors.New("stale subscription snapshot")
var ErrLicenseNotFound = errors.New("billing license not found")
var ErrAssignmentTargetIneligible = errors.New("assignment target is ineligible")
var ErrDIDAlreadyAssigned = errors.New("target already has a billing license")
var ErrAssignmentCooldown = errors.New("billing license reassignment is cooling down")
var ErrProviderBillingMustBeResolved = errors.New("provider billing must be resolved")

// Tier identifies a CraftSky access level used by billing and authorization.
type Tier string

const (
	TierFree     Tier = "free"
	TierPlus     Tier = "plus"
	TierBusiness Tier = "business"
)

// Valid reports whether tier is one of the complete public/storage tier keys.
func (tier Tier) Valid() bool {
	return tier == TierFree || tier == TierPlus || tier == TierBusiness
}

// Paid reports whether tier may be used by a provider product mapping.
func (tier Tier) Paid() bool {
	return tier == TierPlus || tier == TierBusiness
}

// LocalAssignment summarizes the locally assigned license state used to resolve access for a DID.
type LocalAssignment struct {
	Tier                     Tier
	GivesAccess              bool
	AccessEndsAt             *time.Time
	ReconciliationStaleSince *time.Time
}

// SelfAccess is the privacy-minimized subscription access response for the authenticated DID.
type SelfAccess struct {
	DID           syntax.DID `json:"did"`
	EffectiveTier Tier       `json:"effectiveTier"`
	GivesAccess   bool       `json:"givesAccess"`
	AccessEndsAt  *time.Time `json:"accessEndsAt,omitempty"`
	AssignedTier  *Tier      `json:"assignedTier,omitempty"`
}

// BillingAccount identifies a private RevenueCat customer and its reconciliation progress for one owner.
type BillingAccount struct {
	ID                      uuid.UUID
	OwnerDID                syntax.DID
	RevenueCatAppUserID     uuid.UUID
	RequestedGeneration     int64
	ReconciledGeneration    int64
	ReconciliationRequested *time.Time
	ReconciledAt            *time.Time
}

// BillingState is the owner-visible view of a billing account, its subscriptions, and assignable licenses.
type BillingState struct {
	BillingAccountID        uuid.UUID             `json:"billingAccountId"`
	RevenueCatAppUserID     uuid.UUID             `json:"revenueCatAppUserId"`
	RequestedGeneration     int64                 `json:"requestedGeneration"`
	ReconciledGeneration    int64                 `json:"reconciledGeneration"`
	ReconciliationRequested *time.Time            `json:"reconciliationRequestedAt,omitempty"`
	ReconciledAt            *time.Time            `json:"reconciledAt,omitempty"`
	ReconciliationStale     bool                  `json:"reconciliationStale"`
	Subscriptions           []BillingSubscription `json:"subscriptions"`
	Licenses                []BillingLicense      `json:"licenses"`
}

// BillingSubscription exposes provider subscription state needed by its billing owner.
type BillingSubscription struct {
	ID                    uuid.UUID  `json:"id"`
	ProductID             string     `json:"productId"`
	Store                 string     `json:"store"`
	Status                string     `json:"status"`
	GivesAccess           bool       `json:"givesAccess"`
	PendingPayment        bool       `json:"pendingPayment"`
	AutoRenewalStatus     *string    `json:"autoRenewalStatus,omitempty"`
	CurrentPeriodStartsAt *time.Time `json:"currentPeriodStartsAt,omitempty"`
	CurrentPeriodEndsAt   *time.Time `json:"currentPeriodEndsAt,omitempty"`
	EndsAt                *time.Time `json:"endsAt,omitempty"`
	Anomaly               string     `json:"anomaly"`
}

// BillingLicense represents one paid entitlement that an owner can assign to a CraftSky DID.
type BillingLicense struct {
	ID          uuid.UUID   `json:"id"`
	Tier        Tier        `json:"tier"`
	AssignedDID *syntax.DID `json:"assignedDid,omitempty"`
	AssignedAt  *time.Time  `json:"assignedAt,omitempty"`
	Assignable  bool        `json:"assignable"`
	Anomaly     string      `json:"anomaly"`
}

// ProviderSubscriptionSnapshot carries one authoritative subscription returned by the billing provider.
type ProviderSubscriptionSnapshot struct {
	ID                    string
	ProductID             string
	PendingProductID      string
	Store                 string
	Environment           string
	Status                string
	GivesAccess           bool
	PendingPayment        bool
	AutoRenewalStatus     string
	StartsAt              *time.Time
	CurrentPeriodStartsAt *time.Time
	CurrentPeriodEndsAt   *time.Time
	EndsAt                *time.Time
}

// CompleteSnapshot groups a provider customer's subscriptions and records whether enumeration completed.
type CompleteSnapshot struct {
	Subscriptions []ProviderSubscriptionSnapshot
	Complete      bool
}

// SnapshotClaim fences application of a fetched provider snapshot to one reconciliation lease and generation.
type SnapshotClaim struct {
	BillingAccountID    uuid.UUID
	RevenueCatAppUserID uuid.UUID
	Generation          int64
	LeaseToken          uuid.UUID
}

// RevenueCatEvent contains the minimal webhook identity needed to deduplicate and schedule reconciliation.
type RevenueCatEvent struct {
	ID                  string
	Type                string
	RevenueCatAppUserID uuid.UUID
	OccurredAt          time.Time
}

// AssignParams supplies the authenticated owner, target, and request context for assigning a license.
type AssignParams struct {
	OwnerDID  syntax.DID
	LicenseID uuid.UUID
	TargetDID syntax.DID
	DeviceID  string
	Now       time.Time
}

// Assignment reports the resulting license-to-DID relationship after an assignment operation.
type Assignment struct {
	LicenseID  uuid.UUID  `json:"licenseId"`
	TargetDID  syntax.DID `json:"targetDid"`
	AssignedAt time.Time  `json:"assignedAt"`
}

// UnassignParams supplies the authenticated owner and request context for removing a license assignment.
type UnassignParams struct {
	OwnerDID  syntax.DID
	LicenseID uuid.UUID
	Now       time.Time
}
