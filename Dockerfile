FROM golang:1.19.4-alpine AS builder
ARG TARGETARCH TARGETOS
WORKDIR /src
COPY . ./
RUN CGO_ENABLED=0 GOARCH=$TARGETARCH GOOS=$TARGETOS go build -trimpath -ldflags="-s -w"

FROM scratch
COPY --from=builder /src/router /bin/router
ADD https://s3.amazonaws.com/rds-downloads/rds-combined-ca-bundle.pem \
    /etc/ssl/certs/rds-combined-ca-bundle.pem
USER 1001
CMD ["/bin/router"]
LABEL org.opencontainers.image.source="https://github.com/alphagov/router"
LABEL org.opencontainers.image.license=MIT
