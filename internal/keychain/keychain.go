package keychain

import (
	"errors"
	"fmt"
	"os"
)

var (
	// ErrNotFound is returned when the requested credential is not found.
	// platformGet implementations return ("", nil) for the common
	// "not present" case; ErrNotFound is reserved for callers that want to
	// turn that empty-string signal into a typed error.
	ErrNotFound = errors.New("keychain: item not found")

	// errNotInitialized is the internal sentinel used when the master key is
	// missing or invalid. It triggers a more specific operator hint in
	// wrapError so users know to re-login rather than blaming permissions.
	errNotInitialized = errors.New("keychain not initialized")
)

// HassCliService is the unified keychain service name for all hass-cli
// secrets. Per-secret records are distinguished by their account name, which
// for hass-cli is always the profile name.
const HassCliService = "hass-cli"

// debugEnv toggles the long, multi-line operator hint that wrapError can
// attach. Gating verbose hints behind this env var keeps everyday failures
// grep-friendly.
const debugEnv = "HASS_CLI_DEBUG"

// debugLookup is a package-level seam so tests can flip the hint on/off
// without writing to process env.
var debugLookup = func() bool { return os.Getenv(debugEnv) != "" }

// wrapError turns underlying backend errors into user-facing messages.
// Returning ErrNotFound (or nil) is preserved verbatim so callers can use
// errors.Is on it.
func wrapError(op, service, account string, err error) error {
	if err == nil || errors.Is(err, ErrNotFound) {
		return err
	}

	base := fmt.Errorf("keychain %s failed for %s/%s: %w", op, service, account, err)
	if !debugLookup() {
		return base
	}

	hint := "Check whether the OS keychain / credential manager is locked or accessible. " +
		"If you are running inside a sandbox or CI environment, ensure the process has " +
		"permission to use the keychain — running outside the sandbox usually fixes it."
	if errors.Is(err, errNotInitialized) {
		hint = "The keychain master key may have been deleted or corrupted. " +
			"Re-run `hass-cli profile login` to re-store credentials. " +
			"In sandboxed / CI environments, ensure the process can access the OS keychain."
	}
	return fmt.Errorf("%w (%s)", base, hint)
}

// KeychainAccess abstracts Get/Set/Remove for dependency injection. Production
// code wires the package-level functions through Default(); tests can pass a
// fake to assert call patterns without touching the real OS keychain.
type KeychainAccess interface {
	Get(service, account string) (string, error)
	Set(service, account, value string) error
	Remove(service, account string) error
}

// Get retrieves a value from the keychain. Returns ("", nil) when the entry
// does not exist; callers that prefer a typed "not found" should check
// len(value)==0.
func Get(service, account string) (string, error) {
	val, err := platformGet(service, account)
	return val, wrapError("Get", service, account, err)
}

// Set stores a value in the keychain, overwriting any existing entry.
func Set(service, account, data string) error {
	return wrapError("Set", service, account, platformSet(service, account, data))
}

// Remove deletes an entry from the keychain. Removing a non-existent entry is
// a no-op and returns nil.
func Remove(service, account string) error {
	return wrapError("Remove", service, account, platformRemove(service, account))
}

// Backend returns a short, machine-friendly identifier of the platform
// backend currently in effect for service:
//
//   - "system-keychain" — darwin, master key lives in the OS keychain
//   - "file-fallback"   — darwin, sandbox/CI path: master key on disk
//   - "file"            — linux, master key on disk under XDG dir
//   - "registry+dpapi"  — windows, registry value protected by DPAPI
func Backend(service string) string { return platformBackend(service) }

// PurgeService wipes ALL keychain state owned by the given service: the
// per-account encrypted blobs AND the master key. Designed to be called when
// the last hass-cli profile is removed so we don't leave orphan secrets.
func PurgeService(service string) error {
	return wrapError("Purge", service, "*", platformPurge(service))
}
