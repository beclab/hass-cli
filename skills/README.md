# hass-cli skills

LLM-readable skills, one folder per command tree or task family, that teach an agent to drive `hass-cli` against a live Home Assistant. They are compiled into the binary and reach a machine through it: `hass-cli skill list`, `skill show <name>`, or `skill export <dir>` for the whole tree including `references/`.

These rules follow the olares-cli skill suite (`Olares/cli/skills/README.md`), which reached them by the same route this suite was on: patching each observed failure into whichever file the agent happened to be reading. `validate.py` enforces what a machine can check; the rest is review.

## Layout

```
skills/
├── README.md            # this file
├── validate.py          # the checks, run in CI
├── stamp.py             # writes the release into every frontmatter
├── ha-shared/           # front door: every other skill reads it first
│   ├── SKILL.md
│   └── references/      # connection, auth, raw access: read only when triggered
├── ha-lighting/
│   └── SKILL.md
├── ...
└── ha-workflow-automation-builder/
    └── SKILL.md         # playbook: a recipe over several command trees
```

**Install the whole suite together.** Skills link each other by relative path (`../ha-shared/SKILL.md`); a subset leaves those links dangling. Use `skill export`, which writes the tree as it was compiled.

## The front door

`ha-shared/SKILL.md` is read before any other skill, so everything that more than one skill needs lives there once: how to find an entity and call a service on it, when a call needs checking afterwards, which skill owns which request, and the rules no domain skill may contradict (a device a script or automation is driving is stopped there first). Every other skill opens with:

```markdown
> **Shared front door:** read [`../ha-shared/SKILL.md`](../ha-shared/SKILL.md) first; it holds the two-command control and the rules every request follows.
```

A domain skill then carries only what is its own: colour modes for lights, HVAC modes for climate, the shape of a script for automation. It does not repeat the two commands, the confirmation rule, or the suite map. A rule copied into N skills drifts in N places; that is how the "do not check the state afterwards" line ended up in two files with neither saying when it does not hold.

Connection, auth recovery and raw REST/WebSocket access are in `ha-shared/references/`, because a request that connects needs none of them.

## Writing style

`SKILL.md` is the agent's decision layer, not a command manual. Flag syntax is `hass-cli <command> --help`'s job. A skill keeps what changes the agent's next step:

- **When to use**: the trigger scope and the one or two neighbouring skills most easily confused with it.
- **Fast paths**: the first table. It routes one intent (what the user asked for, in their words: "make the light cycle colours", "stop that") to the files to read and the command it ends at. An agent arrives holding a goal, not a verb.
- **Verb index**: the command tree, second, under `## Verb index`, whose last column is `Read when triggered`. Every row must change a decision; a row that only says `<cmd> --help` is collapsed into one "everything else is `--help`" row.
- **Error → fix**: only what the CLI output cannot resolve by itself. If stderr already names the cause and the next action, do not copy it.
- **Safety and escalation**: closes every skill under that name. Missing input, ambiguous targets, actions beyond what was asked.

Three names are fixed and checked: `## Fast paths`, `## Verb index`, `## Safety and escalation`, plus the front-door block above.

**Point at a section with a link, never with its name in prose.** `(see "Quick control")` resolves for nobody and survives no rename; `[quick control](../ha-shared/SKILL.md#control-a-device)` fails the build when the heading moves.

### CLI and skill responsibilities

- A fact known at the moment of a request belongs in the CLI's output: what failed, why, and the next command. `service call` naming the script that is still driving a light it just turned off is one; a wrong save id refused locally with the right form is another.
- Skills carry what spans commands: multi-step flows, Home Assistant's model (script id versus entity id, UI versus YAML storage), and safety decisions.
- A CLI message and the skill cleanup it allows ship together. Do not delete guidance until the released CLI says it.

### Ground every claim in the implementation

Before documenting what a verb does, read the command. `workflow script trigger` was documented as working for months while it called a service Home Assistant does not have. Keep the verification in the commit message; no Go source paths in what ships (`validate.py` refuses them).

**One representation per fact within a file.** Do not encode the same thing in two tables.

## Sizes and the read path

Per-file ceilings: `SKILL.md` ≤ 250 lines, each reference ≤ 150.

The budget an agent pays is the **read path**: `ha-shared/SKILL.md`, plus the domain `SKILL.md`, plus the references a task triggers, everything before its first correct command. Each row of `## Fast paths` is totalled by `validate.py` and held to **250 lines**. Every voice request pays this path, so keep the front door short.

**Split references on triggers, not on line count.** A reference serves one trigger. Two questions an agent would never ask on the same task are two files, even at 40 lines.

**One level deep.** Every reference is linked from its own `SKILL.md`. A reference may point at a sibling reference of the same skill. A skill reaches a peer through the peer's `SKILL.md`, never its references; `ha-shared`'s references are the one exception, and only from a `SKILL.md`.

No table of contents: the `##` headings are the structure.

What to leave out: per-flag descriptions, error strings the CLI already prints, Go source paths, "Source layout" walkthroughs, "not implemented yet" sections.

## Frontmatter

```yaml
---
name: ha-lighting                      # the folder name
version: "0.0.0"                       # placeholder; the release stamps it
description: "One line: what it does and the requests it is for."
compatibility: Requires hass-cli on PATH
metadata:
  requires:
    bins: ["hass-cli"]
---
```

- `description` is one physical line (`hass-cli skill list` reads one line per field) and at most 1024 characters. Every agent request lists all of them, so front-load the words a user says ("loop", "every few seconds", "stop the script") and keep it to a sentence or two.
- `version` is the hass-cli release the suite ships in. Do not edit it: what is committed is `0.0.0`, and GoReleaser runs `stamp.py` with the release version before compiling, so a binary and its skills always agree.

## Validating

```bash
python3 -m pip install -r skills/requirements.txt
python3 -m unittest skills/test_validate.py
python3 skills/validate.py
```

CI runs all three. `validate.py` checks frontmatter, link and anchor targets, per-file ceilings, deep links into a peer's references, sections named in prose, verb-index rows that only say `--help`, the three fixed headings, the fast-path budget, a single suite version, and a few pinned facts that once went missing.

## Names

Command-tree skills mirror the command (`ha-states`, `ha-registry`, `ha-system`). Playbooks that orchestrate several trees carry a descriptive suffix (`ha-workflow-audit`, `ha-workflow-automation-builder`). Renaming a skill means updating every link to it.
