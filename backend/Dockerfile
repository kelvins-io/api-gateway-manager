FROM golang:1.22-alpine AS builder

WORKDIR /src

RUN apk add --no-cache ca-certificates git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/server ./cmd/server

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata wget \
    && adduser -D -H -u 10001 appuser

WORKDIR /app

COPY --from=builder /out/server /app/server
COPY configs/config.docker.yaml /app/configs/config.yaml

USER appuser

EXPOSE 10000

HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=5 \
  CMD wget -qO- http://127.0.0.1:10000/health || exit 1

ENTRYPOINT ["/app/server", "-config", "/app/configs/config.yaml"]
