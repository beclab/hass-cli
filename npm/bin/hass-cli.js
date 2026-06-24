#!/usr/bin/env node
'use strict';

const { spawnSync } = require('node:child_process');
const fs = require('node:fs');
const path = require('node:path');

// `npx @olares/hass-cli install` is a first-run wizard implemented purely in
// JS: it promotes the npx invocation to a global `npm install -g` and installs
// the agent skills via `npx skills add beclab/hass-cli`. It does NOT touch
// the Go binary, so it works even when the vendor binary failed to download.
// Every other verb is forwarded to the Go binary.
const args = process.argv.slice(2);
if (args[0] === 'install') {
  require('../scripts/install-wizard.js');
  return;
}

const isWindows = process.platform === 'win32';
const binName = isWindows ? 'hass-cli.exe' : 'hass-cli';
const bin = path.join(__dirname, '..', 'vendor', binName);

if (!fs.existsSync(bin)) {
  console.error(`[@olares/hass-cli] vendor binary not found at ${bin}`);
  console.error('[@olares/hass-cli] Re-run `npm install -g @olares/hass-cli` to repopulate it, or set');
  console.error('[@olares/hass-cli] HASS_CLI_DOWNLOAD_MIRROR / HASS_CLI_SKIP_DOWNLOAD if the');
  console.error('[@olares/hass-cli] postinstall step was skipped on purpose.');
  process.exit(1);
}

const res = spawnSync(bin, args, {
  stdio: 'inherit',
  windowsHide: true,
});

if (res.error) {
  console.error('[@olares/hass-cli] failed to spawn vendor binary:', res.error.message);
  process.exit(1);
}
process.exit(typeof res.status === 'number' ? res.status : 1);
