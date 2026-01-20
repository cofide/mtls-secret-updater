FROM golang:1.25 AS build

WORKDIR /build

# Download dependencies and add them to a cache.
# Do this before copying the source code to avoid cache invalidation following code changes.
RUN go env -w GOCACHE=/go-cache
RUN go env -w GOMODCACHE=/gomod-cache
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/gomod-cache go mod download

COPY cmd cmd

RUN --mount=type=cache,target=/gomod-cache --mount=type=cache,target=/go-cache \
    CGO_ENABLED=0 go build -ldflags="-s -w" -o ./mtls-secret-updater cmd/main.go

##################################################

FROM cgr.dev/chainguard/static:latest

WORKDIR /

COPY --from=build /build/mtls-secret-updater /mtls-secret-updater

USER nonroot:nonroot

ENTRYPOINT ["/mtls-secret-updater"]
