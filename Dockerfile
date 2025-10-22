FROM alpine:3.20 AS certs
RUN apk add --no-cache ca-certificates

FROM golang:1.24 AS builder
ENV GOOS=linux GOARCH=amd64 CGO_ENABLED=0
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /out/bot ./cmd/main.go

FROM scratch AS telegram-bot

COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /out/bot /bot
ENTRYPOINT ["/bot"]