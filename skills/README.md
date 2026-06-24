# hass-cli skills

LLM-readable skills (one folder per command tree) that teach a Claude-style
agent how to drive `hass-cli` against a live Home Assistant instance. Each
folder contains a single `SKILL.md` whose YAML front-matter declares the
skill's name, description, version, and runtime requirements.

The bundled copy ships inside the binary and is browsable with
`hass-cli skill list` / `hass-cli skill show <name>`.

## Layout

```
skills/
├── README.md          # this file (authoring spec)
├── ha-shared/
│   └── SKILL.md        # foundation: connection, transport, output, verb suite map, error recovery
├── ha-states/
│   └── SKILL.md
├── ha-services/
│   └── SKILL.md
├── ...                 # one folder per command tree (see the suite map in ha-shared)
└── ha-workflow-automation-builder/
    └── SKILL.md        # playbook skill (multi-step recipe, not a 1:1 command wrapper)

# Optional, when a single SKILL.md would exceed its size budget:
ha-<area>/
├── SKILL.md
└── references/
    ├── ha-<area>-<verb>.md
    └── ...
```

**Install the whole suite together.** The skills are designed as one set and
cross-reference each other by relative path (e.g. `../ha-shared/SKILL.md`).
Installing only a subset leaves those links dangling. `ha-shared` is the
foundation: every other skill assumes it for connection setup, transport
routing, the `--data` quoting rules, and auth-error recovery.

## Writing style

`SKILL.md` is NOT a copy of `hass-cli <command> --help`. The body defers to
`--help` for authoritative flag syntax and only carries what `--help` cannot:

- **When to use** — the skill's scope / trigger phrases, plus a pointer to the
  suite map in `ha-shared` for outbound routing.
- **Cross-cutting concepts** referenced by ≥2 subcommands (e.g. the WebSocket
  `config/*_registry/*` model, the `@file` `--data` convention).
- **Client-side constraints that bite users** (server-side auto-rename traps,
  required fields the API rejects without, Supervisor-only commands, …).
- **Error → fix matrix** that is not in `--help`.
- **Verb index** — one row per verb pointing at `--help` and (if it exists)
  `references/<skill>-<verb>.md`.

What to leave out:

- Per-flag descriptions (they live in `--help`).
- Go source-path citations like `cmd/state.go` — agents don't read the source.
- Internal package walkthroughs or "Source layout" sections.
- "What's NOT here yet" / future-work sections — keep skills to current scope.

Target sizes: `SKILL.md` ≤ 250 lines (≤ 300 for the most complex command tree);
each reference ≤ 150 lines.

**Reference depth (one level deep):** reference files link **directly from a
`SKILL.md`**, never reference→reference. A concept buried two hops down is
unreliable because agents preview deep files instead of reading them whole. Any
reference longer than ~100 lines gets a table of contents at the top.

## Single source of truth

Facts used by **≥2 skills** are defined **once** and linked, never copied:

- **Connection, transport routing, output formats, `--data` quoting, and
  auth-error recovery** live once in [`ha-shared/SKILL.md`](ha-shared/SKILL.md).
  Every runtime skill assumes it as a prerequisite.
- **Routing has one home too: the verb / skill suite map** in `ha-shared`. The
  canonical intent → command → skill table for the whole suite lives there. Each
  skill folds routing into its `## When to use` section (trigger phrases + a
  pointer to the suite map) rather than repeating the table.
- When a fact is genuinely two skills' own angle, let each keep its framing
  (e.g. `ha-registry` describes a device as a registry row; `ha-gateway`
  describes the same device as a radio node). Only the shared mechanics are
  centralized.

## Front-matter contract

`hass-cli skill list` reads `description` from the front-matter, and the loader
([`cmd/skill.go`](../cmd/skill.go)) expects a YAML block fenced by `---`. Each
`SKILL.md` declares:

```yaml
---
name: ha-states                       # required; must match the folder slug
version: 0.1.0                        # recommended; semver, bump per release
description: "..."                    # required; ≤ 1024 chars, front-loaded with triggers
metadata:
  requires:
    bins: ["hass-cli"]                # gates the skill behind hass-cli on PATH
  cliHelp: "hass-cli state --help"    # the command tree this skill documents
---
```

Rules:

- `name` MUST equal the folder name (the loader keys skills by folder).
- `description` MUST be a single-line scalar ≤ 1024 characters. The loader's
  front-matter parser reads one line per field, so do not wrap `description`
  across lines. Put long trigger lists in the body's `## When to use`.
- `version` is semver; bump it when the skill's guidance changes materially.
- `metadata.requires.bins` MUST include `hass-cli`. The binary is not installed
  by a skill registry — this line just gates the skill behind "you are on a host
  that has `hass-cli` on PATH". Install the binary via npx / install script /
  release archive (see the repo README).

## Naming

- Command-tree skills mirror the command: `ha-<command>` (e.g. `ha-state` →
  `ha-states`, `ha-registry`, `ha-system`). Where the noun reads better
  pluralized, prefer the existing folder name over renaming.
- Playbook skills (multi-step recipes that orchestrate several commands rather
  than wrapping one) use a descriptive suffix: `ha-workflow-audit`,
  `ha-workflow-automation-builder`.
