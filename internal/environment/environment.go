package environment

import (
	"os"
	"path/filepath"
	"strings"
)

const appName = "giztui"

// ConfigDir returns the XDG configuration directory for giztui.
// Uses $XDG_CONFIG_HOME if set, otherwise falls back to ~/.config.
func ConfigDir() string {
	return xdgDir("XDG_CONFIG_HOME", ".config")
}

// DataDir returns the XDG data directory for giztui.
// Uses $XDG_DATA_HOME if set, otherwise falls back to ~/.local/share.
func DataDir() string {
	return xdgDir("XDG_DATA_HOME", filepath.Join(".local", "share"))
}

// StateDir returns the XDG state directory for giztui.
// Uses $XDG_STATE_HOME if set, otherwise falls back to ~/.local/state.
func StateDir() string {
	return xdgDir("XDG_STATE_HOME", filepath.Join(".local", "state"))
}

// CacheDir returns the XDG cache directory for giztui.
// Uses $XDG_CACHE_HOME if set, otherwise falls back to ~/.cache.
func CacheDir() string {
	return xdgDir("XDG_CACHE_HOME", ".cache")
}

// ConfigPath returns the full path to the configuration file.
func ConfigPath() string {
	return filepath.Join(ConfigDir(), "config.json")
}

// CredentialsPath returns the full path to the default OAuth2 credentials file.
func CredentialsPath() string {
	return filepath.Join(DataDir(), "credentials.json")
}

// TokenPath returns the full path to the default OAuth2 token file.
func TokenPath() string {
	return filepath.Join(StateDir(), "token.json")
}

// CredentialsDir returns the directory containing per-account credential files.
func CredentialsDir() string {
	return filepath.Join(DataDir(), "credentials")
}

// TokensDir returns the directory containing per-account token files.
func TokensDir() string {
	return filepath.Join(StateDir(), "tokens")
}

// AccountCredentialsPath returns the path to the credential file for the given account.
// credName is the stem name of the credentials file (without .json extension).
func AccountCredentialsPath(credName string) string {
	return filepath.Join(CredentialsDir(), credName+".json")
}

// AccountTokenPath returns the path to the token file for the given account ID.
func AccountTokenPath(accountID string) string {
	return filepath.Join(TokensDir(), accountID+".json")
}

// LogPath returns the full path to the log file.
func LogPath() string {
	return filepath.Join(CacheDir(), "giztui.log")
}

// SavedDir returns the path to the saved emails directory.
func SavedDir() string {
	return filepath.Join(StateDir(), "saved")
}

// ThemesDir returns the path to the user themes directory.
func ThemesDir() string {
	return filepath.Join(ConfigDir(), "themes")
}

// TemplatesDir returns the path to the AI prompt templates directory.
func TemplatesDir() string {
	return filepath.Join(ConfigDir(), "templates")
}

// PiperDir returns the path to the piper TTS models directory.
func PiperDir() string {
	return filepath.Join(DataDir(), "piper")
}

// xdgDir resolves an XDG base directory for the giztui application.
// It checks the given environment variable first; if unset or empty,
// it falls back to defaultPath relative to the user's home directory.
func xdgDir(envVar, defaultPath string) string {
	if v := os.Getenv(envVar); v != "" {
		return filepath.Join(v, appName)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	return filepath.Join(home, defaultPath, appName)
}

// ExpandPath expands a leading ~ to the user's home directory.
// If the path does not start with ~, it is returned unchanged.
func ExpandPath(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}

	if path == "~" {
		return home
	}

	return filepath.Join(home, path[2:])
}
