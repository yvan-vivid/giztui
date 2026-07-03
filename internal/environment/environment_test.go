package environment

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigDir(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "")
		home, _ := os.UserHomeDir()
		expected := filepath.Join(home, ".config", appName)
		if got := ConfigDir(); got != expected {
			t.Errorf("ConfigDir() = %q, want %q", got, expected)
		}
	})

	t.Run("xdg override", func(t *testing.T) {
		tmp := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tmp)
		expected := filepath.Join(tmp, appName)
		if got := ConfigDir(); got != expected {
			t.Errorf("ConfigDir() = %q, want %q", got, expected)
		}
	})
}

func TestDataDir(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		t.Setenv("XDG_DATA_HOME", "")
		home, _ := os.UserHomeDir()
		expected := filepath.Join(home, ".local", "share", appName)
		if got := DataDir(); got != expected {
			t.Errorf("DataDir() = %q, want %q", got, expected)
		}
	})

	t.Run("xdg override", func(t *testing.T) {
		tmp := t.TempDir()
		t.Setenv("XDG_DATA_HOME", tmp)
		expected := filepath.Join(tmp, appName)
		if got := DataDir(); got != expected {
			t.Errorf("DataDir() = %q, want %q", got, expected)
		}
	})
}

func TestStateDir(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		t.Setenv("XDG_STATE_HOME", "")
		home, _ := os.UserHomeDir()
		expected := filepath.Join(home, ".local", "state", appName)
		if got := StateDir(); got != expected {
			t.Errorf("StateDir() = %q, want %q", got, expected)
		}
	})

	t.Run("xdg override", func(t *testing.T) {
		tmp := t.TempDir()
		t.Setenv("XDG_STATE_HOME", tmp)
		expected := filepath.Join(tmp, appName)
		if got := StateDir(); got != expected {
			t.Errorf("StateDir() = %q, want %q", got, expected)
		}
	})
}

func TestCacheDir(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		t.Setenv("XDG_CACHE_HOME", "")
		home, _ := os.UserHomeDir()
		expected := filepath.Join(home, ".cache", appName)
		if got := CacheDir(); got != expected {
			t.Errorf("CacheDir() = %q, want %q", got, expected)
		}
	})

	t.Run("xdg override", func(t *testing.T) {
		tmp := t.TempDir()
		t.Setenv("XDG_CACHE_HOME", tmp)
		expected := filepath.Join(tmp, appName)
		if got := CacheDir(); got != expected {
			t.Errorf("CacheDir() = %q, want %q", got, expected)
		}
	})
}

func TestConfigPath(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".config", appName, "config.json")
	if got := ConfigPath(); got != expected {
		t.Errorf("ConfigPath() = %q, want %q", got, expected)
	}
}

func TestCredentialsPath(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "")
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".local", "share", appName, "credentials.json")
	if got := CredentialsPath(); got != expected {
		t.Errorf("CredentialsPath() = %q, want %q", got, expected)
	}
}

func TestTokenPath(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "")
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".local", "state", appName, "token.json")
	if got := TokenPath(); got != expected {
		t.Errorf("TokenPath() = %q, want %q", got, expected)
	}
}

func TestLogPath(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", "")
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".cache", appName, "giztui.log")
	if got := LogPath(); got != expected {
		t.Errorf("LogPath() = %q, want %q", got, expected)
	}
}

func TestSavedDir(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "")
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".local", "state", appName, "saved")
	if got := SavedDir(); got != expected {
		t.Errorf("SavedDir() = %q, want %q", got, expected)
	}
}

func TestThemesDir(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".config", appName, "themes")
	if got := ThemesDir(); got != expected {
		t.Errorf("ThemesDir() = %q, want %q", got, expected)
	}
}

func TestTemplatesDir(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".config", appName, "templates")
	if got := TemplatesDir(); got != expected {
		t.Errorf("TemplatesDir() = %q, want %q", got, expected)
	}
}

func TestPiperDir(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "")
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".local", "share", appName, "piper")
	if got := PiperDir(); got != expected {
		t.Errorf("PiperDir() = %q, want %q", got, expected)
	}
}

func TestExpandPath(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"no tilde", "/some/path", "/some/path"},
		{"tilde only", "~", home},
		{"tilde with suffix", "~/foo/bar", filepath.Join(home, "foo", "bar")},
		{"relative no tilde", "relative/path", "relative/path"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExpandPath(tt.input); got != tt.expected {
				t.Errorf("ExpandPath(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestXDGPrecedence(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmp, "data"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(tmp, "state"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(tmp, "cache"))

	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"config", ConfigDir(), filepath.Join(tmp, "config", appName)},
		{"data", DataDir(), filepath.Join(tmp, "data", appName)},
		{"state", StateDir(), filepath.Join(tmp, "state", appName)},
		{"cache", CacheDir(), filepath.Join(tmp, "cache", appName)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("got %q, want %q", tt.got, tt.expected)
			}
		})
	}
}
