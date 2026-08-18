package instagram

import (
	"errors"
	"fmt"
)

var ErrSuggestionGenerationChanged = errors.New("Instagram suggestion owner generation changed")

// SuggestionState describes only caller-private suggestion state. No state in
// this machine grants a background component authority to cross a PDS
// boundary; accepting is a short reservation owned by an explicit request.
type SuggestionState string

const (
	SuggestionPending          SuggestionState = "pending"
	SuggestionAccepting        SuggestionState = "accepting"
	SuggestionFollowed         SuggestionState = "followed"
	SuggestionAlreadyFollowing SuggestionState = "alreadyFollowing"
	SuggestionDismissed        SuggestionState = "dismissed"
	SuggestionInvalidated      SuggestionState = "invalidated"
)

func (state SuggestionState) Valid() bool {
	switch state {
	case SuggestionPending,
		SuggestionAccepting,
		SuggestionFollowed,
		SuggestionAlreadyFollowing,
		SuggestionDismissed,
		SuggestionInvalidated:
		return true
	default:
		return false
	}
}

func (state SuggestionState) Terminal() bool {
	switch state {
	case SuggestionFollowed,
		SuggestionAlreadyFollowing,
		SuggestionDismissed,
		SuggestionInvalidated:
		return true
	default:
		return false
	}
}

func ValidateSuggestionTransition(from, to SuggestionState) error {
	if !from.Valid() || !to.Valid() {
		return ErrInvalidInstagramState
	}
	if from == to {
		return nil
	}
	allowed := false
	switch from {
	case SuggestionPending:
		allowed = oneOf(to, SuggestionAccepting, SuggestionDismissed, SuggestionInvalidated)
	case SuggestionAccepting:
		allowed = oneOf(
			to,
			SuggestionPending,
			SuggestionFollowed,
			SuggestionAlreadyFollowing,
			SuggestionInvalidated,
		)
	}
	return transitionResult(allowed, from, to)
}

func ValidateSuggestionGenerations(
	storedImporter int64,
	storedTarget int64,
	currentImporter int64,
	currentTarget int64,
) error {
	if storedImporter <= 0 || storedTarget <= 0 ||
		currentImporter <= 0 || currentTarget <= 0 ||
		storedImporter != currentImporter || storedTarget != currentTarget {
		return fmt.Errorf(
			"%w: stored importer/target %d/%d, current %d/%d",
			ErrSuggestionGenerationChanged,
			storedImporter,
			storedTarget,
			currentImporter,
			currentTarget,
		)
	}
	return nil
}
