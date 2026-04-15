# osrs-news

CLI tool to fetch the latest news and updates from [Old School RuneScape](https://oldschool.runescape.com).

## Installation

Download the binary for your platform from the [releases page](https://github.com/Blizzard-fs/osrs-news/releases/latest), then follow the steps below.

### Linux

```bash
cp osrs-news-linux-amd64 ~/.local/bin/osrs-news
chmod +x ~/.local/bin/osrs-news
```

> `~/.local/bin` is on `$PATH` by default on most Linux distros. If `osrs-news` still isn't found, add this to your `~/.bashrc` or `~/.zshrc` and restart your terminal:
> ```bash
> export PATH="$HOME/.local/bin:$PATH"
> ```

### macOS

```bash
# Intel
cp osrs-news-mac-amd64 /usr/local/bin/osrs-news

# Apple Silicon (M1/M2/M3)
cp osrs-news-mac-arm64 /usr/local/bin/osrs-news

chmod +x /usr/local/bin/osrs-news
```

> **Gatekeeper warning:** macOS may block the binary on first run since it isn't signed. To fix this, run once:
> ```bash
> xattr -d com.apple.quarantine /usr/local/bin/osrs-news
> ```
> Alternatively, right-click the file in Finder → Open → Open.

### Windows

Rename `osrs-news-windows-amd64.exe` to `osrs-news.exe` and move it to a folder on your `PATH`, for example `C:\Windows\System32\` or a custom folder you've added to your user `PATH` in System Settings.

### Build from source

Requires [Go](https://go.dev/dl/).

```bash
git clone git@github.com:Blizzard-fs/osrs-news.git
cd osrs-news
go build -o osrs-news .
cp osrs-news ~/.local/bin/
```

## Usage

```
osrs-news [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `-n <count>` | `5` | Number of articles to show. Use `0` for all. |
| `-c <category>` | _(all)_ | Filter by category: `Game Updates`, `Community`, or `Technical`. |
| `-read <N>` | off | Fetch and display the full content of article N in the list. |
| `-full` | off | Show short description below each result in the list. |
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

Read the full content of article 1 in your terminal:
```bash
osrs-news -read 1
```

Read the latest Game Update in full:
```bash
osrs-news -c "Game Updates" -read 1
```

Plain output (no colors):
```bash
osrs-news -no-color
```

## Source

News is pulled from the official OSRS RSS feed:
`https://secure.runescape.com/m=news/latest_news.rss?oldschool=true`
