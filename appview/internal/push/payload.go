package push

import "social.craftsky/appview/internal/notifications"

type Payload struct {
	Title string            `json:"title"`
	Body  string            `json:"body"`
	Data  map[string]string `json:"data"`
}

func BuildPayload(notificationID string, category notifications.Category, routingID, actorDisplayName string) Payload {
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
	return Payload{
		Title: actorDisplayName,
		Body:  action,
		Data:  map[string]string{"notificationId": notificationID, "type": string(category), "accountSubscriptionId": routingID},
	}
}
