# syntax=docker/dockerfile:1.7
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/appview ./cmd/appview \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/cli     ./cmd/cli

FROM alpine:3.19
RUN addgroup -S app && adduser -S -G app app \
 && apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /out/appview /app/appview
COPY --from=build /out/cli     /app/cli
COPY migrations /app/migrations
USER app
EXPOSE 8080
ENTRYPOINT ["/app/appview"]
CMD ["dev"]
