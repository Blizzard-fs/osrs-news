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
	read     := flag.Int("read", 0, "fetch and display full content of article N from the list")
	openFlag := flag.Bool("open", false, "open newest matching article in browser")
	noColor  := flag.Bool("no-color", false, "disable ANSI colors")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "osrs-news — fetch latest Old School RuneScape news\n\n")
		fmt.Fprintf(os.Stderr, "Usage: osrs-news [flags]\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	client := &http.Client{Timeout: 15 * time.Second}
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

	// -read N: fetch and render full article content
	if *read > 0 {
		if *read > len(items) {
			fmt.Fprintf(os.Stderr, "error: only %d articles available\n", len(items))
			os.Exit(1)
		}
		item := items[*read-1]
		if err := printFullArticle(client, item, *noColor); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
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

func printFullArticle(client *http.Client, item Item, noColor bool) error {
	resp, err := client.Get(item.Link)
	if err != nil {
		return fmt.Errorf("fetch article failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(charmap.ISO8859_1.NewDecoder().Reader(resp.Body))
	if err != nil {
		return fmt.Errorf("read article failed: %v", err)
	}
	html := string(body)

	// Different article layouts use different containers — try all known variants
	content := extractID(html, "newspost-content")
	if content == "" {
		content = extractClass(html, "news-article-content")
	}
	if content == "" {
		content = extractID(html, "background")
	}
	if content == "" {
		return fmt.Errorf("could not find article body on page")
	}

	bold   := "\033[1m"
	yellow := "\033[33m"
	green  := "\033[32m"
	reset  := "\033[0m"
	dim    := "\033[2m"
	if noColor {
		bold, yellow, green, reset, dim = "", "", "", "", ""
	}

	date := item.PubDate
	if t, err := time.Parse("Mon, 02 Jan 2006 15:04:05 MST", item.PubDate); err == nil {
		date = t.Format("02 Jan 2006")
	}

	fmt.Printf("\n%s%s%s\n", bold, item.Title, reset)
	fmt.Printf("%s%-15s%s %s%s%s\n", green, item.Category, reset, yellow, date, reset)
	fmt.Printf("%s%s%s\n", dim, item.Link, reset)
	fmt.Println(strings.Repeat("─", 60))
	fmt.Println()

	rendered := renderHTML(content, noColor)
	fmt.Println(rendered)
	return nil
}

// extractClass pulls the inner HTML of the first div with class="cls".
func extractClass(html, cls string) string {
	needle := `class="` + cls + `"`
	idx := strings.Index(html, needle)
	if idx == -1 {
		return ""
	}
	start := strings.LastIndex(html[:idx], "<")
	if start == -1 {
		return ""
	}
	end := strings.Index(html[start:], ">")
	if end == -1 {
		return ""
	}
	content := html[start+end+1:]
	depth := 1
	pos := 0
	for pos < len(content) && depth > 0 {
		open := strings.Index(content[pos:], "<div")
		close := strings.Index(content[pos:], "</div>")
		if close == -1 {
			break
		}
		if open != -1 && open < close {
			depth++
			pos += open + 4
		} else {
			depth--
			if depth == 0 {
				return content[:pos+close]
			}
			pos += close + 6
		}
	}
	return content
}

// extractID pulls the inner HTML of the first element with id="id".
func extractID(html, id string) string {
	needle := `id="` + id + `"`
	idx := strings.Index(html, needle)
	if idx == -1 {
		return ""
	}
	// find the opening tag start
	start := strings.LastIndex(html[:idx], "<")
	if start == -1 {
		return ""
	}
	// find end of opening tag
	end := strings.Index(html[start:], ">")
	if end == -1 {
		return ""
	}
	content := html[start+end+1:]
	// find closing </div>
	depth := 1
	pos := 0
	for pos < len(content) && depth > 0 {
		open := strings.Index(content[pos:], "<div")
		close := strings.Index(content[pos:], "</div>")
		if close == -1 {
			break
		}
		if open != -1 && open < close {
			depth++
			pos += open + 4
		} else {
			depth--
			if depth == 0 {
				return content[:pos+close]
			}
			pos += close + 6
		}
	}
	return content
}

// renderHTML converts HTML to readable plain text with basic formatting.
func renderHTML(html string, noColor bool) string {
	bold  := "\033[1m"
	reset := "\033[0m"
	dim   := "\033[2m"
	if noColor {
		bold, reset, dim = "", "", ""
	}

	var out strings.Builder
	pos := 0
	var buf strings.Builder

	flushBuf := func() {
		text := strings.Join(strings.Fields(buf.String()), " ")
		buf.Reset()
		if text == "" {
			return
		}
		out.WriteString(wrapText(text, 76, "") + "\n")
	}

	for pos < len(html) {
		if html[pos] != '<' {
			// decode basic HTML entities inline
			if strings.HasPrefix(html[pos:], "&amp;") {
				buf.WriteByte('&')
				pos += 5
			} else if strings.HasPrefix(html[pos:], "&lt;") {
				buf.WriteByte('<')
				pos += 4
			} else if strings.HasPrefix(html[pos:], "&gt;") {
				buf.WriteByte('>')
				pos += 4
			} else if strings.HasPrefix(html[pos:], "&nbsp;") {
				buf.WriteByte(' ')
				pos += 6
			} else if strings.HasPrefix(html[pos:], "&#39;") {
				buf.WriteByte('\'')
				pos += 5
			} else if strings.HasPrefix(html[pos:], "&quot;") {
				buf.WriteByte('"')
				pos += 6
			} else {
				buf.WriteByte(html[pos])
				pos++
			}
			continue
		}

		// inside a tag
		end := strings.Index(html[pos:], ">")
		if end == -1 {
			break
		}
		tag := html[pos+1 : pos+end]
		pos += end + 1
		tagLower := strings.ToLower(strings.Fields(tag)[0])
		isClose := strings.HasPrefix(tagLower, "/")
		if isClose {
			tagLower = tagLower[1:]
		}

		switch tagLower {
		case "p", "div", "br", "tr":
			flushBuf()
			if tagLower != "br" && !isClose {
				// nothing extra
			} else if tagLower == "br" {
				out.WriteByte('\n')
			}
		case "h1", "h2", "h3", "h4":
			flushBuf()
			if !isClose {
				buf.WriteString(bold)
			} else {
				text := strings.TrimSpace(buf.String())
				buf.Reset()
				if text != "" {
					out.WriteString("\n" + bold + strings.ToUpper(text) + reset + "\n")
				}
			}
		case "li":
			flushBuf()
			if !isClose {
				buf.WriteString("  • ")
			}
		case "b", "strong":
			if !isClose {
				buf.WriteString(bold)
			} else {
				buf.WriteString(reset)
			}
		case "i", "em":
			if !isClose {
				buf.WriteString(dim)
			} else {
				buf.WriteString(reset)
			}
		case "th":
			flushBuf()
			if !isClose {
				buf.WriteString(bold + "| ")
			} else {
				buf.WriteString(reset)
			}
		case "td":
			flushBuf()
			if !isClose {
				buf.WriteString("| ")
			}
		case "summary":
			flushBuf()
			if !isClose {
				buf.WriteString(dim + "▶ ")
			} else {
				text := strings.TrimSpace(buf.String())
				buf.Reset()
				out.WriteString(dim + text + reset + "\n")
			}
		case "script", "style":
			// skip until closing tag
			closeTag := "</" + tagLower
			skip := strings.Index(strings.ToLower(html[pos:]), closeTag)
			if skip != -1 {
				pos += skip
			}
		}
	}
	flushBuf()
	return out.String()
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
