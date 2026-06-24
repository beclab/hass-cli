# AGENTS.md

Working rules for this repository (the public release directory for hass-cli).

## Rules

1. **Single output directory** — all code and docs live in this `hass-cli/`
   repository. Nothing is written outside it.
2. **Commit per meaningful change** — make a separate git commit after each
   sizeable chunk of work, with a message focused on the "why".
3. **Local smoke testing** — exercise commands against a simulated HA before
   shipping. `go test ./...` runs an in-process mock Home Assistant
   (`smoke_test.go`) covering both REST and WebSocket transports. When a real
   instance is reachable, also smoke test against it (`hass-cli ping`).

## Architecture

- `main.go` — entrypoint; embeds skills (`skills_embed.go`).
- `cmd/` — small set of generic, schema-friendly verbs (cobra). Entity domains
  (light/climate/...) are NOT separate commands; they are driven by
  `service call` + the service schema, with knowledge living in skills.
- `internal/client/` — unified transport facade. Business methods route to REST
  (`rest.go`) or WebSocket (`ws.go`) per capability; callers never pick the
  transport. WS is lazily connected.
- `internal/config/` — flag < env (`HASS_SERVER`/`HASS_TOKEN`/
  `HASS_SUPERVISOR_TOKEN`) < profile resolution.
- `internal/output/` — table/json/yaml/ndjson rendering with `--columns`/`--sort-by`.
- `skills/` — agent skills (`ha-shared` first), embedded into the binary.

## Conventions

- Comments explain intent/constraints, not the obvious.
- Prefer `--output json` shapes that are stable for agent consumption.
- Mutating commands should support `--dry-run` as they are added.
