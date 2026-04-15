# osrs-news

CLI tool to fetch the latest news and updates from [Old School RuneScape](https://oldschool.runescape.com).

## Installation

Clone the repo and either use the pre-built binary directly or build from source:

```bash
git clone git@github.com:Blizzard-fs/osrs-news.git
cd osrs-news

# Use pre-built binary (Linux x86_64)
./osrs-news

# Or build from source (requires Go)
go build -o osrs-news .
```

Optionally move the binary somewhere on your `$PATH`:

```bash
mv osrs-news ~/.local/bin/
```

## Usage

```
osrs-news [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `-n <count>` | `5` | Number of articles to show. Use `0` for all. |
| `-c <category>` | _(all)_ | Filter by category: `Game Updates`, `Community`, or `Technical`. |
| `-full` | off | Show article description below each result. |
| `-open` | off | Open the newest matching article in your browser. |
| `-no-color` | off | Disable ANSI colors (useful for piping or logging). |

## Examples

Show the 5 latest articles:
```bash
osrs-news
```

Show the 10 latest articles:
```bash
osrs-news -n 10
```

Show only Game Updates:
```bash
osrs-news -c "Game Updates"
```

Show all articles with descriptions:
```bash
osrs-news -n 0 -full
```

Open the latest article in your browser:
```bash
osrs-news -open
```

Open the latest Game Update in your browser:
```bash
osrs-news -c "Game Updates" -open
```

Plain output (no colors):
```bash
osrs-news -no-color
```

## Source

News is pulled from the official OSRS RSS feed:
`https://secure.runescape.com/m=news/latest_news.rss?oldschool=true`
