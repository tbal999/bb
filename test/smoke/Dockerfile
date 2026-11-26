# Multi-user smoke test: unit tests + a tiny fake pubnix (admin, alice, bob)
# where bb is driven as each user with real Linux permissions.
#
# Run from the repo root:  make smoke
#   (= docker build -f test/smoke/Dockerfile -t bb-smoke . && docker run --rm bb-smoke)
# Or play with it:          make demo
#   (= the same build, then docker run --rm -it bb-smoke bb-demo)

FROM golang:1.23-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go vet ./... && go test ./...
# admin is normally set in bb.go; override it for the test pubnix.
RUN CGO_ENABLED=0 go build -ldflags "-X main.admin=admin -X main.boardtitle=smoke-bb" -o /out/bb .

FROM debian:bookworm-slim
# Pubnix-style homes: everyone can read and traverse everyone else's home folder.
RUN apt-get update && apt-get install -y --no-install-recommends tmux && rm -rf /var/lib/apt/lists/*
RUN for u in admin alice bob; do useradd -m -s /bin/bash "$u"; done && chmod 755 /home/*
COPY --from=build /out/bb /usr/local/bin/bb
COPY test/smoke/smoke.sh /smoke.sh
COPY test/smoke/demo.sh /usr/local/bin/bb-demo
ENV TERM=xterm
CMD ["bash", "/smoke.sh"]
