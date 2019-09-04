FROM golang:1.12.9 AS builder
ADD . /go/src/github.com/alphagov/router
WORKDIR /go/src/github.com/alphagov/router
RUN CGO_ENABLED=0 go build -o router

FROM alpine:3.8
COPY --from=builder /go/src/github.com/alphagov/router/router /bin/router
RUN wget -O /etc/ssl/certs/aws-rds-ca.pem https://s3.amazonaws.com/rds-downloads/rds-combined-ca-bundle.pem
ENV GOVUK_APP_NAME router
ENV ROUTER_PUBADDR :3054
ENV ROUTER_APIADDR :3055
ENV ROUTER_MONGO_URL mongo
ENV ROUTER_MONGO_DB router
ENV DEBUG true
ENTRYPOINT ["/bin/router"]
CMD []
