package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"golang.org/x/text/encoding/charmap"
)

const feedURL = "https://secure.runescape.com/m=news/latest_news.rss?oldschool=true"

type RSS struct {
	Channel Channel `xml:"channel"`
}

type Channel struct {
	Title     string `xml:"title"`
	BuildDate string `xml:"lastBuildDate"`
	Items     []Item `xml:"item"`
}

type Item struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	PubDate     string `xml:"pubDate"`
	Category    string `xml:"category"`
	Description string `xml:"description"`
}

func main() {
	count    := flag.Int("n", 5, "number of articles to show (0 = all)")
	category := flag.String("c", "", "filter by category: 'Game Updates', 'Community', 'Technical'")
	full     := flag.Bool("full", false, "show article description")
	openFlag := flag.Bool("open", false, "open newest matching article in browser")
	noColor  := flag.Bool("no-color", false, "disable ANSI colors")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "osrs-news — fetch latest Old School RuneScape news\n\n")
		fmt.Fprintf(os.Stderr, "Usage: osrs-news [flags]\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(feedURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: fetch failed: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Fprintf(os.Stderr, "error: HTTP %d from feed\n", resp.StatusCode)
		os.Exit(1)
	}

	// Feed declares ISO-8859-1; decode to UTF-8 before XML parsing
	utf8Body := charmap.ISO8859_1.NewDecoder().Reader(resp.Body)
	rawBytes, err := io.ReadAll(utf8Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: read failed: %v\n", err)
		os.Exit(1)
	}
	// Strip encoding declaration so xml.Decoder doesn't re-apply it
	xmlStr := strings.Replace(string(rawBytes), ` encoding="ISO-8859-1"`, "", 1)

	var rss RSS
	if err := xml.NewDecoder(strings.NewReader(xmlStr)).Decode(&rss); err != nil {
		fmt.Fprintf(os.Stderr, "error: parse failed: %v\n", err)
		os.Exit(1)
	}

	items := rss.Channel.Items

	if *category != "" {
		var filtered []Item
		for _, item := range items {
			if strings.EqualFold(item.Category, *category) {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}

	if len(items) == 0 {
		fmt.Println("no articles found")
		return
	}

	if *openFlag {
		openBrowser(items[0].Link)
		return
	}

	if *count > 0 && len(items) > *count {
		items = items[:*count]
	}

	printHeader(rss.Channel, *noColor)
	for i, item := range items {
		printItem(i+1, item, *full, *noColor)
	}
}

func printHeader(ch Channel, noColor bool) {
	bold, reset, dim := "\033[1m", "\033[0m", "\033[2m"
	if noColor {
		bold, reset, dim = "", "", ""
	}

	built := ch.BuildDate
	if t, err := time.Parse("Mon, 02 Jan 2006 15:04:05 MST", ch.BuildDate); err == nil {
		built = t.Format("02 Jan 2006 15:04 MST")
	}

	fmt.Printf("\n%sOld School RuneScape — News%s  %s(updated %s)%s\n", bold, reset, dim, built, reset)
	fmt.Println(strings.Repeat("─", 60))
}

func printItem(n int, item Item, full bool, noColor bool) {
	bold   := "\033[1m"
	cyan   := "\033[36m"
	yellow := "\033[33m"
	green  := "\033[32m"
	reset  := "\033[0m"
	dim    := "\033[2m"
	if noColor {
		bold, cyan, yellow, green, reset, dim = "", "", "", "", "", ""
	}

	date := item.PubDate
	if t, err := time.Parse("Mon, 02 Jan 2006 15:04:05 MST", item.PubDate); err == nil {
		date = t.Format("02 Jan 2006")
	}

	fmt.Printf("%s%d. %s%s\n", bold, n, item.Title, reset)
	fmt.Printf("   %s%-15s%s %s%s%s\n", green, item.Category, reset, yellow, date, reset)

	if full && item.Description != "" {
		desc := stripHTML(strings.TrimSpace(item.Description))
		fmt.Printf("   %s%s%s\n", cyan, wrapText(desc, 72, "   "), reset)
	}

	fmt.Printf("   %s%s%s\n\n", dim, item.Link, reset)
}

func stripHTML(s string) string {
	var b strings.Builder
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
			b.WriteByte(' ')
		case !inTag:
			b.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func wrapText(s string, width int, indent string) string {
	words := strings.Fields(s)
	var lines []string
	var line strings.Builder
	for _, w := range words {
		if line.Len() > 0 && line.Len()+1+len(w) > width {
			lines = append(lines, line.String())
			line.Reset()
			line.WriteString(indent)
		}
		if line.Len() == 0 {
			line.WriteString(indent)
		} else {
			line.WriteByte(' ')
		}
		line.WriteString(w)
	}
	if line.Len() > 0 {
		lines = append(lines, line.String())
	}
	return strings.Join(lines, "\n")
}

func openBrowser(url string) {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "linux":
		cmd, args = "xdg-open", []string{url}
	case "darwin":
		cmd, args = "open", []string{url}
	case "windows":
		cmd, args = "cmd", []string{"/c", "start", url}
	default:
		fmt.Println(url)
		return
	}
	if err := exec.Command(cmd, args...).Start(); err != nil {
		fmt.Println(url)
	}
}
