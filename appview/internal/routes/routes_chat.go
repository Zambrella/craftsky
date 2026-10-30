package routes

import (
	"log/slog"
	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/auth"
)

type chatRouteBundle struct {
	mux            Registrar
	middleware     v1Middleware
	status         api.ChatStatusReader
	sender         api.ChatSender
	consumer       api.ChatConsumer
	authorization  auth.ChatAuthorizationStarter
	allowDevScheme bool
	logger         *slog.Logger
}

func registerChatRoutes(routes chatRouteBundle) {
	routes.mux.Handle("GET /v1/chat/status", routes.middleware.wrap(mustPolicy("GET", "/v1/chat/status"), api.ChatStatusHandler(routes.status)))
	routes.mux.Handle("POST /v1/chat/authorization", routes.middleware.wrap(mustPolicy("POST", "/v1/chat/authorization"), auth.ChatAuthorizationHandler(routes.authorization, routes.allowDevScheme, routes.logger)))
	routes.mux.Handle("POST /v1/chat/conversations/{id}/messages", routes.middleware.wrap(mustPolicy("POST", "/v1/chat/conversations/{id}/messages"), api.ChatSendHandler(routes.sender)))
	routes.mux.Handle("PUT /v1/chat/connection", routes.middleware.wrap(mustPolicy("PUT", "/v1/chat/connection"), api.ChatConsumerHandler(routes.consumer, "connection")))
	routes.mux.Handle("GET /v1/chat/conversations", routes.middleware.wrap(mustPolicy("GET", "/v1/chat/conversations"), api.ChatConsumerHandler(routes.consumer, "list")))
	routes.mux.Handle("GET /v1/chat/availability", routes.middleware.wrap(mustPolicy("GET", "/v1/chat/availability"), api.ChatConsumerHandler(routes.consumer, "availability")))
	routes.mux.Handle("POST /v1/chat/conversations", routes.middleware.wrap(mustPolicy("POST", "/v1/chat/conversations"), api.ChatConsumerHandler(routes.consumer, "start")))
	routes.mux.Handle("GET /v1/chat/conversations/{id}", routes.middleware.wrap(mustPolicy("GET", "/v1/chat/conversations/{id}"), api.ChatConsumerHandler(routes.consumer, "conversation")))
	routes.mux.Handle("GET /v1/chat/conversations/{id}/details", routes.middleware.wrap(mustPolicy("GET", "/v1/chat/conversations/{id}/details"), api.ChatConsumerHandler(routes.consumer, "details")))
	routes.mux.Handle("GET /v1/chat/conversations/{id}/messages", routes.middleware.wrap(mustPolicy("GET", "/v1/chat/conversations/{id}/messages"), api.ChatConsumerHandler(routes.consumer, "history")))
	routes.mux.Handle("GET /v1/chat/operations/{operationId}", routes.middleware.wrap(mustPolicy("GET", "/v1/chat/operations/{operationId}"), api.ChatConsumerHandler(routes.consumer, "receipt")))
	routes.mux.Handle("PUT /v1/chat/conversations/{id}/read", routes.middleware.wrap(mustPolicy("PUT", "/v1/chat/conversations/{id}/read"), api.ChatConsumerHandler(routes.consumer, "read")))
	routes.mux.Handle("POST /v1/chat/conversations/{id}/accept", routes.middleware.wrap(mustPolicy("POST", "/v1/chat/conversations/{id}/accept"), api.ChatConsumerHandler(routes.consumer, "accept")))
	routes.mux.Handle("PUT /v1/chat/conversations/{id}/mute", routes.middleware.wrap(mustPolicy("PUT", "/v1/chat/conversations/{id}/mute"), api.ChatConsumerHandler(routes.consumer, "mute")))
	routes.mux.Handle("DELETE /v1/chat/conversations/{id}/mute", routes.middleware.wrap(mustPolicy("DELETE", "/v1/chat/conversations/{id}/mute"), api.ChatConsumerHandler(routes.consumer, "unmute")))
	routes.mux.Handle("POST /v1/chat/conversations/{id}/leave", routes.middleware.wrap(mustPolicy("POST", "/v1/chat/conversations/{id}/leave"), api.ChatConsumerHandler(routes.consumer, "leave")))
	routes.mux.Handle("POST /v1/chat/reports", routes.middleware.wrap(mustPolicy("POST", "/v1/chat/reports"), api.ChatConsumerHandler(routes.consumer, "report")))
	routes.mux.Handle("GET /v1/chat/unread-counts", routes.middleware.wrap(mustPolicy("GET", "/v1/chat/unread-counts"), api.ChatConsumerHandler(routes.consumer, "counts")))
	routes.mux.Handle("PUT /v1/chat/incoming-policy", routes.middleware.wrap(mustPolicy("PUT", "/v1/chat/incoming-policy"), api.ChatConsumerHandler(routes.consumer, "setPolicy")))
	routes.mux.Handle("GET /v1/chat/incoming-policy", routes.middleware.wrap(mustPolicy("GET", "/v1/chat/incoming-policy"), api.ChatConsumerHandler(routes.consumer, "policy")))
	routes.mux.Handle("GET /v1/chat/push-preferences", routes.middleware.wrap(mustPolicy("GET", "/v1/chat/push-preferences"), api.ChatConsumerHandler(routes.consumer, "pushPreference")))
	routes.mux.Handle("PUT /v1/chat/push-preferences", routes.middleware.wrap(mustPolicy("PUT", "/v1/chat/push-preferences"), api.ChatConsumerHandler(routes.consumer, "setPushPreference")))
	routes.mux.Handle("POST /v1/chat/sync", routes.middleware.wrap(mustPolicy("POST", "/v1/chat/sync"), api.ChatConsumerHandler(routes.consumer, "sync")))
	routes.mux.Handle("GET /v1/chat/notification-opens/{reference}", routes.middleware.wrap(mustPolicy("GET", "/v1/chat/notification-opens/{reference}"), api.ChatConsumerHandler(routes.consumer, "open")))
}
