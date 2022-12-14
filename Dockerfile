
FROM golang:1.19-alpine3.16 AS build-go

WORKDIR /app

RUN apk --no-cache add git

COPY . .

RUN go build -o herbie.bin

########################################################################################################################

FROM alpine:3.16

WORKDIR /app

COPY --from=build-go /app/herbie.bin .

ENTRYPOINT ["/app/herbie.bin"]
