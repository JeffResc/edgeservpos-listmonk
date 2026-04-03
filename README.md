# EdgeServPOS-Listmonk

A Go service that synchronizes customer data from [EdgeServPOS](https://www.edgeserv.com/) (a restaurant point-of-sale system) to [Listmonk](https://listmonk.app/) (a self-hosted email marketing platform).

## Overview

This service fetches customer records from EdgeServPOS and creates or updates corresponding subscribers in Listmonk. It handles data cleaning and normalization including:

- Phone number formatting (strips non-digits, keeps last 10)
- Email sanitization (removes spaces and commas)
- Name parsing (combines first/last with proper trimming)
- Last visit date conversion (epoch milliseconds to `YYYY-MM-DD` in Eastern Time)
- ZIP code extraction from customer addresses

Existing subscribers are only updated when their data has actually changed.

## Configuration

The service is configured entirely via environment variables:

| Variable | Description |
|---|---|
| `EDGESERV_POS_HOST` | EdgeServPOS API server URL |
| `RESTAURANT_CODE` | Restaurant identifier |
| `CLIENT_ID` | OAuth client ID for EdgeServPOS |
| `CLIENT_SECRET` | OAuth client secret for EdgeServPOS |
| `USERNAME` | EdgeServPOS username |
| `PASSWORD` | EdgeServPOS password |
| `LISTMONK_HOST` | Listmonk API server URL |
| `LISTMONK_USER` | Listmonk username |
| `LISTMONK_TOKEN` | Listmonk API token |

## Usage

### Run locally

```bash
# Set environment variables (or use a .env file)
export EDGESERV_POS_HOST=https://...
export RESTAURANT_CODE=...
# ... set all variables above

go run main.go
```

### Run with Docker

```bash
docker run --env-file .env ghcr.io/jeffresc/edgeservpos-listmonk:latest
```

### Build from source

```bash
go build -o edgeservpos-listmonk .
./edgeservpos-listmonk
```

## Development

**Prerequisites:** Go 1.25+

### Run tests

```bash
go test -v -race ./...
```

### Build Docker image

```bash
docker build -t edgeservpos-listmonk .
```

## CI/CD

GitHub Actions workflows handle:

- **CI** (on PRs to `main`): runs tests with race detection, reports coverage to Codecov, and validates the Docker build
- **Release** (on push to `main`): uses [release-please](https://github.com/googleapis/release-please) for automated semantic versioning, then builds and pushes the Docker image to GitHub Container Registry (GHCR)

## License

[MIT](LICENSE)
