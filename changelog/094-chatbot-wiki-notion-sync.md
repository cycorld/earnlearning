# 094. 챗봇 위키 — Notion 자동 동기화 (#082)

**날짜**: 2026-04-19
**태그**: 챗봇, 위키, Notion, 통합

## 배경
13개 위키 문서는 노션에서 수동으로 가져와 .md 로 만들어진 상태. 노션이 업데이트
되어도 자동으로 따라오지 않음.

## 결정 (사용자 선택)
- **A1** 마크다운 표 변환 (정렬/cell merge 등 일부 손실 허용)
- **B2** Docker volume mount 로 .md 영구 저장 (재배포 후에도 동기화 결과 유지)
- **C1** 어드민 버튼 (수동 트리거)

## 추가
### Backend
- `internal/infrastructure/notion/client.go` 신규
  - Notion API (`/v1/blocks/:id/children` paginated, `/v1/pages/:id`)
  - markdown 변환: heading 1–3, paragraph, bullet/numbered list (재귀 nesting),
    to_do, toggle, quote, callout (이모지 보존), code (언어 포함), image,
    bookmark, divider, child_page placeholder, **table** (column header 자동
    감지)
  - rich text annotation: bold / italic / strike / inline code / link 보존
- `chat_usecase.go`:
  - `NotionFetcher` 인터페이스 + `SetNotion()` 주입
  - `AdminSyncNotionOne(slug)` — 단일 동기화
  - `AdminSyncNotionAll()` — 전체 일괄 (개별 실패해도 나머지 계속, 결과 리스트 반환)
- 라우트:
  - `POST /admin/chat/wiki/:slug/notion-sync`
  - `POST /admin/chat/wiki/notion-sync-all`
- `config.NotionToken` (env `NOTION_INTEGRATION_TOKEN`) — 비어 있으면 동기화 기능 비활성

### B2: Docker volume 으로 wiki 영구화
- `seedWikiDirIfEmpty(dst, src)` — 컨테이너 부팅 시 LLM_WIKI_DIR (volume) 가
  비어있으면 image 의 `./docs/llm-wiki` 에서 시드. 한 번만.
- `deploy/docker-compose.{blue,green,stage}.yml`:
  - 새 volume `prod_wiki` / `stage_wiki` 마운트 → `/data/wiki`
  - 환경변수 `LLM_WIKI_DIR=/data/wiki`
- 결과:
  - 첫 부팅: 빈 volume → image 에서 .md 복사
  - 다음 부팅: volume 에 데이터 있음 → 그대로 사용
  - 노션 동기화 또는 인라인 에디터 변경: volume 에 영구 저장
  - 재배포해도 사라지지 않음

### Frontend (AdminChatPage)
- 위키 카드 헤더에 "전체 노션 동기화" 버튼 (확인 dialog)
- 각 위키 행에 "동기화" 버튼 (notion_page_id 있을 때만)
- 동기화 중 spinner / disabled 표시

## 사용자 측 사전 작업
1. Notion 통합 만들기: https://www.notion.so/my-integrations → 내부 통합 생성
2. 통합 토큰 (`secret_xxx`) 발급
3. 동기화할 13개 위키 페이지에서 통합 share (페이지 우상단 ⋯ → "Add connections")
4. EC2 prod env 에 `NOTION_INTEGRATION_TOKEN=secret_xxx` 추가:
   ```
   ssh earnlearning
   echo 'NOTION_INTEGRATION_TOKEN=secret_xxx' | sudo tee -a /home/ubuntu/lms/deploy/.env.prod
   ```
5. 다음 배포 시 자동 적용 (또는 즉시 적용은 active slot restart)

## 트레이드오프
- **표**: alignment / cell merge 일부 손실. 본문 텍스트와 헤더 자동 감지는 OK.
- **콜아웃 / 토글**: 텍스트는 보존하되 시각 효과는 단순화 (콜아웃 = `> 이모지 텍스트`)
- **하위 페이지**: 본문 가져오지 않고 "📄 _하위 페이지: title_" placeholder 표시
- **이미지**: Notion 의 file URL 은 만료될 수 있음 (1시간). 외부 URL 이면 안전.
- **git push 자동화는 미포함** (#082 후속) — volume 에만 영구 저장. .md 를 git
  에 커밋하려면 별도 작업 필요.

## 후속 (#082 follow-up)
- git push 자동화 (deploy key + GitHub API)
- 실시간 웹훅 동기화
- 선택적 차이 표시 (변경된 부분만)
