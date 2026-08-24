<div align="center">

<img src="https://capsule-render.vercel.app/api?type=waving&color=0:02100a,100:2fe07a&height=160&section=header&text=WeVibe%20Social%20Graph&fontColor=54f59a&fontSize=42&fontAlignY=40&desc=Public%20contributor%20profiles%20and%20reputation&descAlignY=64&descSize=16" alt="WeVibe Social Graph" width="100%" />

![Go](https://img.shields.io/badge/Go-00ADD8?style=flat-square&logo=go&logoColor=white)
[![status-alpha](https://img.shields.io/badge/status-alpha-ffc266?style=flat-square)](https://github.com/WeVibe-Network)
[![license-Apache--2.0](https://img.shields.io/badge/license-Apache--2.0-82aaff?style=flat-square)](LICENSE)
[![docs-wevibe-docs](https://img.shields.io/badge/docs-wevibe--docs-54f59a?style=flat-square)](https://github.com/WeVibe-Network/wevibe-docs)
[![%40WeVibe__Network](https://img.shields.io/badge/%40WeVibe__Network-0a0a0a?style=flat-square&logo=x&logoColor=white)](https://x.com/WeVibe_Network)

</div>

---

**A forkable, self-hostable display layer for WeVibe contributor profiles and reputation.**

## Overview

`wevibe-social-graph` is a Go microservice (`module github.com/wevibe-network/wevibe-social-graph`) that reads WeVibe chain state over REST and serves public contributor profiles and reputation stats over HTTP. The transport is REST only: it calls the chain's Cosmos SDK gRPC-gateway (port `1317`) with plain `net/http`. There is no gRPC client, no CometBFT RPC, and no websocket surface; the module's dependencies are just secp256k1/bech32 primitives, SQLite, and `x/crypto`.

**The chain is the source of truth; this service only renders it.** Reputation, contribution counts, and emissions aggregates are computed and stored on-chain. Anyone can self-host this read-only client against chain REST and get the same facts, and anyone can fork it — so no single operator controls how reputation is presented.

## What it is — and what it is not

Display-only. This service cannot write, curate, or authoritatively rank chain state: there is no transaction-signing path and no Cosmos SDK client in its dependencies. The only thing it writes is its own local SQLite database — a single `profiles` table (wallet, display name, avatar URL, timestamps) — which is non-authoritative presentation metadata, never chain truth. There are no organization endpoints and no org views.

## Data flow

Contributor stats are read **directly from chain REST** on each request — there is no hub projection in between:

- `GET /wevibe/reputation/v1/contributor/{pubkey}?epoch=0` — contributor reputation aggregates (x/reputation).
- `GET /wevibe/emissions/v1/contributor-reward/{pubkey}` — pending withdrawal and all-time earnings (x/emissions).

The service merges those two chain responses with the local display-name overlay (when one exists) and returns raw stats. It performs no server-side ranking or scoring.

## Badges: designed, not implemented

Three badge families are designed in WeVibe canon — **serve-milestone**, **outcome**, and **contribution-volume** — scoped per organization, with no cross-org leaderboard, and strictly non-economic (serve counts are a social signal, never an economic mechanism). The outcome family supersedes an earlier rarity-tier design.

**None of the families are implemented here.** The current code contains no badge fields, routes, or computation; this repo renders raw contributor stats (reputation + emissions aggregates read from the chain) only.

## Status: alpha

Built:

- HTTP server (JSON, CORS allowed for `*`).
- Local profile overlay: create / get / batch in SQLite; updates require a wallet signature over a canonical update message.
- Contributor stats endpoint backed by direct chain REST reads.

Design-stage (canon only, no code yet):

- Badge computation and rendering for the three families above.

## Endpoints

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/v1/health` | Liveness check. |
| `POST` | `/v1/profiles` | Create a local profile overlay (wallet, display name, optional avatar URL). |
| `GET` | `/v1/profiles/{wallet}` | Read one profile overlay. |
| `PATCH` | `/v1/profiles/{wallet}` | Update a profile overlay; requires wallet pubkey + signature over the canonical update message. |
| `GET` | `/v1/profiles/batch?wallets=a,b,c` | Read profile overlays in batch. |
| `GET` | `/v1/stats/contributor/{pubkey}` | Raw contributor stats: chain reputation + emissions aggregates, merged with the overlay display name when present. |

## Configuration

| Variable | Default | Purpose |
| --- | --- | --- |
| `SOCIAL_GRAPH_DB_PATH` | `/data/social-graph.db` | SQLite database path (local overlay only). |
| `SOCIAL_GRAPH_PORT` | `4470` | HTTP listen port. |
| `CHAIN_REST_URL` | `http://wevibe-chain:1317` | Chain REST base URL (Cosmos SDK gRPC-gateway). |

## Getting started

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

`internal/server/signature_test.go` unit-tests the wallet-signature path: secp256k1 address derivation (including Cosmos bech32 encoding) and Cosmos arbitrary-message signature verification (happy path, tampered message, wrong pubkey, malformed input).

## Roadmap

See [ROADMAP.md](./ROADMAP.md).

## License

Apache-2.0. See [LICENSE](./LICENSE).

## Links

- Docs: https://github.com/WeVibe-Network/wevibe-docs
- Organization: https://github.com/WeVibe-Network
- X: https://x.com/WeVibe_Network
