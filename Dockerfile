# Stage 1: Build
FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o uptime-monitor ./cmd/server

# Stage 2: Run
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/uptime-monitor .
COPY --from=builder /app/migrations ./migrations

EXPOSE 8080

CMD ["./uptime-monitor"]