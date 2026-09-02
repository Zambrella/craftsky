package business

import "errors"

var ErrInvalidActions = errors.New("business: invalid actions")

var actionCatalog = []string{
	"shop",
	"browse-patterns",
	"request-custom-order",
	"book-class",
	"book-appointment",
	"view-event-calendar",
	"email",
	"visit-website",
	"wholesale-enquiries",
}

type Action struct {
	Type        string `json:"type"`
	Destination string `json:"destination"`
}

func ActionCatalog() []string {
	return append([]string(nil), actionCatalog...)
}

func ValidateActions(actions []Action) error {
	if len(actions) > 1 {
		return ErrInvalidActions
	}
	if len(actions) == 1 && !catalogContains(actionCatalog, actions[0].Type) {
		return ErrInvalidActions
	}
	return nil
}
