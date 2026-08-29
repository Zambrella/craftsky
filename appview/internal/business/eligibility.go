package business

import "time"

type EventPolicyInput struct {
	CallerIsOwner bool
	OwnerCurrent  bool
	AccountType   AccountType
	Blocked       bool
	StartsAt      time.Time
	EndsAt        time.Time
	Status        string
	Moderated     bool
	AsOf          time.Time
}

type EventPolicyResult struct {
	OwnerManagement          bool
	VisitorDirect            bool
	DirectVisible            bool
	Upcoming                 bool
	Past                     bool
	PublicSuppressionReasons []string
	UpcomingExclusionReasons []string
}

func IsBusinessClassified(currentMember bool, accountType AccountType) bool {
	return currentMember && accountType == AccountTypeBusiness
}

func EvaluateEvent(input EventPolicyInput) EventPolicyResult {
	invalidRange := !input.EndsAt.After(input.StartsAt)
	overDuration := !invalidRange && input.EndsAt.Sub(input.StartsAt) > 31*24*time.Hour
	ownerNotBusiness := !IsBusinessClassified(input.OwnerCurrent, input.AccountType)
	ended := !input.EndsAt.After(input.AsOf)

	publicReasons := make([]string, 0, 4)
	if ownerNotBusiness {
		publicReasons = append(publicReasons, "owner-not-business")
	}
	if invalidRange {
		publicReasons = append(publicReasons, "invalid-time-range")
	}
	if overDuration {
		publicReasons = append(publicReasons, "duration-exceeds-limit")
	}
	if input.Moderated {
		publicReasons = append(publicReasons, "record-moderated")
	}

	upcomingReasons := make([]string, 0, 7)
	if ended {
		upcomingReasons = append(upcomingReasons, "ended")
	}
	if input.Status == "cancelled" {
		upcomingReasons = append(upcomingReasons, "cancelled")
	}
	if input.Status == "postponed" {
		upcomingReasons = append(upcomingReasons, "postponed")
	}
	upcomingReasons = append(upcomingReasons, publicReasons...)

	publicEligible := len(publicReasons) == 0
	visitorEligible := publicEligible && !input.Blocked
	ownerManagement := input.CallerIsOwner && input.OwnerCurrent
	visitorDirect := !input.CallerIsOwner && visitorEligible
	return EventPolicyResult{
		OwnerManagement:          ownerManagement,
		VisitorDirect:            visitorDirect,
		DirectVisible:            ownerManagement || visitorDirect,
		Upcoming:                 visitorEligible && len(upcomingReasons) == 0,
		Past:                     ended,
		PublicSuppressionReasons: publicReasons,
		UpcomingExclusionReasons: upcomingReasons,
	}
}
