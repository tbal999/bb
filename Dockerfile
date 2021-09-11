FROM golang:1.16-alpine

RUN set -ex; \
    apk update; \
    apk add --no-cache git
RUN apk add build-base

COPY . .

ENV GOPATH=""

RUN go mod tidy

CMD CGO_ENABLED=1 go test -v ./... $(go list ./...)
