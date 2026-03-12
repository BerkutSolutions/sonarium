# syntax=docker/dockerfile:1.7

FROM golang:1.23-alpine AS builder
WORKDIR /src

RUN apk add --no-cache ca-certificates tzdata

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/music-server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/scanner ./cmd/scanner

FROM alpine:3.21
WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata wget ffmpeg
RUN addgroup -S app && adduser -S app -G app

COPY --from=builder /out/music-server /app/music-server
COPY --from=builder /out/scanner /app/scanner
COPY --from=builder /src/internal/storage/migrations /app/migrations

ENV PORT=8080 \
    APP_ENV=production \
    LOG_LEVEL=info \
    MUSIC_LIBRARY_PATH=/music \
    DATA_PATH=/data \
    MIGRATIONS_DIR=/app/migrations

EXPOSE 8080

USER app

CMD ["/app/music-server"]
