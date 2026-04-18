package ragindex

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/earnlearning/backend/internal/domain/chat"
)

// --- fake wiki repository (satisfies chat.WikiRepository only for sync) ---

type fakeWiki struct {
	mu    sync.Mutex
	docs  map[string]struct{ title, body string }
	metas map[string]*chat.WikiDocMeta
}

func newFakeWiki() *fakeWiki {
	return &fakeWiki{
		docs:  map[string]struct{ title, body string }{},
		metas: map[string]*chat.WikiDocMeta{},
	}
}

func (f *fakeWiki) UpsertMeta(m *chat.WikiDocMeta) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.metas[m.Slug] = m
	return nil
}
func (f *fakeWiki) FindMeta(slug string) (*chat.WikiDocMeta, error) {
	return f.metas[slug], nil
}
func (f *fakeWiki) ListMeta() ([]*chat.WikiDocMeta, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]*chat.WikiDocMeta, 0, len(f.metas))
	for _, m := range f.metas {
		out = append(out, m)
	}
	return out, nil
}
func (f *fakeWiki) DeleteMeta(slug string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.metas, slug)
	return nil
}
func (f *fakeWiki) UpsertDoc(slug, title, body string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.docs[slug] = struct{ title, body string }{title, body}
	return nil
}
func (f *fakeWiki) DeleteDoc(slug string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.docs, slug)
	return nil
}
func (f *fakeWiki) Search(query string, scope []string, limit int) ([]*chat.WikiSearchHit, error) {
	return nil, nil
}
func (f *fakeWiki) Reset() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.docs = map[string]struct{ title, body string }{}
	return nil
}

// --- tests ---

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestLoader_SyncsAllMarkdownFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "wallet.md"), "# 내 지갑이란?\n잔액을 조회하는 법.")
	writeFile(t, filepath.Join(dir, "notion-manuals", "grant.md"),
		"---\ntitle: 정부과제 가이드\nnotion_page_id: abc123\n---\n\n본문 내용")
	writeFile(t, filepath.Join(dir, "README.md"), "# README\n") // 가 포함되면 됨

	w := newFakeWiki()
	l := NewLoader(w, dir)
	n, err := l.Sync()
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if n != 3 {
		t.Fatalf("indexed count: got %d, want 3", n)
	}
	// slug 확인: path 기반 + 확장자 제거
	if _, ok := w.docs["wallet"]; !ok {
		t.Errorf("missing wallet slug: %v", keys(w.docs))
	}
	if _, ok := w.docs["notion-manuals/grant"]; !ok {
		t.Errorf("missing notion-manuals/grant slug: %v", keys(w.docs))
	}
	// frontmatter title 우선 사용
	if w.metas["notion-manuals/grant"].Title != "정부과제 가이드" {
		t.Errorf("title from frontmatter: got %q", w.metas["notion-manuals/grant"].Title)
	}
	if w.metas["notion-manuals/grant"].NotionPageID != "abc123" {
		t.Errorf("notion_page_id not captured")
	}
	// title fallback: 첫 H1
	if w.metas["wallet"].Title != "내 지갑이란?" {
		t.Errorf("fallback title: got %q", w.metas["wallet"].Title)
	}
}

func TestLoader_DeletesOrphanedDocs(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.md"), "# A\nabc")
	writeFile(t, filepath.Join(dir, "b.md"), "# B\nbcd")

	w := newFakeWiki()
	l := NewLoader(w, dir)
	if _, err := l.Sync(); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	if len(w.docs) != 2 {
		t.Fatalf("initial count: %d", len(w.docs))
	}

	// b.md 삭제 후 재동기화
	os.Remove(filepath.Join(dir, "b.md"))
	if _, err := l.Sync(); err != nil {
		t.Fatalf("second sync: %v", err)
	}
	if len(w.docs) != 1 {
		t.Errorf("after delete: got %d, want 1", len(w.docs))
	}
	if _, ok := w.docs["b"]; ok {
		t.Errorf("b should be removed")
	}
}

func TestLoader_SkipsMissingDir(t *testing.T) {
	w := newFakeWiki()
	l := NewLoader(w, "/nonexistent/path/that/should/not/exist")
	n, err := l.Sync()
	if err != nil {
		t.Fatalf("missing dir should be ok: %v", err)
	}
	if n != 0 {
		t.Errorf("count: got %d, want 0", n)
	}
}

func keys(m map[string]struct{ title, body string }) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
