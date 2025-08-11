package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/SlyMarbo/rss"
	"github.com/gorilla/feeds"
)

type Config struct {
	InputFile  string
	Count      int
	Mode       string // "single" or "all"
	SingleURL  string
	OutputFile string
}

func main() {
	var (
		inputFile = flag.String("input", "", "Input file containing RSS feed URLs (one per line)")
		count     = flag.Int("count", 10, "Number of items to include")
		mode      = flag.String("mode", "all", "Mode: 'single' for one source, 'all' for all sources")
		singleURL = flag.String("single-url", "", "Single RSS feed URL (when mode=single)")
		outputFile = flag.String("output", "aggregated.xml", "Output file path")
	)
	flag.Parse()

	config := &Config{
		InputFile:  *inputFile,
		Count:      *count,
		Mode:       *mode,
		SingleURL:  *singleURL,
		OutputFile: *outputFile,
	}

	if err := validateConfig(config); err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	aggregatedFeed, err := aggregateFeeds(config)
	if err != nil {
		log.Fatalf("Error aggregating feeds: %v", err)
	}

	if err := outputFeed(aggregatedFeed, config.OutputFile); err != nil {
		log.Fatalf("Error outputting feed: %v", err)
	}
}

func validateConfig(config *Config) error {
	if config.Mode != "single" && config.Mode != "all" {
		return fmt.Errorf("mode must be 'single' or 'all'")
	}

	if config.Mode == "single" {
		if config.SingleURL == "" {
			return fmt.Errorf("single-url must be provided when mode is 'single'")
		}
	} else {
		if config.InputFile == "" {
			return fmt.Errorf("input file must be provided when mode is 'all'")
		}
	}

	if config.Count <= 0 {
		return fmt.Errorf("count must be greater than 0")
	}

	return nil
}

func aggregateFeeds(config *Config) (*feeds.Feed, error) {
	var allItems []*feeds.Item

	if config.Mode == "single" {
		items, err := fetchFeedItems(config.SingleURL)
		if err != nil {
			return nil, fmt.Errorf("error fetching single feed: %v", err)
		}
		allItems = items
	} else {
		urls, err := readURLsFromFile(config.InputFile)
		if err != nil {
			return nil, fmt.Errorf("error reading input file: %v", err)
		}

		var wg sync.WaitGroup
		var mu sync.Mutex

		for _, url := range urls {
			wg.Add(1)
			go func(url string) {
				defer wg.Done()
				items, err := fetchFeedItems(strings.TrimSpace(url))
				if err != nil {
					log.Printf("Warning: failed to fetch feed %s: %v", url, err)
					return
				}
				mu.Lock()
				allItems = append(allItems, items...)
				mu.Unlock()
			}(url)
		}
		wg.Wait()
	}

	sort.Slice(allItems, func(i, j int) bool {
		return allItems[i].Created.After(allItems[j].Created)
	})

	if len(allItems) > config.Count {
		allItems = allItems[:config.Count]
	}

	aggregatedFeed := &feeds.Feed{
		Title:       "RSS Aggregator Feed",
		Link:        &feeds.Link{Href: ""},
		Description: "Aggregated RSS feed",
		Created:     time.Now(),
		Items:       allItems,
	}

	return aggregatedFeed, nil
}

func readURLsFromFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()

	var urls []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			urls = append(urls, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %v", err)
	}

	return urls, nil
}

func fetchFeedItems(url string) ([]*feeds.Item, error) {
	feed, err := rss.Fetch(url)
	if err != nil {
		return nil, err
	}

	var items []*feeds.Item
	for _, item := range feed.Items {
		feedItem := &feeds.Item{
			Title:       item.Title,
			Link:        &feeds.Link{Href: item.Link},
			Description: item.Summary,
			Created:     item.Date,
		}

		if item.Content != "" {
			feedItem.Content = item.Content
		}

		items = append(items, feedItem)
	}

	return items, nil
}

func outputFeed(feed *feeds.Feed, outputFile string) error {
	rssString, err := feed.ToRss()
	if err != nil {
		return fmt.Errorf("error generating RSS: %v", err)
	}

	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("error creating output file: %v", err)
	}
	defer file.Close()

	_, err = file.WriteString(rssString)
	if err != nil {
		return fmt.Errorf("error writing to output file: %v", err)
	}

	return nil
}