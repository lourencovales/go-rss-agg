package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/feeds"
)

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config - all mode",
			config: &Config{
				InputFile:  "test.txt",
				Count:      10,
				Mode:       "all",
				OutputFile: "output.xml",
			},
			wantErr: false,
		},
		{
			name: "valid config - single mode",
			config: &Config{
				Count:      5,
				Mode:       "single",
				SingleURL:  "http://example.com/rss",
				OutputFile: "output.xml",
			},
			wantErr: false,
		},
		{
			name: "invalid mode",
			config: &Config{
				InputFile:  "test.txt",
				Count:      10,
				Mode:       "invalid",
				OutputFile: "output.xml",
			},
			wantErr: true,
			errMsg:  "mode must be 'single' or 'all'",
		},
		{
			name: "single mode missing URL",
			config: &Config{
				Count:      10,
				Mode:       "single",
				OutputFile: "output.xml",
			},
			wantErr: true,
			errMsg:  "single-url must be provided when mode is 'single'",
		},
		{
			name: "all mode missing input file",
			config: &Config{
				Count:      10,
				Mode:       "all",
				OutputFile: "output.xml",
			},
			wantErr: true,
			errMsg:  "input file must be provided when mode is 'all'",
		},
		{
			name: "zero count",
			config: &Config{
				InputFile:  "test.txt",
				Count:      0,
				Mode:       "all",
				OutputFile: "output.xml",
			},
			wantErr: true,
			errMsg:  "count must be greater than 0",
		},
		{
			name: "negative count",
			config: &Config{
				InputFile:  "test.txt",
				Count:      -5,
				Mode:       "all",
				OutputFile: "output.xml",
			},
			wantErr: true,
			errMsg:  "count must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)
			if tt.wantErr {
				if err == nil {
					t.Errorf("validateConfig() expected error but got none")
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validateConfig() error = %v, want error containing %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateConfig() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestReadURLsFromFile(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "rss_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name     string
		content  string
		expected []string
		wantErr  bool
	}{
		{
			name: "valid URLs with comments",
			content: `# This is a comment
http://example.com/feed1.xml
https://example.com/feed2.xml
# Another comment
http://example.com/feed3.xml

# Empty line above should be ignored`,
			expected: []string{
				"http://example.com/feed1.xml",
				"https://example.com/feed2.xml",
				"http://example.com/feed3.xml",
			},
			wantErr: false,
		},
		{
			name: "only comments and empty lines",
			content: `# Comment 1
# Comment 2

# Comment 3`,
			expected: []string{},
			wantErr:  false,
		},
		{
			name: "URLs with whitespace",
			content: `  http://example.com/feed1.xml  
	https://example.com/feed2.xml	
http://example.com/feed3.xml`,
			expected: []string{
				"http://example.com/feed1.xml",
				"https://example.com/feed2.xml",
				"http://example.com/feed3.xml",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file
			testFile := filepath.Join(tempDir, "test.txt")
			err := os.WriteFile(testFile, []byte(tt.content), 0644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			urls, err := readURLsFromFile(testFile)
			if tt.wantErr {
				if err == nil {
					t.Errorf("readURLsFromFile() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("readURLsFromFile() unexpected error = %v", err)
				return
			}

			if len(urls) != len(tt.expected) {
				t.Errorf("readURLsFromFile() got %d URLs, want %d", len(urls), len(tt.expected))
				return
			}

			for i, url := range urls {
				if url != tt.expected[i] {
					t.Errorf("readURLsFromFile() URL[%d] = %v, want %v", i, url, tt.expected[i])
				}
			}
		})
	}

	// Test file not found
	t.Run("file not found", func(t *testing.T) {
		_, err := readURLsFromFile("nonexistent.txt")
		if err == nil {
			t.Errorf("readURLsFromFile() expected error for nonexistent file")
		}
	})
}

func TestOutputFeed(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "rss_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test feed
	feed := &feeds.Feed{
		Title:       "Test Feed",
		Link:        &feeds.Link{Href: "http://example.com"},
		Description: "Test feed description",
		Created:     time.Now(),
		Items: []*feeds.Item{
			{
				Title:       "Test Item 1",
				Link:        &feeds.Link{Href: "http://example.com/item1"},
				Description: "Test item 1 description",
				Created:     time.Now(),
			},
		},
	}

	outputFile := filepath.Join(tempDir, "test_output.xml")
	err = outputFeed(feed, outputFile)
	if err != nil {
		t.Errorf("outputFeed() unexpected error = %v", err)
		return
	}

	// Check if file was created
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Errorf("outputFeed() did not create output file")
		return
	}

	// Read and verify file content
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Errorf("Failed to read output file: %v", err)
		return
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "Test Feed") {
		t.Errorf("Output file does not contain expected feed title")
	}
	if !strings.Contains(contentStr, "Test Item 1") {
		t.Errorf("Output file does not contain expected item title")
	}
}

func createMockRSSServer(rssContent string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		fmt.Fprint(w, rssContent)
	}))
}

func TestFetchFeedItems(t *testing.T) {
	validRSS := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
<channel>
<title>Test Feed</title>
<description>Test Description</description>
<link>http://example.com</link>
<item>
<title>Test Item 1</title>
<link>http://example.com/item1</link>
<description>Test item 1 description</description>
<pubDate>Wed, 01 Jan 2020 00:00:00 GMT</pubDate>
</item>
<item>
<title>Test Item 2</title>
<link>http://example.com/item2</link>
<description>Test item 2 description</description>
<pubDate>Thu, 02 Jan 2020 00:00:00 GMT</pubDate>
</item>
</channel>
</rss>`

	server := createMockRSSServer(validRSS)
	defer server.Close()

	items, err := fetchFeedItems(server.URL)
	if err != nil {
		t.Errorf("fetchFeedItems() unexpected error = %v", err)
		return
	}

	if len(items) != 2 {
		t.Errorf("fetchFeedItems() got %d items, want 2", len(items))
		return
	}

	if items[0].Title != "Test Item 1" {
		t.Errorf("fetchFeedItems() first item title = %v, want 'Test Item 1'", items[0].Title)
	}

	if items[1].Title != "Test Item 2" {
		t.Errorf("fetchFeedItems() second item title = %v, want 'Test Item 2'", items[1].Title)
	}

	// Test invalid URL
	_, err = fetchFeedItems("invalid-url")
	if err == nil {
		t.Errorf("fetchFeedItems() expected error for invalid URL")
	}
}

func TestAggregateFeedsSingleMode(t *testing.T) {
	validRSS := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
<channel>
<title>Test Feed</title>
<description>Test Description</description>
<link>http://example.com</link>
<item>
<title>Test Item 1</title>
<link>http://example.com/item1</link>
<description>Test item 1 description</description>
<pubDate>Wed, 01 Jan 2020 00:00:00 GMT</pubDate>
</item>
</channel>
</rss>`

	server := createMockRSSServer(validRSS)
	defer server.Close()

	config := &Config{
		Mode:      "single",
		SingleURL: server.URL,
		Count:     5,
	}

	feed, err := aggregateFeeds(config)
	if err != nil {
		t.Errorf("aggregateFeeds() unexpected error = %v", err)
		return
	}

	if len(feed.Items) != 1 {
		t.Errorf("aggregateFeeds() got %d items, want 1", len(feed.Items))
	}

	if feed.Items[0].Title != "Test Item 1" {
		t.Errorf("aggregateFeeds() item title = %v, want 'Test Item 1'", feed.Items[0].Title)
	}
}

func TestAggregateFeedsAllMode(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "rss_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	validRSS1 := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
<channel>
<title>Feed 1</title>
<description>Test Description</description>
<link>http://example.com</link>
<item>
<title>Item from Feed 1</title>
<link>http://example.com/item1</link>
<description>Item 1 description</description>
<pubDate>Wed, 01 Jan 2020 00:00:00 GMT</pubDate>
</item>
</channel>
</rss>`

	validRSS2 := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
<channel>
<title>Feed 2</title>
<description>Test Description</description>
<link>http://example.com</link>
<item>
<title>Item from Feed 2</title>
<link>http://example.com/item2</link>
<description>Item 2 description</description>
<pubDate>Thu, 02 Jan 2020 00:00:00 GMT</pubDate>
</item>
</channel>
</rss>`

	server1 := createMockRSSServer(validRSS1)
	defer server1.Close()

	server2 := createMockRSSServer(validRSS2)
	defer server2.Close()

	// Create input file with both server URLs
	inputFile := filepath.Join(tempDir, "feeds.txt")
	content := fmt.Sprintf("%s\n%s\n", server1.URL, server2.URL)
	err = os.WriteFile(inputFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create input file: %v", err)
	}

	config := &Config{
		Mode:      "all",
		InputFile: inputFile,
		Count:     5,
	}

	feed, err := aggregateFeeds(config)
	if err != nil {
		t.Errorf("aggregateFeeds() unexpected error = %v", err)
		return
	}

	if len(feed.Items) != 2 {
		t.Errorf("aggregateFeeds() got %d items, want 2", len(feed.Items))
		return
	}

	// Items should be sorted by date (most recent first)
	// Feed 2 item has a later date, so it should be first
	if feed.Items[0].Title != "Item from Feed 2" {
		t.Errorf("aggregateFeeds() first item title = %v, want 'Item from Feed 2'", feed.Items[0].Title)
	}

	if feed.Items[1].Title != "Item from Feed 1" {
		t.Errorf("aggregateFeeds() second item title = %v, want 'Item from Feed 1'", feed.Items[1].Title)
	}
}