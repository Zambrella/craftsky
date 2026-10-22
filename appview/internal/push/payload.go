package push

import (
	"social.craftsky/appview/internal/notifications"
	"strings"
	"unicode"
	"unicode/utf8"
)

const maxRoutingFactBytes = 1024

type Payload struct {
	Title string            `json:"title"`
	Body  string            `json:"body"`
	Data  map[string]string `json:"data"`
}

func BuildPayload(category notifications.Category, routingID, actorDisplayName string, facts RoutingFacts) Payload {
	if category == notifications.Moderation {
		data := map[string]string{
			"payloadVersion":        "1",
			"type":                  string(category),
			"accountSubscriptionId": routingID,
		}
		addRoutingFact(data, "notificationId", facts.NotificationID)
		addRoutingFact(data, "caseReference", facts.CaseReference)
		return Payload{Title: "CraftSky", Body: "There is an update to your account standing", Data: data}
	}
	if category == notifications.InstagramMatch {
		data := map[string]string{
			"payloadVersion":        "1",
			"type":                  string(category),
			"accountSubscriptionId": routingID,
		}
		addRoutingFact(data, "notificationId", facts.NotificationID)
		return Payload{
			Title: "CraftSky",
			Body:  "Someone you follow on Instagram is now on CraftSky",
			Data:  data,
		}
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
		addRoutingFact(data, "rootUri", facts.RootURI.String())
		addRoutingFact(data, "sourceUri", facts.SourceURI.String())
	}
	return Payload{
		Title: actorDisplayName,
		Body:  action,
		Data:  data,
	}
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
		return "mentioned you in a " + noun
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

// Keep display text out of routing data and below the sender's 256-byte limit.
// Title and excerpt have separate budgets so a long title cannot consume both.
func previewBody(category notifications.Category, action string, preview *notifications.Preview) string {
	if preview == nil {
		return action
	}
	switch category {
	case notifications.Reply, notifications.Mention, notifications.Quote:
	default:
		return action
	}
	parts := []string{action}
	if title := boundedPreviewText(preview.ProjectTitle, 60); title != "" {
		parts = append(parts, title)
	}
	if text := boundedPreviewText(preview.Text, 140); text != "" {
		parts = append(parts, text)
	}
	return strings.Join(parts, ": ")
}

func boundedPreviewText(value string, maxBytes int) string {
	value = strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return ' '
		}
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return -1
		}
		return r
	}, strings.ToValidUTF8(value, ""))
	value = strings.Join(strings.Fields(value), " ")
	if len(value) <= maxBytes {
		return value
	}
	end := maxBytes - len("…")
	for end > 0 && !utf8.RuneStart(value[end]) {
		end--
	}
	return strings.TrimSpace(value[:end]) + "…"
}
