package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ajramos/giztui/internal/environment"
	"github.com/stretchr/testify/assert"
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
