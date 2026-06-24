// Package keychain provides cross-platform secure storage for hass-cli secrets
// (currently the per-profile Home Assistant long-lived access tokens written by
// the `profile login` and `init` commands).
//
// The implementation is adapted from the Olares CLI keychain package (same
// Get/Set/Remove surface, same per-platform strategy split):
//
//   - macOS: a 32-byte AES-256 master key is kept in the system Keychain via
//     github.com/zalando/go-keyring; per-secret data is AES-GCM encrypted and
//     written to ~/Library/Application Support/<service>/<safeFileName>.enc.
//     If the system keychain is blocked (sandbox / CI) the master key falls
//     back to an on-disk master.key.file (mode 0600) under the same dir, so
//     the CLI keeps working at a Linux-equivalent security posture.
//   - Linux: pure file-based AES-GCM. The master key lives at
//     ~/.local/share/<service>/master.key (mode 0600); each secret lives at
//     <safeFileName>.enc next to it. Honors $HASS_CLI_DATA_DIR when set to
//     an absolute path. This matters because Home Assistant often runs on
//     headless hosts without a Secret Service / D-Bus session.
//   - Windows: DPAPI-protected blob (CryptProtectData/CryptUnprotectData)
//     persisted under HKCU\Software\HassCli\keychain\<service>, with
//     deterministic entropy bound to (service, account) to thwart swap/replay.
//
// Secrets are keyed by account name, which for hass-cli is always the profile
// name (e.g. "home", "work").
package keychain
