FROM golang:1.23-alpine

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .

CMD ["sh", "-c", "go vet ./... && go test -v ./..."]
