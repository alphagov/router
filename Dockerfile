FROM golang:1.20.5-alpine AS builder
ARG TARGETARCH TARGETOS
WORKDIR /src
COPY . ./
RUN CGO_ENABLED=0 GOARCH=$TARGETARCH GOOS=$TARGETOS go build -trimpath -ldflags="-s -w"

FROM scratch
COPY --from=builder /src/router-postgres /bin/router-postgres
COPY --from=builder /usr/share/ca-certificates /usr/share/ca-certificates
COPY --from=builder /etc/ssl /etc/ssl
ADD https://s3.amazonaws.com/rds-downloads/rds-combined-ca-bundle.pem \
    /etc/ssl/certs/rds-combined-ca-bundle.pem
USER 1001
CMD ["/bin/router-postgres"]
LABEL org.opencontainers.image.source="https://github.com/alphagov/router-postgres"
LABEL org.opencontainers.image.license=MIT
