//go:build darwin

package keychain

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/zalando/go-keyring"
)

// keychainTimeout bounds system-keychain interactions to avoid hangs when the
// user dismisses (or never sees) the access prompt.
const keychainTimeout = 5 * time.Second

// fileMasterKeyName is the on-disk fallback master key used when the system
// keychain refuses access (sandbox / CI).
const fileMasterKeyName = "master.key.file"

// keyringGet / keyringSet are package-level seams so tests can simulate
// system-keychain behavior without touching the real macOS keychain.
var keyringGet = keyring.Get
var keyringSet = keyring.Set

// StorageDir returns the absolute directory where per-service encrypted blobs
// live on macOS. When HOME can't be resolved we land under os.TempDir() so the
// path is at least absolute.
func StorageDir(service string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		fallback := filepath.Join(os.TempDir(), "hass-cli", "keychain", service)
		fmt.Fprintf(os.Stderr,
			"warning: home directory unresolvable (%v); using fallback keychain dir %s\n",
			err, fallback)
		return fallback
	}
	return filepath.Join(home, "Library", "Application Support", service)
}

// getMasterKey fetches the AES master key from the system keychain. The
// goroutine + timeout dance protects us from a hung permission prompt.
//
// allowCreate gates the write path: Set() may create a fresh key, Get() may
// not (a missing key on read should surface as errNotInitialized).
func getMasterKey(service string, allowCreate bool) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), keychainTimeout)
	defer cancel()

	type result struct {
		key []byte
		err error
	}
	resCh := make(chan result, 1)
	go func() {
		defer func() { _ = recover() }()

		encodedKey, err := keyringGet(service, "master.key")
		if err == nil {
			key, decodeErr := base64.StdEncoding.DecodeString(encodedKey)
			if decodeErr == nil && len(key) == masterKeyBytes {
				resCh <- result{key: key, err: nil}
				return
			}
			resCh <- result{key: nil, err: errors.New("keychain is corrupted")}
			return
		} else if !errors.Is(err, keyring.ErrNotFound) {
			resCh <- result{key: nil, err: errors.New("keychain access blocked")}
			return
		}

		if !allowCreate {
			resCh <- result{key: nil, err: errNotInitialized}
			return
		}

		key := make([]byte, masterKeyBytes)
		if _, randErr := rand.Read(key); randErr != nil {
			resCh <- result{key: nil, err: randErr}
			return
		}
		encodedKeyStr := base64.StdEncoding.EncodeToString(key)
		if setErr := keyringSet(service, "master.key", encodedKeyStr); setErr != nil {
			resCh <- result{key: nil, err: setErr}
			return
		}
		resCh <- result{key: key, err: nil}
	}()

	select {
	case res := <-resCh:
		return res.key, res.err
	case <-ctx.Done():
		return nil, errors.New("keychain access blocked")
	}
}

// getFileMasterKey is the on-disk fallback master key used when the system
// keychain is denied (sandbox / CI). Once a process has created it, future
// reads/writes prefer it over the system keychain so we never re-prompt.
func getFileMasterKey(service string, allowCreate bool) ([]byte, error) {
	dir := StorageDir(service)
	keyPath := filepath.Join(dir, fileMasterKeyName)

	key, err := os.ReadFile(keyPath)
	if err == nil && len(key) == masterKeyBytes {
		return key, nil
	}
	if err == nil && len(key) != masterKeyBytes {
		return nil, errors.New("keychain is corrupted")
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if !allowCreate {
		return nil, errNotInitialized
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	key = make([]byte, masterKeyBytes)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}

	file, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			for i := 0; i < 3; i++ {
				existingKey, readErr := os.ReadFile(keyPath)
				if readErr == nil && len(existingKey) == masterKeyBytes {
					return existingKey, nil
				}
				if readErr != nil {
					return nil, readErr
				}
				if i < 2 {
					time.Sleep(5 * time.Millisecond)
				}
			}
			return nil, errors.New("keychain is corrupted")
		}
		return nil, err
	}

	writeFailed := true
	defer func() {
		if writeFailed {
			_ = os.Remove(keyPath)
		}
	}()
	if _, err := file.Write(key); err != nil {
		_ = file.Close()
		return nil, err
	}
	if err := file.Close(); err != nil {
		return nil, err
	}
	writeFailed = false

	canonicalKey, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}
	if len(canonicalKey) != masterKeyBytes {
		return nil, errors.New("keychain is corrupted")
	}
	return canonicalKey, nil
}

// platformGet is the macOS implementation of Get. The dual-master-key fallback
// (file first, then system keychain) is intentional.
func platformGet(service, account string) (string, error) {
	path := filepath.Join(StorageDir(service), safeFileName(account))
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if key, ferr := getFileMasterKey(service, false); ferr == nil {
		if plaintext, derr := decryptData(data, key); derr == nil {
			return plaintext, nil
		}
	}
	key, err := getMasterKey(service, false)
	if err != nil {
		return "", err
	}
	plaintext, err := decryptData(data, key)
	if err != nil {
		return "", err
	}
	return plaintext, nil
}

// platformSet writes the encrypted blob via temp-file + rename. The
// key-acquisition chain is: prefer existing file master key → try system
// keychain (creating if needed) → fall back to creating a new file master key.
func platformSet(service, account, data string) error {
	key, err := getFileMasterKey(service, false)
	if err != nil {
		key, err = getMasterKey(service, true)
		if err != nil {
			key, err = getFileMasterKey(service, true)
			if err != nil {
				return err
			}
		}
	}
	dir := StorageDir(service)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	encrypted, err := encryptData(data, key)
	if err != nil {
		return err
	}

	targetPath := filepath.Join(dir, safeFileName(account))
	tmpPath := filepath.Join(dir, safeFileName(account)+"."+uuid.New().String()+".tmp")
	defer os.Remove(tmpPath)

	if err := os.WriteFile(tmpPath, encrypted, 0o600); err != nil {
		return err
	}
	return os.Rename(tmpPath, targetPath)
}

// platformRemove deletes the encrypted blob; the master key is left in place
// because it may still encrypt other accounts.
func platformRemove(service, account string) error {
	err := os.Remove(filepath.Join(StorageDir(service), safeFileName(account)))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// platformPurge wipes everything owned by service on darwin: the master key in
// the system keychain (best-effort), the master.key.file on disk, and the
// entire StorageDir.
func platformPurge(service string) error {
	var firstErr error
	if err := keyring.Delete(service, "master.key"); err != nil && !errors.Is(err, keyring.ErrNotFound) {
		firstErr = err
	}
	if err := os.RemoveAll(StorageDir(service)); err != nil && firstErr == nil {
		firstErr = err
	}
	return firstErr
}

// platformBackend reports which master-key path is currently authoritative on
// darwin. Presence of master.key.file is the precise signal that we previously
// fell back off the system keychain.
func platformBackend(service string) string {
	keyPath := filepath.Join(StorageDir(service), fileMasterKeyName)
	if info, err := os.Stat(keyPath); err == nil && info.Size() == masterKeyBytes {
		return "file-fallback"
	}
	return "system-keychain"
}
