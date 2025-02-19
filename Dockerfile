FROM golang:1.23.6 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o userinfo cmd/main.go

FROM debian:bookworm-slim

WORKDIR /app

COPY --from=builder /app/userinfo /app/userinfo

COPY .env /app/.env

ENV CONFIG_PATH=/app/.env

CMD ["/app/userinfo"]
