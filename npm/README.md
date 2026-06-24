# @bytetrade/hass-cli

Node wrapper for **hass-cli**, a command-line interface for [Home Assistant](https://www.home-assistant.io/).

This package ships a small JavaScript shim plus a `postinstall` step that
downloads the matching precompiled Go binary from the
[GitHub releases](https://github.com/bytetrade/hass-cli/releases) and runs it.

## Quick start

```bash
# one-off, no install:
npx @bytetrade/hass-cli@latest --help

# install globally:
npm install -g @bytetrade/hass-cli
hass-cli profile login   # save your Home Assistant URL + token
hass-cli ping            # verify connectivity
```

## First-run wizard

```bash
npx @bytetrade/hass-cli@latest install
```

This installs the CLI globally and adds the bundled AI agent skills
(`npx skills add bytetrade/hass-cli`), then points you at `hass-cli profile login`.

## Supported platforms

`linux`, `darwin`, `win32` on `x64`, `arm64` (and `arm` on linux).

## Environment variables

| Variable | Effect |
|---|---|
| `HASS_CLI_DOWNLOAD_MIRROR` | Base URL to fetch the release tarball from instead of GitHub. |
| `HASS_CLI_SKIP_DOWNLOAD=1` | Install the JS shim without downloading a binary (CI / offline). |

## Configuration

hass-cli reads connection settings from a saved profile (token stored in the OS
keychain), then `HASS_SERVER` / `HASS_TOKEN` environment variables, then
`--server` / `--token` flags. Run `hass-cli profile login` for guided setup, or
see the [main README](https://github.com/bytetrade/hass-cli).
