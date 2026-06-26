# syntax=docker/dockerfile:1.7

ARG GO_VERSION=1.26

FROM golang:${GO_VERSION}-alpine AS builder
WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/web ./cmd/web

FROM alpine:3.22
WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /out/web /usr/local/bin/web
COPY --from=builder /app/templates /app/templates

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/web"]
