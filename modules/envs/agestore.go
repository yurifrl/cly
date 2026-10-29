package envs

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"filippo.io/age"
	"filippo.io/age/agessh"
)

// ageStore keeps the envs blob in an age-encrypted file. It is the default
// cache backend on Linux, where no OS secret-service daemon can be assumed.
//
// Encryption recipient, in order of preference:
//   - an SSH private key in sshDir (id_ed25519, then id_rsa) that parses
//     without a passphrase — the recipient is derived from the same
//     identity, so the file stays decryptable on this machine;
//   - otherwise a generated age identity persisted at keyPath.
//
// Decryption tries the same SSH private keys plus the generated identity.
//
// The stored payload is the same base64 blob the Keychain backend uses, so
// command behavior is identical across platforms.
type ageStore struct {
	sshDir   string // directory with SSH key pairs
	keyPath  string // generated age identity (X25519)
	blobPath string // encrypted cache file
}

// defaultAgeStore builds the store with standard XDG paths.
func defaultAgeStore() *ageStore {
	home, _ := os.UserHomeDir()
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		configDir = filepath.Join(home, ".config")
	}
	stateDir := os.Getenv("XDG_STATE_HOME")
	if stateDir == "" {
		stateDir = filepath.Join(home, ".local", "state")
	}
	return &ageStore{
		sshDir:   filepath.Join(home, ".ssh"),
		keyPath:  filepath.Join(configDir, "cly", "envs.age.key"),
		blobPath: filepath.Join(stateDir, "cly", "envs.age"),
	}
}

// Store encrypts the blob to the selected recipient and atomically replaces
// the cache file with 0600 permissions.
func (s *ageStore) Store(fields []Field) error {
	recipient, err := s.recipient()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.blobPath), 0o700); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.blobPath), "envs.age-*")
	if err != nil {
		return fmt.Errorf("create cache file: %w", err)
	}
	defer func() { _ = os.Remove(tmp.Name()) }()

	w, err := age.Encrypt(tmp, recipient)
	if err != nil {
		return fmt.Errorf("open encryptor: %w", err)
	}
	if _, err := io.WriteString(w, encodeBlob(fields)); err != nil {
		_ = w.Close()
		return fmt.Errorf("write blob: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("close encryptor: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close cache file: %w", err)
	}
	return os.Rename(tmp.Name(), s.blobPath)
}

// Read decrypts the cache file with the identities found on disk.
func (s *ageStore) Read() ([]Field, error) {
	f, err := os.Open(s.blobPath)
	if err != nil {
		return nil, fmt.Errorf("no envs cache at %s (run \"cly envs\" to create it)", s.blobPath)
	}
	defer func() { _ = f.Close() }()

	identities, err := s.identities()
	if err != nil {
		return nil, err
	}
	r, err := age.Decrypt(f, identities...)
	if err != nil {
		return nil, fmt.Errorf("decrypt %s: %w", s.blobPath, err)
	}
	encoded, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read blob: %w", err)
	}
	return decodeBlob(string(encoded))
}

// recipient picks the encryption target. Prefer an SSH private key that
// parses without a passphrase — its Recipient() is derived from the same
// identity, so the cache is always decryptable by construction. Falls back
// to the generated age identity.
func (s *ageStore) recipient() (age.Recipient, error) {
	for _, pemBytes := range sshPrivateKeys(s.sshDir) {
		id, err := agessh.ParseIdentity(pemBytes)
		if err != nil {
			continue
		}
		switch id := id.(type) {
		case *agessh.Ed25519Identity:
			return id.Recipient(), nil
		case *agessh.RSAIdentity:
			return id.Recipient(), nil
		}
	}
	id, err := s.ensureIdentity()
	if err != nil {
		return nil, err
	}
	return id.Recipient(), nil
}

// identities collects every usable decryption identity on disk: SSH private
// keys (encrypted ones are skipped — they cannot be opened without a
// passphrase prompt) and the generated age identity.
func (s *ageStore) identities() ([]age.Identity, error) {
	var ids []age.Identity
	for _, pemBytes := range sshPrivateKeys(s.sshDir) {
		if id, err := agessh.ParseIdentity(pemBytes); err == nil {
			ids = append(ids, id)
		}
	}
	if data, err := os.ReadFile(s.keyPath); err == nil {
		id, err := age.ParseX25519Identity(strings.TrimSpace(string(data)))
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", s.keyPath, err)
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil, errors.New("no usable age identity (looked for SSH keys in " +
			s.sshDir + " and for " + s.keyPath + ")")
	}
	return ids, nil
}

// ensureIdentity returns the stored age identity, generating and persisting
// one with 0600 permissions when absent.
func (s *ageStore) ensureIdentity() (*age.X25519Identity, error) {
	if data, err := os.ReadFile(s.keyPath); err == nil {
		return age.ParseX25519Identity(strings.TrimSpace(string(data)))
	}
	id, err := age.GenerateX25519Identity()
	if err != nil {
		return nil, fmt.Errorf("generate identity: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(s.keyPath), 0o700); err != nil {
		return nil, fmt.Errorf("create key dir: %w", err)
	}
	if err := os.WriteFile(s.keyPath, []byte(id.String()+"\n"), 0o600); err != nil {
		return nil, fmt.Errorf("write %s: %w", s.keyPath, err)
	}
	return id, nil
}

// sshPrivateKeys returns the PEM bytes of each reusable SSH private key in
// dir (ed25519 preferred, then RSA).
func sshPrivateKeys(dir string) [][]byte {
	var out [][]byte
	for _, name := range []string{"id_ed25519", "id_rsa"} {
		if data, err := os.ReadFile(filepath.Join(dir, name)); err == nil {
			out = append(out, data)
		}
	}
	return out
}
