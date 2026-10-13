package app

import (
	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/imageupload"
)

func newImageUploadDependencies(authCapability *authDependencies, federated *federatedClients) (api.ImageUploadAuthorizationIssuer, error) {
	return imageupload.NewIssuer(authCapability.sessionCoordinator, federated.pdsJSON)
}
