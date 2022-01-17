FROM golang:1.17-alpine AS builder
WORKDIR /go/src/github.com/alexisgeoffrey/aoe4elobot
COPY go.mod ./
COPY go.sum ./
RUN go mod download
COPY *.go ./
RUN go build .

FROM alpine:latest
WORKDIR /
COPY --from=builder /go/src/github.com/alexisgeoffrey/aoe4elobot/aoe4elobot ./
ENTRYPOINT ["sh", "-c", "/aoe4elobot -t ${AOE4ELO_TOKEN} -g ${AOE4ELO_GUILD_ID}"]