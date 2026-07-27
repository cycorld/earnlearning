package application

import "testing"

// #176 — 첨부 파일명 검증/정리 단위 테스트.
func TestValidateMilestoneFilename(t *testing.T) {
	bad := []string{"", "../../etc/passwd.html", "a/b.html", `..\..\evil.html`, "plan\x00.html",
		"plan\r\nX-Evil: 1.html", `q"uote.html`, ".", ".."}
	for _, n := range bad {
		if err := validateMilestoneFilename(n); err == nil {
			t.Errorf("validateMilestoneFilename(%q) = nil, want error", n)
		}
	}
	good := []string{"plan.html", "사업계획서 최종.html", "plan (v2).pdf", "a-b_c.docx"}
	for _, n := range good {
		if err := validateMilestoneFilename(n); err != nil {
			t.Errorf("validateMilestoneFilename(%q) = %v, want nil", n, err)
		}
	}
}

func TestSanitizeDownloadFilename(t *testing.T) {
	cases := map[string]string{
		"plan.html":            "plan.html",
		"../../evil.html":      "evil.html",
		`..\..\evil.html`:      "evil.html",
		"plan\r\nX-Evil: 1":    "planX-Evil: 1",
		`q"uote.html`:          "quote.html",
		"":                     "fallback.bin",
		"..":                   "fallback.bin",
		"사업계획서.html": "사업계획서.html",
	}
	for in, want := range cases {
		if got := SanitizeDownloadFilename(in, "fallback.bin"); got != want {
			t.Errorf("SanitizeDownloadFilename(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestIsHTMLAttachmentExt(t *testing.T) {
	for _, e := range []string{".html", ".HTML", ".htm"} {
		if !IsHTMLAttachmentExt(e) {
			t.Errorf("IsHTMLAttachmentExt(%q) = false", e)
		}
	}
	for _, e := range []string{".pdf", ".txt", "", ".htmlx"} {
		if IsHTMLAttachmentExt(e) {
			t.Errorf("IsHTMLAttachmentExt(%q) = true", e)
		}
	}
}
