# syntax=docker/dockerfile:1
FROM golang:1.24-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
COPY vendor ./vendor
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -trimpath -ldflags='-s -w' -o /out/pwndrop ./main.go

FROM alpine:3.22

RUN apk add --no-cache su-exec \
    && addgroup -S -g 10001 pwndrop \
    && adduser -S -D -H -u 10001 -G pwndrop pwndrop

WORKDIR /app
COPY --from=builder /out/pwndrop /app/pwndrop
COPY --from=builder /src/www /app/admin
COPY scripts/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh

EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
CMD ["-config", "/config/config.toml", "-no-autocert", "-no-dns"]
