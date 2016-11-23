FROM golang:1.7

ADD . /go/src/github.com/alphagov/router

RUN go get github.com/tools/godep

RUN go install github.com/alphagov/router
