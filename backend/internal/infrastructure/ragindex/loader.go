// Package ragindex 는 `docs/llm-wiki/**/*.md` 파일을 SQLite FTS5 인덱스에
// 로드하는 RAG 지식베이스 관리 유틸.
//
// # 설계
//   - Source-of-truth 는 git 에 커밋된 .md 파일.
//   - 서버 기동 시 + 관리자가 "재인덱싱" 누를 때 전체 스캔 → FTS5 upsert.
//   - 각 파일의 frontmatter (YAML-lite: `--- key: value ---`) 에서 title,
//     notion_page_id 등 메타를 추출해 chat_wiki_meta 에 upsert.
//   - slug 는 `<relative path without .md>` — 예: `notion-manuals/wallet-guide`.
package ragindex

import (
	"bufio"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/earnlearning/backend/internal/domain/chat"
)

// Loader reads md files and synchronizes FTS5 index + meta table.
type Loader struct {
	wiki    chat.WikiRepository
	rootDir string // 예: "./docs/llm-wiki"
}

func NewLoader(wiki chat.WikiRepository, rootDir string) *Loader {
	return &Loader{wiki: wiki, rootDir: rootDir}
}

// Sync walks the root dir and upserts every .md file into FTS5.
// Returns the number of docs indexed and the first error encountered (other
// errors are logged and indexing continues).
func (l *Loader) Sync() (int, error) {
	if l.rootDir == "" {
		return 0, fmt.Errorf("ragindex: empty root dir")
	}
	info, err := os.Stat(l.rootDir)
	if err != nil {
		if os.IsNotExist(err) {
			// 디렉토리가 아예 없으면 조용히 skip — 개발 환경에서 발생 가능
			log.Printf("[ragindex] root dir %s not found, skipping sync", l.rootDir)
			return 0, nil
		}
		return 0, err
	}
	if !info.IsDir() {
		return 0, fmt.Errorf("ragindex: %s is not a directory", l.rootDir)
	}

	// 현재 meta 에 등록된 slug 수집 (고아 정리용)
	existing, err := l.wiki.ListMeta()
	if err != nil {
		return 0, fmt.Errorf("list existing meta: %w", err)
	}
	seen := make(map[string]bool, len(existing))

	count := 0
	walkErr := filepath.WalkDir(l.rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Printf("[ragindex] walk error at %s: %v", path, err)
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(path), ".md") {
			return nil
		}
		slug, title, body, meta, loadErr := loadFile(l.rootDir, path)
		if loadErr != nil {
			log.Printf("[ragindex] load %s: %v", path, loadErr)
			return nil
		}
		seen[slug] = true
		if err := l.wiki.UpsertDoc(slug, title, body); err != nil {
			log.Printf("[ragindex] upsert doc %s: %v", slug, err)
			return nil
		}
		if err := l.wiki.UpsertMeta(meta); err != nil {
			log.Printf("[ragindex] upsert meta %s: %v", slug, err)
			return nil
		}
		count++
		return nil
	})
	if walkErr != nil {
		return count, walkErr
	}

	// 파일에서 삭제된 meta 는 함께 제거
	for _, m := range existing {
		if !seen[m.Slug] {
			if err := l.wiki.DeleteDoc(m.Slug); err != nil {
				log.Printf("[ragindex] delete orphan doc %s: %v", m.Slug, err)
			}
			if err := l.wiki.DeleteMeta(m.Slug); err != nil {
				log.Printf("[ragindex] delete orphan meta %s: %v", m.Slug, err)
			}
		}
	}

	return count, nil
}

// loadFile 는 하나의 .md 파일을 읽어 frontmatter 와 본문을 분리.
// slug 는 root 디렉토리 기준 상대경로(확장자 제거). OS 구분자는 `/` 로 정규화.
func loadFile(root, path string) (slug, title, body string, meta *chat.WikiDocMeta, err error) {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return "", "", "", nil, err
	}
	rel = filepath.ToSlash(rel)
	slug = strings.TrimSuffix(rel, filepath.Ext(rel))

	f, err := os.Open(path)
	if err != nil {
		return slug, "", "", nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024) // allow long lines (base64 images 등)

	var (
		inFront     bool
		frontMeta   = map[string]string{}
		bodyBuilder strings.Builder
		firstLine   = true
	)
	for scanner.Scan() {
		line := scanner.Text()
		if firstLine {
			firstLine = false
			if strings.TrimSpace(line) == "---" {
				inFront = true
				continue
			}
		}
		if inFront {
			if strings.TrimSpace(line) == "---" {
				inFront = false
				continue
			}
			if idx := strings.Index(line, ":"); idx > 0 {
				k := strings.TrimSpace(line[:idx])
				v := strings.TrimSpace(strings.Trim(line[idx+1:], " \t"))
				// strip surrounding quotes
				v = strings.Trim(v, `"`)
				v = strings.Trim(v, `'`)
				frontMeta[k] = v
			}
			continue
		}
		bodyBuilder.WriteString(line)
		bodyBuilder.WriteByte('\n')
	}
	if err := scanner.Err(); err != nil {
		return slug, "", "", nil, err
	}

	body = bodyBuilder.String()

	// title 우선순위: frontmatter.title > 첫 # 헤더 > slug
	if t, ok := frontMeta["title"]; ok && t != "" {
		title = t
	} else {
		title = firstH1(body)
		if title == "" {
			title = slug
		}
	}

	meta = &chat.WikiDocMeta{
		Slug:         slug,
		Path:         rel,
		Title:        title,
		NotionPageID: frontMeta["notion_page_id"],
	}
	if s := frontMeta["synced_at"]; s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			meta.SyncedAt = t
		}
	}
	return slug, title, body, meta, nil
}

func firstH1(body string) string {
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, "# "))
		}
	}
	return ""
}
