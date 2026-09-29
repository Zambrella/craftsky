package index

import (
	"sort"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/tap"
)

type SetScope struct {
	Kind  string
	Actor syntax.DID
	Key   string
}

type SetSource struct {
	URI                 syntax.ATURI
	Scope               SetScope
	ActivityAt          time.Time
	Eligible            bool
	IneligibilityReason string
	Dependency          tap.Dependency
	SubjectDID          syntax.DID
	SubjectURI          syntax.ATURI
	SubjectCID          syntax.CID
}

func classifySetSource(source SetSource, actorEligible bool, dependency tap.Dependency, dependencyAvailable bool, reason string) SetSource {
	if actorEligible && (dependency == (tap.Dependency{}) || dependencyAvailable) {
		source.Eligible = true
		source.IneligibilityReason = ""
		source.Dependency = tap.Dependency{}
		return source
	}
	source.Eligible = false
	source.IneligibilityReason = reason
	source.Dependency = dependency
	return source
}

type SetAggregate struct {
	EligibleSourceCount int
	RepresentativeURI   syntax.ATURI
	ActivatedAt         time.Time
}

type SetTransition string

const (
	SetUnchanged   SetTransition = "unchanged"
	SetActivated   SetTransition = "activated"
	SetDeactivated SetTransition = "deactivated"
)

type SetAggregateChange struct {
	Scope      SetScope
	Previous   *SetAggregate
	Current    *SetAggregate
	Transition SetTransition
}

func affectedSetScopes(previous, next *SetSource) []SetScope {
	unique := make(map[SetScope]struct{}, 2)
	if previous != nil {
		unique[previous.Scope] = struct{}{}
	}
	if next != nil {
		unique[next.Scope] = struct{}{}
	}
	if len(unique) == 0 {
		return nil
	}
	scopes := make([]SetScope, 0, len(unique))
	for scope := range unique {
		scopes = append(scopes, scope)
	}
	sort.Slice(scopes, func(i, j int) bool {
		if scopes[i].Kind != scopes[j].Kind {
			return scopes[i].Kind < scopes[j].Kind
		}
		if scopes[i].Actor != scopes[j].Actor {
			return scopes[i].Actor < scopes[j].Actor
		}
		return scopes[i].Key < scopes[j].Key
	})
	return scopes
}

func reduceSetAggregate(previous *SetAggregate, sources []SetSource, now time.Time) (*SetAggregate, SetTransition) {
	eligible := make([]SetSource, 0, len(sources))
	for _, source := range sources {
		if source.Eligible {
			eligible = append(eligible, source)
		}
	}
	if len(eligible) == 0 {
		if previous != nil {
			return nil, SetDeactivated
		}
		return nil, SetUnchanged
	}
	sort.Slice(eligible, func(i, j int) bool {
		if !eligible[i].ActivityAt.Equal(eligible[j].ActivityAt) {
			return eligible[i].ActivityAt.Before(eligible[j].ActivityAt)
		}
		return eligible[i].URI < eligible[j].URI
	})
	activatedAt := now
	transition := SetActivated
	if previous != nil {
		activatedAt = previous.ActivatedAt
		transition = SetUnchanged
	}
	return &SetAggregate{
		EligibleSourceCount: len(eligible),
		RepresentativeURI:   eligible[0].URI,
		ActivatedAt:         activatedAt,
	}, transition
}
