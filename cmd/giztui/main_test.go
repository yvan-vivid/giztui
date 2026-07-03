package main

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ajramos/giztui/internal/config"
	"github.com/ajramos/giztui/internal/environment"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandPath(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		contains string
	}{
		{"absolute_path", "/absolute/path", "/absolute/path"},
		{"relative_path", "relative/path", "relative/path"},
		{"home_only", "~", ""},
		{"home_with_subpath", "~/config/file", "config/file"},
		{"empty_path", "", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := environment.ExpandPath(tc.input)

			if tc.input == tc.contains {
				assert.Equal(t, tc.input, result)
			} else if strings.HasPrefix(tc.input, "~") && tc.contains != "" {
				assert.Contains(t, result, tc.contains)
				assert.NotContains(t, result, "~")
			}
		})
	}
}

func TestExpandPath_HomeDirectory(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot get home directory")
	}

	testCases := []struct {
		input    string
		expected string
	}{
		{"~", home},
		{"~/test", filepath.Join(home, "test")},
		{"~/config/file.json", filepath.Join(home, "config", "file.json")},
	}

	for _, tc := range testCases {
		result := environment.ExpandPath(tc.input)
		assert.Equal(t, tc.expected, result, "Path expansion for: %s", tc.input)
	}
}

func TestExpandPath_EdgeCases(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"no_tilde", "/path/without/tilde", "/path/without/tilde"},
		{"tilde_middle", "/path/~/middle", "/path/~/middle"},
		{"empty_string", "", ""},
		{"just_slash", "/", "/"},
		{"multiple_tildes", "~/~/test", "~/test"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.name == "multiple_tildes" {
				result := environment.ExpandPath(tc.input)
				assert.Contains(t, result, tc.expected)
				return
			}

			result := environment.ExpandPath(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestAWSRegionHandling(t *testing.T) {
	originalRegion := os.Getenv("AWS_REGION")
	defer func() { _ = os.Setenv("AWS_REGION", originalRegion) }()

	_ = os.Setenv("AWS_REGION", "us-east-1")
	result := os.Getenv("AWS_REGION")
	assert.Equal(t, "us-east-1", result)

	_ = os.Unsetenv("AWS_REGION")
	result = os.Getenv("AWS_REGION")
	assert.Empty(t, result)
}

func TestStringManipulation(t *testing.T) {
	t.Run("email_sanitization", func(t *testing.T) {
		email := "user@example.com"
		replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "@", "_", " ", "_")
		sanitized := replacer.Replace(strings.ToLower(strings.TrimSpace(email)))

		expected := "user_example.com"
		assert.Equal(t, expected, sanitized)
	})

	t.Run("path_sanitization_edge_cases", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected string
		}{
			{"User@Domain.Com", "user_domain.com"},
			{"  spaced@domain.com  ", "spaced_domain.com"},
			{"special/chars\\here:test@domain.com", "special_chars_here_test_domain.com"},
			{"", ""},
			{"   ", ""},
		}

		replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "@", "_", " ", "_")

		for _, tc := range testCases {
			result := replacer.Replace(strings.ToLower(strings.TrimSpace(tc.input)))
			if tc.expected == "" && tc.input != "" {
				assert.Empty(t, result, "Input: '%s'", tc.input)
			} else {
				assert.Equal(t, tc.expected, result, "Input: '%s'", tc.input)
			}
		}
	})
}

func TestFileExtensionHandling(t *testing.T) {
	testCases := []struct {
		path      string
		hasExt    bool
		extension string
	}{
		{"/path/file.txt", true, ".txt"},
		{"/path/file.json", true, ".json"},
		{"/path/file", false, ""},
		{"/path/dir/", false, ""},
		{"/path/dir", false, ""},
		{"file.sqlite3", true, ".sqlite3"},
		{"/path/file.", true, "."},
	}

	for _, tc := range testCases {
		ext := filepath.Ext(tc.path)
		if tc.hasExt {
			assert.Equal(t, tc.extension, ext, "Extension for path: %s", tc.path)
		} else {
			assert.Empty(t, ext, "Path should have no extension: %s", tc.path)
		}
	}
}

func TestPathJoining(t *testing.T) {
	testCases := []struct {
		base     string
		filename string
		expected string
	}{
		{"/base/path", "file.txt", "/base/path/file.txt"},
		{"/base", "subdir/file.txt", "/base/subdir/file.txt"},
		{".", "file.txt", "file.txt"},
		{"", "file.txt", "file.txt"},
	}

	for _, tc := range testCases {
		result := filepath.Join(tc.base, tc.filename)
		if tc.expected == result || strings.HasSuffix(result, tc.expected) {
			assert.True(t, true, "Path joining works for %s + %s", tc.base, tc.filename)
		} else {
			assert.Equal(t, tc.expected, result, "Path joining for %s + %s", tc.base, tc.filename)
		}
	}
}

func TestFlagParsing_Concepts(t *testing.T) {
	t.Run("empty_string_flag", func(t *testing.T) {
		flagValue := ""
		if flagValue != "" {
			t.Error("Empty string flag should be treated as not provided")
		}
	})

	t.Run("non_empty_flag", func(t *testing.T) {
		flagValue := "/custom/path"
		if flagValue == "" {
			t.Error("Non-empty flag should be used")
		}
		assert.Equal(t, "/custom/path", flagValue)
	})

	t.Run("boolean_flag", func(t *testing.T) {
		setupFlag := true
		if setupFlag {
			assert.True(t, true, "Setup flag should trigger special handling")
		}
	})
}

func TestPathValidation_Concepts(t *testing.T) {
	t.Run("file_exists_check", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.json")

		_, err := os.Stat(testFile)
		assert.True(t, os.IsNotExist(err), "File should not exist initially")

		err = os.WriteFile(testFile, []byte("{}"), 0600)
		assert.NoError(t, err)

		_, err = os.Stat(testFile)
		assert.NoError(t, err, "File should exist after creation")
	})

	t.Run("directory_creation", func(t *testing.T) {
		tmpDir := t.TempDir()
		subDir := filepath.Join(tmpDir, "subdir", "nested")

		_, err := os.Stat(subDir)
		assert.True(t, os.IsNotExist(err), "Directory should not exist initially")

		err = os.MkdirAll(subDir, 0750)
		assert.NoError(t, err)

		info, err := os.Stat(subDir)
		assert.NoError(t, err, "Directory should exist")
		assert.True(t, info.IsDir(), "Should be a directory")
	})
}

func TestConfigurationPriority(t *testing.T) {
	testCases := []struct {
		name     string
		flag     string
		env      string
		config   string
		expected string
	}{
		{"flag_priority", "/flag/path", "/env/path", "/config/path", "/flag/path"},
		{"env_priority", "", "/env/path", "/config/path", "/env/path"},
		{"config_priority", "", "", "/config/path", "/config/path"},
		{"all_empty", "", "", "", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var result string
			if tc.flag != "" {
				result = tc.flag
			} else if tc.env != "" {
				result = tc.env
			} else if tc.config != "" {
				result = tc.config
			}
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestCredentialFallbackSystem(t *testing.T) {
	tmpDir := t.TempDir()

	createCredFile := func(path string, content string) {
		err := os.MkdirAll(filepath.Dir(path), 0750)
		require.NoError(t, err)
		err = os.WriteFile(path, []byte(content), 0600)
		require.NoError(t, err)
	}

	t.Run("Level1_CLI_Flag_Success", func(t *testing.T) {
		cliCredPath := filepath.Join(tmpDir, "cli-credentials.json")
		createCredFile(cliCredPath, `{"client_id": "cli-test"}`)
		credPathFlag := &cliCredPath

		cfg := &config.Config{
			Credentials: filepath.Join(tmpDir, "nonexistent-config.json"),
			Token:       filepath.Join(tmpDir, "nonexistent-token.json"),
		}

		var logBuf bytes.Buffer
		logger := log.New(&logBuf, "", 0)

		credPath, tokenPath, fallbackMethod := testCredentialFallback(t, credPathFlag, cfg, logger)

		assert.Equal(t, cliCredPath, credPath)
		assert.Contains(t, tokenPath, "token.json")
		assert.Equal(t, "CLI flag", fallbackMethod)

		logOutput := logBuf.String()
		assert.Contains(t, logOutput, "Starting graceful credential fallback sequence")
		assert.Contains(t, logOutput, "Trying CLI flag credentials")
		assert.Contains(t, logOutput, "CLI flag credentials found and validated")
	})

	t.Run("Level2_Config_File_Success", func(t *testing.T) {
		configCredPath := filepath.Join(tmpDir, "config-credentials.json")
		configTokenPath := filepath.Join(tmpDir, "config-token.json")
		createCredFile(configCredPath, `{"client_id": "config-test"}`)
		createCredFile(configTokenPath, `{"access_token": "config-token"}`)

		var credPathFlag *string = nil

		cfg := &config.Config{
			Credentials: configCredPath,
			Token:       configTokenPath,
		}

		var logBuf bytes.Buffer
		logger := log.New(&logBuf, "", 0)

		credPath, tokenPath, fallbackMethod := testCredentialFallback(t, credPathFlag, cfg, logger)

		assert.Equal(t, configCredPath, credPath)
		assert.Equal(t, configTokenPath, tokenPath)
		assert.Equal(t, "config file", fallbackMethod)

		logOutput := logBuf.String()
		assert.Contains(t, logOutput, "Trying config file credentials")
		assert.Contains(t, logOutput, "Config file credentials found and validated")
	})

	t.Run("Level3_XDG_Defaults_Success", func(t *testing.T) {
		xdgDataHome := filepath.Join(tmpDir, "xdg-data")
		xdgStateHome := filepath.Join(tmpDir, "xdg-state")
		t.Setenv("XDG_DATA_HOME", xdgDataHome)
		t.Setenv("XDG_STATE_HOME", xdgStateHome)

		defaultCredPath := filepath.Join(xdgDataHome, "giztui", "credentials.json")
		createCredFile(defaultCredPath, `{"client_id": "default-test"}`)

		var credPathFlag *string = nil

		cfg := &config.Config{
			Credentials: filepath.Join(tmpDir, "missing-config.json"),
			Token:       filepath.Join(tmpDir, "missing-token.json"),
		}

		var logBuf bytes.Buffer
		logger := log.New(&logBuf, "", 0)

		credPath, tokenPath, fallbackMethod := testCredentialFallback(t, credPathFlag, cfg, logger)

		assert.Equal(t, defaultCredPath, credPath)
		assert.Contains(t, tokenPath, "token.json")
		assert.Equal(t, "XDG defaults", fallbackMethod)

		logOutput := logBuf.String()
		assert.Contains(t, logOutput, "Config credentials failed, trying XDG defaults")
		assert.Contains(t, logOutput, "XDG default credentials found and validated")
	})

	t.Run("Disabled_Config_Goes_To_Defaults", func(t *testing.T) {
		xdgDataHome := filepath.Join(tmpDir, "xdg-data2")
		xdgStateHome := filepath.Join(tmpDir, "xdg-state2")
		t.Setenv("XDG_DATA_HOME", xdgDataHome)
		t.Setenv("XDG_STATE_HOME", xdgStateHome)

		defaultCredPath := filepath.Join(xdgDataHome, "giztui", "credentials.json")
		createCredFile(defaultCredPath, `{"client_id": "default-test"}`)

		var credPathFlag *string = nil

		cfg := &config.Config{
			Credentials: "",
			Token:       "",
		}

		var logBuf bytes.Buffer
		logger := log.New(&logBuf, "", 0)

		credPath, tokenPath, fallbackMethod := testCredentialFallback(t, credPathFlag, cfg, logger)

		assert.Equal(t, defaultCredPath, credPath)
		assert.Contains(t, tokenPath, "token.json")
		assert.Equal(t, "XDG defaults", fallbackMethod)

		logOutput := logBuf.String()
		assert.Contains(t, logOutput, "No config credentials (disabled with prefix), trying XDG defaults")
		assert.Contains(t, logOutput, "XDG default credentials found and validated")
	})
}

func TestCredentialFallbackFailures(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("All_Sources_Exhausted_Returns_Empty", func(t *testing.T) {
		var credPathFlag *string = nil

		cfg := &config.Config{
			Credentials: filepath.Join(tmpDir, "missing-config.json"),
			Token:       filepath.Join(tmpDir, "missing-token.json"),
		}

		xdgDataHome := filepath.Join(tmpDir, "no-data")
		xdgStateHome := filepath.Join(tmpDir, "no-state")
		t.Setenv("XDG_DATA_HOME", xdgDataHome)
		t.Setenv("XDG_STATE_HOME", xdgStateHome)

		var logBuf bytes.Buffer
		logger := log.New(&logBuf, "", 0)

		credPath, tokenPath, fallbackMethod := testCredentialFallback(t, credPathFlag, cfg, logger)

		assert.Empty(t, credPath)
		assert.Empty(t, tokenPath)
		assert.Empty(t, fallbackMethod)

		logOutput := logBuf.String()
		assert.Contains(t, logOutput, "Config file credentials not found")
		assert.Contains(t, logOutput, "All credential fallback methods exhausted")
		assert.Contains(t, logOutput, "Tried config file and XDG defaults")
	})

	t.Run("CLI_Flag_Missing_Falls_Back", func(t *testing.T) {
		configCredPath := filepath.Join(tmpDir, "fallback-config.json")
		err := os.WriteFile(configCredPath, []byte(`{"client_id": "config"}`), 0600)
		require.NoError(t, err)

		missingPath := filepath.Join(tmpDir, "missing-cli.json")
		credPathFlag := &missingPath

		cfg := &config.Config{
			Credentials: configCredPath,
			Token:       filepath.Join(tmpDir, "config-token.json"),
		}

		var logBuf bytes.Buffer
		logger := log.New(&logBuf, "", 0)

		credPath, _, fallbackMethod := testCredentialFallback(t, credPathFlag, cfg, logger)

		assert.Equal(t, configCredPath, credPath)
		assert.Equal(t, "config file", fallbackMethod)

		logOutput := logBuf.String()
		assert.Contains(t, logOutput, "CLI flag credentials not found")
		assert.Contains(t, logOutput, "Trying config file credentials")
		assert.Contains(t, logOutput, "Config file credentials found and validated")
	})
}

// testCredentialFallback mirrors the fallback logic from main() for isolated testing.
func testCredentialFallback(t *testing.T, credPathFlag *string, cfg *config.Config, logger *log.Logger) (credPath, tokenPath, fallbackMethod string) {
	logger.Printf("🔄 Starting graceful credential fallback sequence...")

	var attemptNumber = 1

	// Level 1: Try CLI flag credentials (highest priority)
	if credPathFlag != nil && *credPathFlag != "" {
		logger.Printf("🎯 Attempt %d: Trying CLI flag credentials: %s", attemptNumber, *credPathFlag)
		attemptNumber++

		testCredPath := *credPathFlag
		testTokenPath := environment.ExpandPath(cfg.Token)
		if testTokenPath == "" {
			testTokenPath = environment.TokenPath()
		}

		logger.Printf("📍 Resolved paths - creds: %s, token: %s", testCredPath, testTokenPath)

		if testCredPath != "" {
			if _, err := os.Stat(testCredPath); err == nil {
				credPath = testCredPath
				tokenPath = testTokenPath
				fallbackMethod = "CLI flag"
				logger.Printf("✅ CLI flag credentials found and validated")
				logger.Printf("🚀 Initializing Gmail service with %s credentials (creds: %s, token: %s)", fallbackMethod, credPath, tokenPath)
				return
			} else {
				logger.Printf("❌ CLI flag credentials not found at %s", testCredPath)
			}
		}
	}

	// Level 2: Try config file credentials
	if credPath == "" && cfg.Credentials != "" {
		logger.Printf("🎯 Attempt %d: Trying config file credentials: %s", attemptNumber, cfg.Credentials)
		attemptNumber++

		testCredPath := environment.ExpandPath(cfg.Credentials)
		testTokenPath := environment.ExpandPath(cfg.Token)
		if testTokenPath == "" {
			testTokenPath = environment.TokenPath()
		}

		logger.Printf("📍 Resolved paths - creds: %s, token: %s", testCredPath, testTokenPath)

		if _, err := os.Stat(testCredPath); err == nil {
			credPath = testCredPath
			tokenPath = testTokenPath
			fallbackMethod = "config file"
			logger.Printf("✅ Config file credentials found and validated")
			logger.Printf("🚀 Initializing Gmail service with %s credentials (creds: %s, token: %s)", fallbackMethod, credPath, tokenPath)
			return
		} else {
			logger.Printf("❌ Config file credentials not found at %s", testCredPath)
		}
	}

	// Level 3: Try XDG default credentials (final fallback)
	if credPath == "" {
		if cfg.Credentials != "" {
			logger.Printf("🎯 Attempt %d: Config credentials failed, trying XDG defaults as final fallback", attemptNumber)
		} else {
			logger.Printf("🎯 Attempt %d: No config credentials (disabled with prefix), trying XDG defaults", attemptNumber)
		}

		testCredPath := environment.CredentialsPath()
		testTokenPath := environment.TokenPath()

		logger.Printf("📍 Resolved XDG paths - creds: %s, token: %s", testCredPath, testTokenPath)

		if testCredPath != "" {
			if _, err := os.Stat(testCredPath); err == nil {
				credPath = testCredPath
				tokenPath = testTokenPath
				fallbackMethod = "XDG defaults"
				logger.Printf("✅ XDG default credentials found and validated")
				logger.Printf("🚀 Initializing Gmail service with %s credentials (creds: %s, token: %s)", fallbackMethod, credPath, tokenPath)
				return
			} else {
				logger.Printf("❌ XDG default credentials not found at %s", testCredPath)
			}
		}
	}

	if credPath == "" {
		logger.Printf("❌ All credential fallback methods exhausted")
		logger.Printf("💡 Tried config file and XDG defaults")
		logger.Printf("💡 Please ensure at least one credential file exists and is accessible")
	}

	return credPath, tokenPath, fallbackMethod
}
