ARG go_registry=""
ARG go_version=1.21
ARG go_tag_suffix=-alpine

FROM ${go_registry}golang:${go_version}${go_tag_suffix} AS builder
ARG TARGETARCH TARGETOS
ARG GOARCH=$TARGETARCH GOOS=$TARGETOS
ARG CGO_ENABLED=0
ARG GOFLAGS="-trimpath"
ARG go_ldflags="-s -w"
# Go needs git for `-buildvcs`, but the alpine version lacks git :( It's still
# way cheaper to `apk add git` than to pull the Debian-based golang image.
# hadolint ignore=DL3018
RUN apk add --no-cache git
WORKDIR /src
COPY . ./
RUN go build -ldflags="$go_ldflags" && \
    ./router -version && \
    go version -m ./router

FROM scratch
COPY --from=builder /src/router /bin/router
COPY --from=builder /usr/share/ca-certificates /usr/share/ca-certificates
COPY --from=builder /etc/ssl /etc/ssl
# TODO: remove rds-combined-ca-bundle.pem once app is using global-bundle.pem.
ADD https://s3.amazonaws.com/rds-downloads/rds-combined-ca-bundle.pem \
    /etc/ssl/certs/rds-combined-ca-bundle.pem
ADD https://truststore.pki.rds.amazonaws.com/global/global-bundle.pem \
    /etc/ssl/certs/rds-global-bundle.pem
USER 1001
CMD ["/bin/router"]
LABEL org.opencontainers.image.source="https://github.com/alphagov/router"
LABEL org.opencontainers.image.license=MIT
