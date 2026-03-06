FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /runeplan ./cmd/server

FROM alpine:3

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /runeplan /runeplan

EXPOSE 8080

ENTRYPOINT ["/runeplan"]
