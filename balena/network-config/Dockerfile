# netcfgd — network and proxy configuration for balenaOS devices.
#
# Two stages, no build tooling in the final image. Everything the console
# serves is embedded in the binary through go:embed, so there is no asset
# pipeline and nothing to copy but the binary itself.

FROM --platform=$BUILDPLATFORM golang:1.23-alpine AS build

WORKDIR /src

# Dependencies first, so a source-only change does not re-download them.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG TARGETARCH
ARG TARGETOS=linux

# CGO is off so the binary is static and the runtime image needs no libc.
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /netcfgd .

RUN go vet ./... && go test ./...

FROM alpine:3.21

# ca-certificates is required: the proxy prober and the endpoint checks all
# complete real TLS handshakes, and without a trust store every one of them
# would report a certificate failure. tzdata keeps log timestamps sane.
RUN apk add --no-cache ca-certificates tzdata

COPY --from=build /netcfgd /usr/local/bin/netcfgd

# Configuration and watchdog state live here. Mount a named volume over it:
# the watchdog's record of what it has already applied is what stops it
# re-applying a proxy — and restarting every container — after each restart.
VOLUME /data
ENV NETCFG_DATA=/data \
    NETCFG_LISTEN=:8081

EXPOSE 8081

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s \
    CMD wget -qO- http://127.0.0.1:8081/healthz || exit 1

ENTRYPOINT ["/usr/local/bin/netcfgd"]
