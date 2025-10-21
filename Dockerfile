FROM golang:1.24 AS builder

ENV GOOS=linux
ENV GOARCH=amd64
ENV CGO_ENABLED=0

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o bot ./cmd/main.go

FROM scratch AS telegram-bot

WORKDIR /

COPY --from=builder /app/bot /bot

ENTRYPOINT ["/bot"]