package push

import (
	"strconv"

	"social.craftsky/appview/internal/notifications"
)

const maxRoutingFactBytes = 1024

type Payload struct {
	Title string            `json:"title"`
	Body  string            `json:"body"`
	Data  map[string]string `json:"data"`
}

func BuildPayload(category notifications.Category, routingID, actorDisplayName string, facts RoutingFacts) Payload {
	if category == notifications.InstagramMatch {
		return buildInstagramMatchPayload(routingID, facts)
	}
	if actorDisplayName == "" {
		actorDisplayName = "Someone"
	}
	action := visibleBody(category, facts.TargetRole)
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
		addRoutingFact(data, "rootUri", facts.RootURI.String())
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

func buildInstagramMatchPayload(routingID string, facts RoutingFacts) Payload {
	count := facts.SystemCount
	if count < 1 {
		count = 1
	}
	capped := facts.SystemCountCapped || count > 99
	if count > 99 {
		count = 99
	}
	data := map[string]string{
		"payloadVersion":        "1",
		"type":                  string(notifications.InstagramMatch),
		"accountSubscriptionId": routingID,
		"count":                 strconv.Itoa(count),
		"countCapped":           strconv.FormatBool(capped),
	}
	addRoutingFact(data, "notificationId", facts.NotificationID)
	if facts.SystemDestination == "instagramMigration" {
		data["destination"] = facts.SystemDestination
	}
	body := "New Instagram matches are ready to review"
	if count == 1 && !capped {
		body = "A new Instagram match is ready to review"
	}
	return Payload{Title: "CraftSky", Body: body, Data: data}
}

func visibleBody(category notifications.Category, role ContentRole) string {
	if role != ContentRoleComment && role != ContentRoleReply {
		role = ContentRolePost
	}
	noun := string(role)
	switch category {
	case notifications.Like:
		return "liked your " + noun
	case notifications.Follow:
		return "followed you"
	case notifications.Reply:
		if role == ContentRolePost {
			return "commented on your post"
		}
		return "replied to your " + noun
	case notifications.Mention:
		return "mentioned you"
	case notifications.Quote:
		return "quoted your " + noun
	case notifications.Repost:
		return "reposted your " + noun
	default:
		return "sent you a notification"
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
