<div align="center">

<img src="https://capsule-render.vercel.app/api?type=waving&color=0:02100a,100:2fe07a&height=160&section=header&text=WeVibe%20Social%20Graph&fontColor=54f59a&fontSize=42&fontAlignY=40&desc=Public%20profiles%20reputation%20and%20badges&descAlignY=64&descSize=16" alt="WeVibe Social Graph" width="100%" />

![Go](https://img.shields.io/badge/Go-00ADD8?style=flat-square&logo=go&logoColor=white)
[![status-alpha](https://img.shields.io/badge/status-alpha-ffc266?style=flat-square)](https://github.com/WeVibe-Network)
[![license-Apache--2.0](https://img.shields.io/badge/license-Apache--2.0-82aaff?style=flat-square)](LICENSE)
[![docs-wevibe-docs](https://img.shields.io/badge/docs-wevibe--docs-54f59a?style=flat-square)](https://github.com/WeVibe-Network/wevibe-docs)
[![%40WeVibe__Network](https://img.shields.io/badge/%40WeVibe__Network-0a0a0a?style=flat-square&logo=x&logoColor=white)](https://x.com/WeVibe_Network)

</div>

---

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
