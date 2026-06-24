#!/usr/bin/env node
'use strict';

const v = require('../package.json').version;

if (v === '0.0.0-placeholder') {
  console.error(`[@olares/hass-cli] refusing to publish placeholder version "${v}".`);
  console.error('[@olares/hass-cli] CI must run "npm version <semver>" before "npm publish".');
  console.error('[@olares/hass-cli] Locally, do NOT publish from this clone.');
  process.exit(1);
}
