
# docker run -d \
#	--restart=unless-stopped \
#	--mount type=bind,source=/home/pi/Servers/DiscordBots/herbie/feeds.db,target=/app/feeds.db \
#	--mount type=bind,source=/home/pi/Servers/DiscordBots/herbie/herbie.quotes,target=/app/herbie.quotes \
#	herbie

FROM golang:1.19-alpine3.16 AS build-go

WORKDIR /app

RUN apk --no-cache add git gcc musl-dev

COPY . .

WORKDIR /app/herbie

RUN go build -o ../herbie.bin

########################################################################################################################

FROM alpine:3.16

RUN apk --no-cache add musl

WORKDIR /app

COPY --from=build-go /app/herbie.bin .

ENTRYPOINT ["/app/herbie.bin"]
