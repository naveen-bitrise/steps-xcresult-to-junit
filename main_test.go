package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bitrise-io/go-steputils/stepconf"
)

func TestStepConfigValidation(t *testing.T) {
	t.Run("valid config should not error", func(t *testing.T) {
		// Create a temporary XCResult file
		tempDir, err := os.MkdirTemp("", "xcresult-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		xcresultPath := filepath.Join(tempDir, "test.xcresult")
		if err := os.Mkdir(xcresultPath, 0755); err != nil {
			t.Fatalf("Failed to create test.xcresult dir: %v", err)
		}

		// Set environment variables for testing
		os.Setenv("xcresult_path", xcresultPath)
		os.Setenv("output_dir", tempDir)
		os.Setenv("junit_filename", "junit.xml")
		os.Setenv("verbose", "yes")

		// Parse config (should not fail)
		var config Config
		err = stepconf.Parse(&config)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Validate config values
		if config.XCResultPath != xcresultPath {
			t.Errorf("Expected XCResultPath to be %s, got %s", xcresultPath, config.XCResultPath)
		}
		if config.OutputDir != tempDir {
			t.Errorf("Expected OutputDir to be %s, got %s", tempDir, config.OutputDir)
		}
		if config.JUnitFilename != "junit.xml" {
			t.Errorf("Expected JUnitFilename to be junit.xml, got %s", config.JUnitFilename)
		}
		if config.Verbose != "yes" {
			t.Errorf("Expected Verbose to be yes, got %s", config.Verbose)
		}
	})

	t.Run("missing required config should error", func(t *testing.T) {
		// Clear environment variables
		os.Unsetenv("xcresult_path")
		os.Unsetenv("output_dir")
		os.Unsetenv("junit_filename")
		os.Unsetenv("verbose")

		// Parse config (should fail)
		var config Config
		err := stepconf.Parse(&config)
		if err == nil {
			t.Errorf("Expected error for missing required inputs, got nil")
		}
	})
}

func TestExportOutput(t *testing.T) {
	// Skip this test in CI environments where envman might not be available
	if os.Getenv("CI") != "" {
		t.Skip("Skipping test in CI environment")
	}

	t.Run("export output should not error", func(t *testing.T) {
		err := exportOutput("TEST_KEY", "test_value")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})
}
