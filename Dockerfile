FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod ./
COPY *.go ./
RUN go build -o github-serve-flow .

FROM alpine:3.24

RUN apk add --no-cache ca-certificates

LABEL org.opencontainers.image.title="github-serve-flow" \
      org.opencontainers.image.description="Zero-dependency Go HTTP server that serves GitHub Actions artifact files from a disk cache with rate limiting." \
      org.opencontainers.image.ref.name="main" \
      org.opencontainers.image.licenses='MIT' \
      org.opencontainers.image.vendor="Timo Reymann <mail@timo-reymann.de>" \
      org.opencontainers.image.authors="Timo Reymann <mail@timo-reymann.de>" \
      org.opencontainers.image.url="https://github.com/timo-reymann/github-serve-flow" \
      org.opencontainers.image.documentation="https://github.com/timo-reymann/github-serve-flow" \
      org.opencontainers.image.source="https://github.com/timo-reymann/github-serve-flow.git"

COPY --from=builder /app/github-serve-flow /usr/local/bin/

EXPOSE 8080
ENTRYPOINT ["github-serve-flow"]
