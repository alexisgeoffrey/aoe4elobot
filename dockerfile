FROM golang:1.18-alpine AS builder
WORKDIR /go/src/github.com/alexisgeoffrey/aoe4elobot
RUN apk --update add ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build ./cmd/aoe4elobot

FROM scratch
WORKDIR /app
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /go/src/github.com/alexisgeoffrey/aoe4elobot/aoe4elobot ./
USER 1000:1000
ENTRYPOINT ["./aoe4elobot"]
