FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod ./
COPY *.go ./
RUN go build -o github-serve-flow .

FROM alpine:3.20

LABEL org.opencontainers.image.title="github-serve-flow"
LABEL org.opencontainers.image.description="Zero-dependency Go HTTP server that serves GitHub Actions artifact files from a disk cache with rate limiting."
LABEL org.opencontainers.image.ref.name="main"
LABEL org.opencontainers.image.licenses='MIT'
LABEL org.opencontainers.image.vendor="Timo Reymann <mail@timo-reymann.de>"
LABEL org.opencontainers.image.authors="Timo Reymann <mail@timo-reymann.de>"
LABEL org.opencontainers.image.url="https://github.com/timo-reymann/github-serve-flow"
LABEL org.opencontainers.image.documentation="https://github.com/timo-reymann/github-serve-flow"
LABEL org.opencontainers.image.source="https://github.com/timo-reymann/github-serve-flow.git"

RUN apk add --no-cache ca-certificates
COPY --from=builder /app/github-serve-flow /usr/local/bin/
EXPOSE 8080
ENTRYPOINT ["github-serve-flow"]
