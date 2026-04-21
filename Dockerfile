# Stage 1: Build
FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
# RUN go build -o uptime-monitor ./cmd/server
RUN CGO_ENABLED=0 go build -o uptime-monitor ./cmd/server

# Stage 2: Run
FROM alpine:latest

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /app/uptime-monitor .
COPY --from=builder /app/migrations ./migrations

USER appuser

EXPOSE 8080

CMD ["./uptime-monitor"]