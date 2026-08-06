package api

import "errors"

type ProfilePinSlot string

const (
	ProfilePinSlotStandard ProfilePinSlot = "standard"
	ProfilePinSlotProject  ProfilePinSlot = "project"
)

var ErrProfilePinNotAllowed = errors.New("profile pin: target not allowed")

type ProfilePinTargetShape struct {
	IsProject                 bool
	HasProjectMaterialization bool
	HasReplyRoot              bool
	HasReplyParent            bool
	HasQuote                  bool
}

func ClassifyProfilePinSlot(target ProfilePinTargetShape) (ProfilePinSlot, error) {
	if target.HasReplyRoot || target.HasReplyParent {
		return "", ErrProfilePinNotAllowed
	}
	if target.IsProject != target.HasProjectMaterialization {
		return "", ErrProfilePinNotAllowed
	}
	if target.IsProject {
		if target.HasQuote {
			return "", ErrProfilePinNotAllowed
		}
		return ProfilePinSlotProject, nil
	}
	return ProfilePinSlotStandard, nil
}
