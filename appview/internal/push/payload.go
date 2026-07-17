package push

import "social.craftsky/appview/internal/notifications"

const maxRoutingFactBytes = 1024

type Payload struct {
	Title string            `json:"title"`
	Body  string            `json:"body"`
	Data  map[string]string `json:"data"`
}

func BuildPayload(category notifications.Category, routingID, actorDisplayName string, facts RoutingFacts) Payload {
	if actorDisplayName == "" {
		actorDisplayName = "Someone"
	}
	action := map[notifications.Category]string{
		notifications.Like:           "liked your post",
		notifications.Follow:         "followed you",
		notifications.Reply:          "replied to your post",
		notifications.Mention:        "mentioned you",
		notifications.Quote:          "quoted your post",
		notifications.Repost:         "reposted your post",
		notifications.EverythingElse: "sent you a notification",
	}[category]
	data := map[string]string{
		"payloadVersion":        "1",
		"type":                  string(category),
		"accountSubscriptionId": routingID,
	}
	switch category {
	case notifications.Follow:
		addRoutingFact(data, "actorDid", facts.ActorDID.String())
	case notifications.Like, notifications.Repost:
		addRoutingFact(data, "subjectUri", facts.SubjectURI.String())
	case notifications.Mention, notifications.Quote:
		addRoutingFact(data, "sourceUri", facts.SourceURI.String())
	case notifications.Reply:
		addRoutingFact(data, "subjectUri", facts.SubjectURI.String())
		addRoutingFact(data, "sourceUri", facts.SourceURI.String())
	}
	return Payload{
		Title: actorDisplayName,
		Body:  action,
		Data:  data,
	}
}

func addRoutingFact(data map[string]string, key, value string) {
	if value == "" || len(value) > maxRoutingFactBytes {
		return
	}
	for i := range len(value) {
		if value[i] > 0x7f {
			return
		}
	}
	data[key] = value
}
