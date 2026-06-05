package milestone

import (
	"reflect"
	"testing"
)

func TestIsValidMilestoneURL(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want bool
	}{
		// vercel.app, netlify.app and custom domains → 인정
		{"vercel.app", "https://my-mvp.vercel.app", true},
		{"vercel subdomain", "https://my-mvp.vercel.app/path", true},
		{"custom domain", "https://example.com", true},
		{"custom subdomain", "https://www.example.com/foo", true},
		{"netlify", "https://my-mvp.netlify.app", true},
		{"github.io", "https://student.github.io/project", true},

		// 연습용 (deny list)
		{"ai.studio root", "https://ai.studio/apps/123", false},
		{"aistudio.google.com", "https://aistudio.google.com/prompts/new", false},
		{"claude.ai", "https://claude.ai/chat/abc", false},
		{"chatgpt.com", "https://chatgpt.com/c/123", false},
		{"chat.openai.com", "https://chat.openai.com/c/123", false},
		{"gemini", "https://gemini.google.com/app", false},
		{"localhost", "http://localhost:3000", false},
		{"127.0.0.1", "http://127.0.0.1:5173", false},

		// subdomain of a denied host should also be denied
		{"sub.claude.ai", "https://www.claude.ai/foo", false},
		{"sub.chatgpt.com", "https://api.chatgpt.com", false},

		// invalid
		{"empty", "", false},
		{"no scheme", "example.com", false},
		{"ftp", "ftp://example.com", false},
		{"file", "file:///etc/passwd", false},
		{"whitespace only", "   ", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := IsValidMilestoneURL(c.in)
			if got != c.want {
				t.Errorf("IsValidMilestoneURL(%q) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}

func TestFilterValidURLs(t *testing.T) {
	in := []string{
		"https://my-mvp.vercel.app",
		"https://aistudio.google.com/prompts/1",
		"https://example.com",
		"",
		"https://claude.ai/chat/xyz",
		"https://netlify.app/dummy",
	}
	want := []string{
		"https://my-mvp.vercel.app",
		"https://example.com",
		"https://netlify.app/dummy",
	}
	got := FilterValidURLs(in)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("FilterValidURLs = %v, want %v", got, want)
	}
}

func TestParseCommaSeparated(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"https://a.com", []string{"https://a.com"}},
		{"https://a.com, https://b.com ", []string{"https://a.com", "https://b.com"}},
		{",,, https://a.com,, ", []string{"https://a.com"}},
	}
	for _, c := range cases {
		got := ParseCommaSeparated(c.in)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("ParseCommaSeparated(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestExtractURLsFromText(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{
			name: "single url",
			in:   "저희 MVP는 https://my-mvp.vercel.app 입니다.",
			want: []string{"https://my-mvp.vercel.app"},
		},
		{
			name: "multiple urls",
			in:   "랜딩: https://my.vercel.app, 데모: https://demo.example.com 보세요",
			want: []string{"https://my.vercel.app", "https://demo.example.com"},
		},
		{
			name: "no urls",
			in:   "그냥 텍스트",
			want: []string{},
		},
		{
			name: "trailing punctuation stripped",
			in:   "(see https://a.vercel.app),",
			want: []string{"https://a.vercel.app"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ExtractURLsFromText(c.in)
			if len(got) != len(c.want) {
				t.Fatalf("ExtractURLsFromText(%q) = %v, want %v", c.in, got, c.want)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Errorf("got[%d] = %q, want %q", i, got[i], c.want[i])
				}
			}
		})
	}
}
