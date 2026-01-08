# Multi-stage build for convoy container
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags '-extldflags "-static"' -o server ./cmd/orchestrator

FROM alpine:latest
RUN apk add --no-cache supervisor
WORKDIR /app
COPY --from=builder /app/server .
COPY configs/supervisord.conf /etc/supervisord.conf
EXPOSE 50051
CMD ["supervisord", "-c", "/etc/supervisord.conf"]