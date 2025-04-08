package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/bitrise-io/go-steputils/stepconf"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
)

// Config holds the step configuration
type Config struct {
	XCResultPath  string `env:"xcresult_path,required"`
	OutputDir     string `env:"output_dir,required"`
	JUnitFilename string `env:"junit_filename,required"`
	Verbose       string `env:"verbose"`
}

func main() {
	var config Config
	if err := stepconf.Parse(&config); err != nil {
		failf("Failed to parse config: %s", err)
	}
	stepconf.Print(config)
	log.SetEnableDebugLog(config.Verbose == "yes")

	// Check if XCResult path exists
	if exists, err := pathutil.IsPathExists(config.XCResultPath); err != nil {
		failf("Failed to check if XCResult path exists: %s", err)
	} else if !exists {
		failf("XCResult path does not exist: %s", config.XCResultPath)
	}

	// Create output directory if it doesn't exist
	if exists, err := pathutil.IsPathExists(config.OutputDir); err != nil {
		failf("Failed to check if output directory exists: %s", err)
	} else if !exists {
		if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
			failf("Failed to create output directory: %s", err)
		}
	}

	// Convert XCResult to JSON
	log.Infof("Converting XCResult to JSON...")
	jsonData, err := convertXCResultToJSON(config.XCResultPath)
	if err != nil {
		failf("Failed to convert XCResult to JSON: %s", err)
	}

	// Convert JSON to JUnit XML
	log.Infof("Converting JSON to JUnit XML...")
	//log.Infof("JSON data: %s", string(jsonData))
	junitXML, err := ConvertXCResultJSONToJUnitXML(jsonData)
	if err != nil {
		failf("Failed to convert JSON to JUnit XML: %s", err)
	}

	// Write JUnit XML to file
	outputPath := filepath.Join(config.OutputDir, config.JUnitFilename)
	log.Infof("Writing JUnit XML to file: %s", outputPath)
	if err := os.WriteFile(outputPath, junitXML, 0644); err != nil {
		failf("Failed to write JUnit XML to file: %s", err)
	}

	// Export output
	if err := exportOutput("XCRESULT_TO_JUNIT_OUTPUT_PATH", outputPath); err != nil {
		failf("Failed to export output: %s", err)
	}

	log.Donef("XCResult successfully converted to JUnit XML")
}

// convertXCResultToJSON executes xcrun xcresulttool to get test results as JSON
func convertXCResultToJSON(xcresultPath string) ([]byte, error) {
	cmd := exec.Command("xcrun", "xcresulttool", "get", "test-results", "tests", "--path", xcresultPath)
	output, err := cmd.Output()
	if err != nil {
		//var exitErr *exec.ExitError
		if err, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("command failed with exit code %d: %s", err.ExitCode(), string(err.Stderr))
		}
		return nil, fmt.Errorf("failed to execute command: %w", err)
	}

	log.Debugf("XCResult JSON output length: %d bytes", len(output))
	return output, nil
}

// exportOutput exports a step output
func exportOutput(key, value string) error {
	cmd := exec.Command("envman", "add", "--key", key, "--value", value)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

// failf prints an error message and exits
func failf(format string, args ...interface{}) {
	log.Errorf(format, args...)
	os.Exit(1)
}
