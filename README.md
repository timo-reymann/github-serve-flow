github-serve-flow
===
[![LICENSE](https://img.shields.io/github/license/timo-reymann/github-serve-flow)](https://github.com/timo-reymann/github-serve-flow/blob/main/LICENSE)
[![GitHub Actions](https://github.com/timo-reymann/github-serve-flow/actions/workflows/main.yml/badge.svg)](https://github.com/timo-reymann/github-serve-flow/actions/workflows/main.yml)
[![GitHub Release](https://img.shields.io/github/v/tag/timo-reymann/github-serve-flow?label=version)](https://github.com/timo-reymann/github-serve-flow/releases)

<p align="center">
    Zero-dependency Go HTTP server that serves GitHub Actions artifact files from a disk cache with rate limiting.
</p>

## Features
- Serves individual files from GitHub Actions artifact zips via HTTP
- Automatic artifact download and disk caching with TTL and size-based eviction
- Per-IP sliding window rate limiting
- Owner allowlist for access control
- Correct content-type detection
- Client-side cache headers
- Concurrent fetch deduplication

## Requirements
- Go 1.22+ (for build)
- GitHub personal access token with `actions:read` scope
- Docker (optional, for container deployment)

## Installation

### From source
```bash
go build -o github-serve-flow .
```

### Docker
```bash
docker build -t github-serve-flow .
docker run -e GITHUB_TOKEN=ghp_... -e ALLOWED_OWNERS=your-org -p 8080:8080 github-serve-flow
```

## Usage

### Configuration

All configuration is via environment variables:

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `GITHUB_TOKEN` | Yes | - | GitHub personal access token |
| `ALLOWED_OWNERS` | Yes | - | Comma-separated list of allowed GitHub owners |
| `LISTEN_ADDR` | No | `:8080` | Server listen address |
| `CACHE_DIR` | No | `$TMPDIR/github-serve-flow` | Disk cache directory |
| `CACHE_TTL` | No | `1h` | Cache entry time-to-live |
| `CACHE_MAX_SIZE` | No | `5GB` | Maximum cache size on disk |
| `CLIENT_CACHE_MAX_AGE` | No | `3600` | Cache-Control max-age in seconds |
| `RATE_LIMIT_WINDOW` | No | `1m` | Rate limit sliding window |
| `RATE_LIMIT_MAX` | No | `60` | Max requests per IP per window |

### Running

```bash
export GITHUB_TOKEN="ghp_your_token"
export ALLOWED_OWNERS="your-github-username"

go run .
```

### Endpoints

```
GET /health
GET /{owner}/{repo}/actions/runs/{runId}/artifacts/{artifactName}/{filePath}
```

Example:
```bash
curl http://localhost:8080/your-org/your-repo/actions/runs/12345/artifacts/my-site/index.html
```

## Contributing
I love your input! I want to make contributing to this project as easy and transparent as possible, whether it's:

- Reporting a bug
- Discussing the current state of the configuration
- Submitting a fix
- Proposing new features
- Becoming a maintainer

To get started please read the [Contribution Guidelines](./CONTRIBUTING.md).

## Development

### Requirements
- [Go 1.22+](https://golang.org/dl/)
- [GNU make](https://www.gnu.org/software/make/)
- [Docker](https://docs.docker.com/get-docker/) (optional)

### Test
```bash
make test
```

### Build
```bash
make build
```

### Coverage
```bash
make cover
```
