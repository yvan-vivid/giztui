package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ajramos/giztui/internal/config"
	"github.com/stretchr/testify/assert"
)

// writeAccountCredentialFiles creates credentials and token files for an account
// in the new directory-based structure (credentials/<credName>.json, tokens/<accountID>.json).
func writeAccountCredentialFiles(t *testing.T, tmpRoot, credName, accountID string, withToken bool) {
	t.Helper()
	credDir := filepath.Join(tmpRoot, "share", "giztui", "credentials")
	tokenDir := filepath.Join(tmpRoot, "state", "giztui", "tokens")
	if err := os.MkdirAll(credDir, 0o750); err != nil {
		t.Fatalf("mkdir credDir: %v", err)
	}
	if err := os.MkdirAll(tokenDir, 0o750); err != nil {
		t.Fatalf("mkdir tokenDir: %v", err)
	}
	credFile := filepath.Join(credDir, credName+".json")
	if err := os.WriteFile(credFile, []byte(`{"installed":{}}`), 0o600); err != nil {
		t.Fatalf("write credentials: %v", err)
	}
	if withToken {
		tokenFile := filepath.Join(tokenDir, accountID+".json")
		if err := os.WriteFile(tokenFile, []byte(`{"access_token":"x"}`), 0o600); err != nil {
			t.Fatalf("write token: %v", err)
		}
	}
}

// TestAccountService_LoadAccounts_ConfiguredAccount verifies that configured accounts
// are loaded and paths are resolved from the credentials name.
func TestAccountService_LoadAccounts_ConfiguredAccount(t *testing.T) {
	tmpRoot := t.TempDir()

	// Create credential files for "personal" account using "google-oauth" credentials
	writeAccountCredentialFiles(t, tmpRoot, "google-oauth", "personal", true)

	t.Setenv("XDG_DATA_HOME", filepath.Join(tmpRoot, "share"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(tmpRoot, "state"))

	cfg := &config.Config{
		Accounts: []config.AccountConfig{
			{ID: "personal", Credentials: "google-oauth", DisplayName: "Personal", Active: true},
		},
	}
	svc := NewAccountService(cfg, nil)

	acc, err := svc.GetActiveAccount(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, acc)
	assert.Equal(t, "personal", acc.ID)
	assert.Equal(t, "google-oauth", acc.CredentialsName)
	assert.Equal(t, "Personal", acc.DisplayName)
	assert.True(t, acc.IsActive)
}

// TestAccountService_LoadAccounts_NoAccounts verifies that with no accounts configured,
// GetActiveAccount returns an error (no legacy fallback).
func TestAccountService_LoadAccounts_NoAccounts(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(home, ".local", "state"))

	cfg := &config.Config{}
	svc := NewAccountService(cfg, nil)

	acc, err := svc.GetActiveAccount(context.Background())
	assert.Error(t, err)
	assert.Nil(t, acc)
}

// TestAccountService_LoadAccounts_MultipleAccounts verifies multiple accounts
// sharing the same credentials and that only the first active one is selected.
func TestAccountService_LoadAccounts_MultipleAccounts(t *testing.T) {
	tmpRoot := t.TempDir()

	// Both accounts share "google-oauth" credentials, separate tokens
	writeAccountCredentialFiles(t, tmpRoot, "google-oauth", "personal", true)
	writeAccountCredentialFiles(t, tmpRoot, "google-oauth", "work", true)

	t.Setenv("XDG_DATA_HOME", filepath.Join(tmpRoot, "share"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(tmpRoot, "state"))

	cfg := &config.Config{
		Accounts: []config.AccountConfig{
			{ID: "personal", Credentials: "google-oauth", DisplayName: "Personal", Active: true},
			{ID: "work", Credentials: "google-oauth", DisplayName: "Work", Active: true},
		},
	}
	svc := NewAccountService(cfg, nil)

	acc, err := svc.GetActiveAccount(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, acc)
	// First active account wins
	assert.Equal(t, "personal", acc.ID)

	accounts, err := svc.ListAccounts(context.Background())
	assert.NoError(t, err)
	assert.Len(t, accounts, 2)
}

// TestAccountService_AddAccount_ValidateCredentialFile verifies AddAccount checks
// that the credential file exists at the resolved path.
func TestAccountService_AddAccount_ValidateCredentialFile(t *testing.T) {
	tmpRoot := t.TempDir()

	writeAccountCredentialFiles(t, tmpRoot, "newaccount-creds", "newaccount", false)

	t.Setenv("XDG_DATA_HOME", filepath.Join(tmpRoot, "share"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(tmpRoot, "state"))

	cfg := &config.Config{}
	svc := NewAccountService(cfg, nil)

	// Should succeed - credential file exists
	err := svc.AddAccount(context.Background(), &Account{
		ID:              "newaccount",
		CredentialsName: "newaccount-creds",
		DisplayName:     "New Account",
	})
	assert.NoError(t, err)

	// Should fail - credential file does not exist for "nonexistent"
	err = svc.AddAccount(context.Background(), &Account{
		ID:              "nonexistent",
		CredentialsName: "nonexistent-creds",
		DisplayName:     "Nonexistent",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "credentials file not found")
}

// TestAccountService_AddAccount_MissingCredentialsField verifies AddAccount rejects
// accounts with empty credentials field.
func TestAccountService_AddAccount_MissingCredentialsField(t *testing.T) {
	cfg := &config.Config{}
	svc := NewAccountService(cfg, nil)

	err := svc.AddAccount(context.Background(), &Account{
		ID:          "no-creds",
		DisplayName: "No Creds",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "credentials field is required")
}

// TestAccountService_LoadAccounts_MissingCredentialsField verifies that loadAccountsFromConfig
// returns an error when an account has an empty credentials field.
func TestAccountService_LoadAccounts_MissingCredentialsField(t *testing.T) {
	cfg := &config.Config{
		Accounts: []config.AccountConfig{
			{ID: "personal", DisplayName: "Personal", Active: true},
		},
	}
	svc := NewAccountService(cfg, nil)

	// Should fail to load - credentials field is empty
	acc, err := svc.GetActiveAccount(context.Background())
	assert.Error(t, err)
	assert.Nil(t, acc)
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
