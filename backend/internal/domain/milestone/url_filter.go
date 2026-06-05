package milestone

import (
	"net/url"
	"regexp"
	"strings"
)

// denyHostSuffixes — these hosts (and subdomains) are 연습용 only and never count as a real MVP.
// We match suffix to also exclude e.g. `studio.ai.studio` or `someone.aistudio.google.com`.
var denyHostSuffixes = []string{
	"aistudio.google.com",
	"ai.studio",
	"claude.ai",
	"chatgpt.com",
	"chat.openai.com",
	"gemini.google.com",
	"bard.google.com",
	"localhost",
	"127.0.0.1",
}

// urlPattern — extracts http(s) URLs from free text.
var urlPattern = regexp.MustCompile(`https?://[^\s<>()"']+`)

// IsValidMilestoneURL — true iff the URL is http(s) AND its host is not in the deny list.
// vercel.app, netlify.app, custom domains, etc. all count.
func IsValidMilestoneURL(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	host := strings.ToLower(u.Hostname())
	if host == "" {
		return false
	}
	for _, deny := range denyHostSuffixes {
		if host == deny || strings.HasSuffix(host, "."+deny) {
			return false
		}
	}
	return true
}

// FilterValidURLs — return only the URLs that pass IsValidMilestoneURL, preserving order.
func FilterValidURLs(urls []string) []string {
	out := make([]string, 0, len(urls))
	for _, u := range urls {
		if IsValidMilestoneURL(u) {
			out = append(out, u)
		}
	}
	return out
}

// ParseCommaSeparated — split a comma-separated URL string (matches frontend parseServiceUrls).
// Trims whitespace and drops empty pieces.
func ParseCommaSeparated(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// ExtractURLsFromText — find all http(s) URLs in free text (e.g. grant proposal body).
// Order preserved, no dedup (caller can dedup if needed).
func ExtractURLsFromText(text string) []string {
	if text == "" {
		return nil
	}
	matches := urlPattern.FindAllString(text, -1)
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		// Strip common trailing punctuation that regex may have included.
		m = strings.TrimRight(m, ".,;:!?")
		if m != "" {
			out = append(out, m)
		}
	}
	return out
}
