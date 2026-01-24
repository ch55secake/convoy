FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags '-extldflags "-static"' -o convoy-agent ./cmd/agent

FROM alpine:latest
RUN apk add --no-cache bash curl libgcc libstdc++ ripgrep supervisor
RUN curl -fsSL https://opencode.ai/install | bash
WORKDIR /app
COPY --from=builder /app/convoy-agent .
RUN mkdir -p /root/.config/convoy
COPY configs/agent.yaml /root/.config/convoy/agent.yaml
COPY configs/supervisord.conf /etc/supervisord.conf
EXPOSE 6000
CMD ["supervisord", "-c", "/etc/supervisord.conf"]
