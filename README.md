# WeVibe Social Graph

The WeVibe Social Graph service tracks relationships and trust edges across the
WeVibe Network. It exposes an HTTP API served by `cmd/server` and persists state
in a SQLite database located at the path specified by `SOCIAL_GRAPH_DB_PATH`.

## Development

- `go build ./...` — compile the service against the Go 1.25.9 toolchain.
- `go test ./...` — execute module tests (currently none defined).
- `go run ./cmd/server` — launch the API server with the local configuration.

## Docker

```
docker build -t wevibe-social-graph .
docker run -p 4470:4470 -e SOCIAL_GRAPH_DB_PATH=/data/social-graph.db \
  -v "$(pwd)/data:/data" wevibe-social-graph
```

## License

Apache-2.0. See `LICENSE` for details.
