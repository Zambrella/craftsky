# syntax=docker/dockerfile:1.7
FROM golang:1.26.6-alpine@sha256:af8d6740070b8906d12eae1c3e3ea0957fb63f492051ea05e354c38ef9fe88df AS build
WORKDIR /src
RUN test "$(go env GOVERSION)" = "go1.26.6"
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN mkdir -p /out /evidence /media-evidence \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/appview ./cmd/appview \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/cli     ./cmd/cli \
 && CGO_ENABLED=0 go build -trimpath -o /evidence/appview ./cmd/appview \
 && CGO_ENABLED=0 go build -trimpath -o /evidence/cli     ./cmd/cli \
 && CGO_ENABLED=0 go test -c -trimpath -o /media-evidence/api.test ./internal/api \
 && CGO_ENABLED=0 go list -deps ./cmd/appview > /evidence/appview-deps.txt \
 && CGO_ENABLED=0 go list -deps ./cmd/cli     > /evidence/cli-deps.txt

# Same-source, same-toolchain unstripped companions used only to prove whether
# a package/symbol is linked when govulncheck's stripped-binary scan is less
# precise. This stage is exported by appview-check and is not part of runtime.
FROM scratch AS audit-evidence
COPY --from=build /evidence/ /

FROM alpine:3.24@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b AS runtime-base
RUN addgroup -S app && adduser -S -G app app \
 && apk add --no-cache ca-certificates
WORKDIR /app
USER app

# Test-only evidence target. It uses the exact release runtime base and the
# same-builder API test binary, but is never copied into the shipped image.
FROM runtime-base AS media-decode-evidence
COPY --from=build /media-evidence/api.test /evidence/api.test
ENTRYPOINT ["/evidence/api.test"]

FROM runtime-base
COPY --from=build /out/appview /app/appview
COPY --from=build /out/cli     /app/cli
COPY migrations /app/migrations
COPY devmedia /app/devmedia
EXPOSE 8080
ENTRYPOINT ["/app/appview"]
CMD ["dev"]
