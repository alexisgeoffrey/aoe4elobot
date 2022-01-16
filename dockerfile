FROM golang:1.17-alpine

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY *.go ./

RUN go build -o /aoe4elobot

ENTRYPOINT ["sh", "-c", "/aoe4elobot -t ${AOE4ELO_TOKEN} -g ${AOE4ELO_GUILD_ID}"]