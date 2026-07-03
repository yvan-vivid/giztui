package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ajramos/giztui/internal/config"
	"github.com/stretchr/testify/assert"
)

// writeDefaultCredentialFiles creates credentials and token files in XDG directories
// under the given temp root and returns the data/state dirs.
func writeDefaultCredentialFiles(t *testing.T, withToken bool) (string, string) {
	t.Helper()
	dataDir := filepath.Join(t.TempDir(), "share", "giztui")
	stateDir := filepath.Join(t.TempDir(), "state", "giztui")
	if err := os.MkdirAll(dataDir, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.MkdirAll(stateDir, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "credentials.json"), []byte(`{"installed":{}}`), 0o600); err != nil {
		t.Fatalf("write credentials: %v", err)
	}
	if withToken {
		if err := os.WriteFile(filepath.Join(stateDir, "token.json"), []byte(`{"access_token":"x"}`), 0o600); err != nil {
			t.Fatalf("write token: %v", err)
		}
	}
	return dataDir, stateDir
}

// TestAccountService_LegacyFallback_DefaultFilesExist verifies the regression from issue #42:
// with no `accounts` and no `credentials`/`token` in config, but the default credential files
// present, a default account is created so the database can initialize.
func TestAccountService_LegacyFallback_DefaultFilesExist(t *testing.T) {
	tmpRoot := t.TempDir()

	dataDir := filepath.Join(tmpRoot, "share", "giztui")
	stateDir := filepath.Join(tmpRoot, "state", "giztui")
	if err := os.MkdirAll(dataDir, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.MkdirAll(stateDir, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "credentials.json"), []byte(`{"installed":{}}`), 0o600); err != nil {
		t.Fatalf("write credentials: %v", err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, "token.json"), []byte(`{"access_token":"x"}`), 0o600); err != nil {
		t.Fatalf("write token: %v", err)
	}

	t.Setenv("XDG_DATA_HOME", filepath.Join(tmpRoot, "share"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(tmpRoot, "state"))

	cfg := &config.Config{} // no Accounts, no Credentials, no Token
	svc := NewAccountService(cfg, nil)

	acc, err := svc.GetActiveAccount(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, acc)
	assert.Equal(t, "default", acc.ID)
	assert.True(t, acc.IsActive)
	assert.Contains(t, acc.CredPath, "credentials.json")
	assert.Contains(t, acc.TokenPath, "token.json")
}

// TestAccountService_LegacyFallback_NoFiles verifies that without any credential files and no
// config, no account is created (and GetActiveAccount errors), rather than a phantom account.
func TestAccountService_LegacyFallback_NoFiles(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(home, ".local", "state"))
	// Intentionally do NOT create any credential files.

	cfg := &config.Config{}
	svc := NewAccountService(cfg, nil)

	acc, err := svc.GetActiveAccount(context.Background())
	assert.Error(t, err)
	assert.Nil(t, acc)
}

// TestAccountService_LegacyFallback_ExplicitConfigStillWins verifies that an explicit config
// Credentials path is honored (and the existing behavior is preserved).
func TestAccountService_LegacyFallback_ExplicitConfigWins(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Put the credential file at a non-default location and point config at it.
	customDir := filepath.Join(home, "custom")
	if err := os.MkdirAll(customDir, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	credPath := filepath.Join(customDir, "creds.json")
	if err := os.WriteFile(credPath, []byte(`{"installed":{}}`), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	cfg := &config.Config{Credentials: credPath}
	svc := NewAccountService(cfg, nil)

	acc, err := svc.GetActiveAccount(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, acc)
	assert.Equal(t, credPath, acc.CredPath)
}

func TestResolveLegacyCredentialPath(t *testing.T) {
	tmpRoot := t.TempDir()
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmpRoot, "share"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(tmpRoot, "state"))

	// Empty configured -> default XDG paths.
	gotToken := resolveLegacyCredentialPath("", "token.json")
	assert.Equal(t, filepath.Join(tmpRoot, "state", "giztui", "token.json"), gotToken)

	gotCred := resolveLegacyCredentialPath("", "credentials.json")
	assert.Equal(t, filepath.Join(tmpRoot, "share", "giztui", "credentials.json"), gotCred)

	// Configured absolute path passes through unchanged.
	abs := filepath.Join(tmpRoot, "x", "creds.json")
	assert.Equal(t, abs, resolveLegacyCredentialPath(abs, "credentials.json"))

	// Configured ~ path is expanded.
	home, _ := os.UserHomeDir()
	assert.Equal(t, filepath.Join(home, "creds.json"), resolveLegacyCredentialPath("~/creds.json", "credentials.json"))
}

func TestFileExists(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "f.json")
	assert.False(t, fileExists(f))
	assert.False(t, fileExists(""))
	if err := os.WriteFile(f, []byte("x"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	assert.True(t, fileExists(f))
	assert.False(t, fileExists(dir)) // directory is not a regular file
}
