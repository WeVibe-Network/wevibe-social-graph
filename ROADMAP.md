# WeVibe Social Graph Roadmap

## Status

- Alpha-stage public display service backed by Go + SQLite.
- Reads contributor and profile data from chain RPC/REST and presents it through HTTP APIs.
- Badge rendering and rarity-tier logic are still alpha/design-stage.
- In alpha, all badge families are computed at read time from chain RPC inputs (no on-chain badge entity yet).
- Serve counts are treated as a social signal only, never an economic value.
- There is no cross-org leaderboard.

## Near-term

- Render serve-milestone badges from chain RPC inputs.
- Render contribution-volume badges from chain RPC inputs.
- Render read-time rarity badges from chain RPC inputs.
- Add per-field graded-provenance display, strictly as presentation metadata and never coupled to VIBE.

## Mainnet

- Freeze rarity-tier logic on-chain.
- Retune provisional alpha thresholds to finalized network thresholds.

## Future

- In v2, consume attested session claims as graded provenance once the attestation framework is available.

## Design references

- WeVibe documentation: https://github.com/WeVibe-Network/wevibe-docs
