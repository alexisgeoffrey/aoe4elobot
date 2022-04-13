FROM golang:1.18-alpine AS builder
WORKDIR /go/src/github.com/alexisgeoffrey/aoe4elobot
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN go build ./cmd/aoe4elobot

FROM alpine:latest
WORKDIR /app
COPY --from=builder /go/src/github.com/alexisgeoffrey/aoe4elobot/aoe4elobot ./
USER 1000:1000
ENTRYPOINT ["sh", "-c", "/app/aoe4elobot"]
