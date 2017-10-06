FROM golang:1.9.1

ADD . /go/src/github.com/alphagov/router

RUN go install github.com/alphagov/router

ENV GOVUK_APP_NAME router
ENV ROUTER_PUBADDR :3054
ENV ROUTER_APIADDR :3055
ENV ROUTER_MONGO_URL mongo
ENV ROUTER_MONGO_DB router
ENV DEBUG true

CMD /go/bin/router
