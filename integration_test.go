package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestCLIIntegration tests the complete CLI workflow
func TestCLIIntegration(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "rss_integration_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Build the binary for testing
	binaryPath := filepath.Join(tempDir, "rss-agg-test")
	cmd := exec.Command("go", "build", "-o", binaryPath)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}

	validRSS := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
<channel>
<title>Integration Test Feed</title>
<description>Test feed for integration testing</description>
<link>http://example.com</link>
<item>
<title>Integration Test Item 1</title>
<link>http://example.com/item1</link>
<description>First test item</description>
<pubDate>Wed, 01 Jan 2020 12:00:00 GMT</pubDate>
</item>
<item>
<title>Integration Test Item 2</title>
<link>http://example.com/item2</link>
<description>Second test item</description>
<pubDate>Thu, 02 Jan 2020 12:00:00 GMT</pubDate>
</item>
</channel>
</rss>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		fmt.Fprint(w, validRSS)
	}))
	defer server.Close()

	t.Run("all mode with input file", func(t *testing.T) {
		// Create input file
		inputFile := filepath.Join(tempDir, "test_feeds.txt")
		feedContent := fmt.Sprintf("# Test feeds\n%s\n", server.URL)
		err := os.WriteFile(inputFile, []byte(feedContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create input file: %v", err)
		}

		// Create output file path
		outputFile := filepath.Join(tempDir, "test_output.xml")

		// Run the CLI
		cmd := exec.Command(binaryPath, 
			"-input", inputFile,
			"-output", outputFile,
			"-count", "2",
			"-mode", "all")
		
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("CLI command failed: %v\nOutput: %s", err, output)
		}

		// Verify output file was created
		if _, err := os.Stat(outputFile); os.IsNotExist(err) {
			t.Errorf("Output file was not created")
			return
		}

		// Read and verify output content
		content, err := os.ReadFile(outputFile)
		if err != nil {
			t.Errorf("Failed to read output file: %v", err)
			return
		}

		contentStr := string(content)
		if !strings.Contains(contentStr, "Integration Test Item 1") {
			t.Errorf("Output file does not contain expected item 1")
		}
		if !strings.Contains(contentStr, "Integration Test Item 2") {
			t.Errorf("Output file does not contain expected item 2")
		}
	})

	t.Run("single mode", func(t *testing.T) {
		outputFile := filepath.Join(tempDir, "single_output.xml")

		// Run the CLI in single mode
		cmd := exec.Command(binaryPath,
			"-mode", "single",
			"-single-url", server.URL,
			"-output", outputFile,
			"-count", "1")

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("CLI command failed: %v\nOutput: %s", err, output)
		}

		// Verify output file was created
		if _, err := os.Stat(outputFile); os.IsNotExist(err) {
			t.Errorf("Output file was not created")
			return
		}

		// Read and verify output content
		content, err := os.ReadFile(outputFile)
		if err != nil {
			t.Errorf("Failed to read output file: %v", err)
			return
		}

		contentStr := string(content)
		if !strings.Contains(contentStr, "Integration Test Item") {
			t.Errorf("Output file does not contain expected items")
		}
	})

	t.Run("error cases", func(t *testing.T) {
		// Test missing input file in all mode
		cmd := exec.Command(binaryPath,
			"-mode", "all",
			"-output", filepath.Join(tempDir, "error_output.xml"))

		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Errorf("Expected error for missing input file, but command succeeded")
		}
		if !strings.Contains(string(output), "input file must be provided") {
			t.Errorf("Expected specific error message about input file")
		}

		// Test missing single-url in single mode
		cmd = exec.Command(binaryPath,
			"-mode", "single",
			"-output", filepath.Join(tempDir, "error_output2.xml"))

		output, err = cmd.CombinedOutput()
		if err == nil {
			t.Errorf("Expected error for missing single-url, but command succeeded")
		}
		if !strings.Contains(string(output), "single-url must be provided") {
			t.Errorf("Expected specific error message about single-url")
		}

		// Test invalid mode
		cmd = exec.Command(binaryPath,
			"-mode", "invalid",
			"-output", filepath.Join(tempDir, "error_output3.xml"))

		output, err = cmd.CombinedOutput()
		if err == nil {
			t.Errorf("Expected error for invalid mode, but command succeeded")
		}
		if !strings.Contains(string(output), "mode must be 'single' or 'all'") {
			t.Errorf("Expected specific error message about invalid mode")
		}
	})

	t.Run("help flag", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "--help")
		output, err := cmd.CombinedOutput()
		
		// Help should exit with code 2, which is normal for flag package
		if err != nil {
			if exitError, ok := err.(*exec.ExitError); ok {
				if exitError.ExitCode() != 2 {
					t.Errorf("Help flag should exit with code 2, got %d", exitError.ExitCode())
				}
			}
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "Usage of") {
			t.Errorf("Help output should contain usage information")
		}
		if !strings.Contains(outputStr, "-input") {
			t.Errorf("Help output should contain -input flag")
		}
		if !strings.Contains(outputStr, "-output") {
			t.Errorf("Help output should contain -output flag")
		}
		if !strings.Contains(outputStr, "-mode") {
			t.Errorf("Help output should contain -mode flag")
		}
	})

	t.Run("count limiting", func(t *testing.T) {
		// Create input file
		inputFile := filepath.Join(tempDir, "count_test_feeds.txt")
		feedContent := fmt.Sprintf("%s\n", server.URL)
		err := os.WriteFile(inputFile, []byte(feedContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create input file: %v", err)
		}

		outputFile := filepath.Join(tempDir, "count_test_output.xml")

		// Run with count=1 to limit items
		cmd := exec.Command(binaryPath,
			"-input", inputFile,
			"-output", outputFile,
			"-count", "1",
			"-mode", "all")

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("CLI command failed: %v\nOutput: %s", err, output)
		}

		// Read output and verify only 1 item is included
		content, err := os.ReadFile(outputFile)
		if err != nil {
			t.Errorf("Failed to read output file: %v", err)
			return
		}

		contentStr := string(content)
		// Count occurrences of <item> tags (should be only 1)
		itemCount := strings.Count(contentStr, "<item>")
		if itemCount != 1 {
			t.Errorf("Expected 1 item in output, found %d", itemCount)
		}
	})
}