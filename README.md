# RSS Aggregator

A command-line tool to aggregate RSS feeds into a single XML file.

## Usage

### Aggregate multiple feeds
```bash
./rss-agg -input feeds.txt -count 20 -output aggregated.xml
```

### Aggregate single feed
```bash
./rss-agg -mode single -single-url https://example.com/rss.xml -count 10
```

## Options

- `-input`: File containing RSS URLs (one per line)
- `-mode`: "all" (default) or "single" 
- `-single-url`: RSS feed URL for single mode
- `-count`: Number of items to include (default: 10)
- `-output`: Output file name (default: aggregated.xml)

## Feed file format

```
# Comments start with #
https://feeds.bbci.co.uk/news/rss.xml
https://rss.cnn.com/rss/edition.rss
```

## Build

```bash
go build -o rss-agg
```

## Test

```bash
go test
```