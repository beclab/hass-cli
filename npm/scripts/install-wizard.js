#!/usr/bin/env node
'use strict';

const { execFileSync, execFile } = require('node:child_process');
const p = require('@clack/prompts');

const PKG = '@bytetrade/hass-cli';
const SKILLS_REPO = 'bytetrade/hass-cli';
const isWindows = process.platform === 'win32';

const msg = {
  setup:        'Setting up hass-cli...',
  step1:        'Installing %s globally...',
  step1Upgrade: 'Upgrading %s (v%s -> v%s)...',
  step1Skip:    'Already installed (v%s). Skipped',
  step1Done:    'Installed globally',
  step1Upgraded:'Upgraded to v%s',
  step1Fail:    'Failed to install globally. Run manually: npm install -g %s',
  step1EaccesHint:
    'npm install -g hit EACCES while writing to its global prefix.\n' +
    'On distro-packaged Node the prefix is typically /usr or /usr/local and needs root.\n' +
    'Recommended one-time fix: switch npm to a user-owned prefix so global installs do not need sudo:\n' +
    '  mkdir -p ~/.npm-global\n' +
    '  npm config set prefix ~/.npm-global\n' +
    '  echo \'export PATH="$HOME/.npm-global/bin:$PATH"\' >> ~/.bashrc\n' +
    '  export PATH="$HOME/.npm-global/bin:$PATH"\n' +
    'then re-run:  npx -y %s@latest install',
  step1TimeoutHint:
    'Timed out after 10 min. Likely cause: slow / proxied connection while downloading the Go binary from github.com.\n' +
    'Retry outside the wizard so you can watch progress: npm install -g %s',
  step2Spinner: 'Installing AI skills...',
  step2Skip:    'Skills already installed. Skipped',
  step2Done:    'Skills installed',
  step2Fail:    'Failed to install skills. Run manually: npx skills add %s -y -g',
  done:
    'You are all set!\n\n' +
    'Next:\n' +
    '  hass-cli init             # save your Home Assistant URL + token\n' +
    '  hass-cli ping             # verify connectivity\n\n' +
    'Then tell your AI agent: "Load the ha-shared skill, then use hass-cli to ..."',
  nonTtyHint:
    'To finish setup, run:\n' +
    '  hass-cli init\n' +
    '  hass-cli ping',
};

function fmt(template, ...values) {
  let i = 0;
  return template.replace(/%s/g, () => values[i++] ?? '');
}

function execCmd(cmd, args, opts) {
  if (isWindows) {
    return execFileSync('cmd.exe', ['/c', cmd, ...args], opts);
  }
  return execFileSync(cmd, args, opts);
}

function runSilent(cmd, args, opts = {}) {
  return execCmd(cmd, args, { stdio: ['ignore', 'pipe', 'pipe'], ...opts });
}

function runSilentAsync(cmd, args, opts = {}) {
  const actualCmd = isWindows ? 'cmd.exe' : cmd;
  const actualArgs = isWindows ? ['/c', cmd, ...args] : args;
  return new Promise((resolve, reject) => {
    execFile(actualCmd, actualArgs, { stdio: ['ignore', 'pipe', 'pipe'], ...opts }, (err, stdout, stderr) => {
      if (err) {
        err.stderr = stderr;
        err.stdout = stdout;
        reject(err);
      } else {
        resolve(stdout);
      }
    });
  });
}

function getLatestVersion() {
  try {
    const out = runSilent('npm', ['view', PKG, 'version'], { timeout: 15000 });
    const ver = out.toString().trim();
    return /^\d+\.\d+\.\d+/.test(ver) ? ver : null;
  } catch (_) {
    return null;
  }
}

function getGloballyInstalledVersion() {
  try {
    const out = runSilent('npm', ['list', '-g', PKG], { timeout: 15000 });
    const match = out.toString().match(/hass-cli@(\d+\.\d+\.\d+[^\s]*)/);
    return match ? match[1] : null;
  } catch (_) {
    return null;
  }
}

async function stepInstallGlobally(interactive) {
  const installedVer = getGloballyInstalledVersion();
  const latestVer = getLatestVersion();
  const needsUpgrade = installedVer && latestVer && installedVer !== latestVer;

  if (installedVer && !needsUpgrade) {
    const line = fmt(msg.step1Skip, installedVer);
    if (interactive) p.log.info(line); else console.log(line);
    return;
  }

  const startLine = needsUpgrade ? fmt(msg.step1Upgrade, PKG, installedVer, latestVer) : fmt(msg.step1, PKG);
  const doneLine = needsUpgrade ? fmt(msg.step1Upgraded, latestVer) : msg.step1Done;

  const s = interactive ? p.spinner() : null;
  if (s) s.start(startLine); else console.log(startLine);

  try {
    await runSilentAsync('npm', ['install', '-g', PKG], { timeout: 600000 });
    if (s) s.stop(doneLine); else console.log(doneLine);
  } catch (err) {
    const line = fmt(msg.step1Fail, PKG);
    if (s) s.stop(line); else console.error(line);

    const stderrStr = err && err.stderr ? err.stderr.toString() : '';
    const isEacces = /EACCES|permission denied/i.test(stderrStr);
    const isTimeout = !!(err && (err.code === 'ETIMEDOUT' || err.killed));
    if (isEacces) {
      const hint = fmt(msg.step1EaccesHint, PKG);
      if (interactive) p.log.warn(hint); else console.error(hint);
    } else if (isTimeout) {
      const hint = fmt(msg.step1TimeoutHint, PKG);
      if (interactive) p.log.warn(hint); else console.error(hint);
    }
    process.exit(1);
  }
}

async function skillsAlreadyInstalled() {
  try {
    const out = await runSilentAsync('npx', ['-y', 'skills', 'ls', '-g'], { timeout: 600000 });
    return /^ha-/m.test(out.toString());
  } catch (_) {
    return false;
  }
}

async function stepInstallSkills(interactive) {
  const s = interactive ? p.spinner() : null;
  if (s) s.start(msg.step2Spinner); else console.log(msg.step2Spinner);

  try {
    if (await skillsAlreadyInstalled()) {
      if (s) s.stop(msg.step2Skip); else console.log(msg.step2Skip);
      return;
    }
    await runSilentAsync('npx', ['-y', 'skills', 'add', SKILLS_REPO, '-y', '-g'], { timeout: 600000 });
    if (s) s.stop(msg.step2Done); else console.log(msg.step2Done);
  } catch (err) {
    const line = fmt(msg.step2Fail, SKILLS_REPO);
    if (s) s.stop(line); else console.error(line);
    const isTimeout = !!(err && (err.code === 'ETIMEDOUT' || err.killed));
    if (isTimeout) {
      const hint = `Timed out after 10 min. Retry outside the wizard: npx skills add ${SKILLS_REPO} -y -g`;
      if (interactive) p.log.warn(hint); else console.error(hint);
    }
    process.exit(1);
  }
}

async function main() {
  const interactive = !!process.stdin.isTTY && !!process.stdout.isTTY;

  if (interactive) {
    p.intro(msg.setup);
    await stepInstallGlobally(true);
    await stepInstallSkills(true);
    p.outro(msg.done);
  } else {
    console.log(msg.setup);
    await stepInstallGlobally(false);
    await stepInstallSkills(false);
    console.log(msg.nonTtyHint);
  }
}

main().catch((err) => {
  const line = 'Unexpected error: ' + (err && err.message ? err.message : err);
  try { p.cancel(line); } catch (_) { console.error(line); }
  process.exit(1);
});
