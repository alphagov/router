FROM golang:1.18.3 AS builder
ADD . /go/src/github.com/alphagov/router
WORKDIR /go/src/github.com/alphagov/router
RUN CGO_ENABLED=0 go build -o router

FROM alpine:3.15.0
COPY --from=builder /go/src/github.com/alphagov/router/router /bin/router
RUN wget -O /etc/ssl/certs/rds-combined-ca-bundle.pem https://s3.amazonaws.com/rds-downloads/rds-combined-ca-bundle.pem
ENV GOVUK_APP_NAME router
ENV ROUTER_PUBADDR :3054
ENV ROUTER_APIADDR :3055
ENV ROUTER_MONGO_URL mongo
ENV ROUTER_MONGO_DB router
CMD ["/bin/router"]
