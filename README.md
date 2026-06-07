# WeVibe Social Graph

**Forkable, self-hostable public display service for WeVibe contributor and org views.**

## Overview

`wevibe-social-graph` is a Go + SQLite microservice (`module github.com/wevibe-network/wevibe-social-graph`) that reads chain state via RPC/REST and serves public profile and contributor summary data over HTTP.

Status: **alpha**. Badge rendering and rarity-tier logic are still in alpha/design stage. In this phase, badge families are computed at read time from chain RPC inputs (there is no on-chain badge entity yet).

## Role in the WeVibe Network

This service is a **presentation layer**, not a consensus system. The chain remains the source of truth.

Design constraints reflected in the current implementation and roadmap:
- strict per-org breakdowns
- **no cross-org leaderboard**
- serve counts are a **social signal only**, not an economic mechanism

## Getting started (build/run)

```sh
go build ./...
go run ./cmd/server
```

Docker:

```sh
docker build -t wevibe-social-graph .
docker run -p 4470:4470 -e SOCIAL_GRAPH_DB_PATH=/data/social-graph.db \
  -v "$(pwd)/data:/data" wevibe-social-graph
```

## Testing

```sh
go test ./...
```

At this time, no `*_test.go` module tests are defined yet.

## Configuration (environment and ports)

- `SOCIAL_GRAPH_DB_PATH` (default: `/data/social-graph.db`) — SQLite database path.
- `SOCIAL_GRAPH_PORT` (default: `4470`) — HTTP listen port.
- `CHAIN_REST_URL` (default: `http://wevibe-chain:1317`) — chain REST base URL used for reads.
- Health endpoint: `GET /v1/health`.

## Roadmap

See [ROADMAP.md](./ROADMAP.md).

## License

Apache-2.0. See [LICENSE](./LICENSE).

## Links

- Docs: https://github.com/WeVibe-Network/wevibe-docs
- Organization: https://github.com/WeVibe-Network
- X: https://x.com/WeVibe_Network
