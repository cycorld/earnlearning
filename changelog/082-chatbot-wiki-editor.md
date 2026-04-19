# 082. 챗봇 위키 — 관리자 인라인 에디터 (MVP)

**날짜**: 2026-04-19
**태그**: 챗봇, 위키, 관리자, 에디터

## 배경
원래 #075 는 "Notion 자동 동기화 + 관리자 편집 UI" 통합 티켓이었음. 하지만
Notion 동기화는 git push 권한 / 보안 / 파일 vs DB persistence 결정이 많아 분리.

이 PR 은 **인라인 에디터 MVP** 만 다룸 — 학생 질문에 잘못 답변하는 위키 조항을
관리자가 즉석에서 수정 → 저장 → 다음 질문부터 반영. (영구화는 다음 단계)

## 추가
### Backend
- `WikiRepository.GetDoc(slug)` — FTS5 가상 테이블에서 title + body 직접 조회
- `ChatUseCase.AdminGetWikiDoc(slug)` — meta + body 반환
- `ChatUseCase.AdminUpdateWikiDoc(slug, title, body, rootDir)` — FTS5 + meta 갱신,
  가능하면 .md 파일도 frontmatter 보존하면서 덮어씀
- `ChatUseCase.SetWikiRootDir(dir)` — main.go 에서 wiki 루트 주입
- 라우트:
  - `GET /admin/chat/wiki/:slug` — body + meta
  - `PUT /admin/chat/wiki/:slug` — body/title 업데이트

### Frontend (AdminChatPage)
- 위키 행에 "편집" 버튼 추가
- `WikiEditorModal` — title input + 마크다운 textarea + 저장 버튼
- 저장 시 안내: "영구 반영은 git 커밋 필요"

## 트레이드오프 (중요)
**프로덕션 컨테이너의 .md 파일은 이미지에 패키징 → 재배포 시 사라짐.**
즉, 에디터 변경은 DB FTS5 에 반영되지만 부팅 시 `ragindex.Sync()` 가 .md 파일로
DB 를 다시 덮어씀. → 다음 재배포 후엔 변경사항 사라짐.

영구화하려면 .md 파일을 git 에 커밋해야 함. 이 부분은 추후 #082 (Notion 동기화 +
git push 자동화) 에서 다룰 예정.

## 후속 (#082)
- Notion API 클라이언트 + 노션 동기화 버튼
- git commit + push 자동화 (deploy key 보안 검토 필요)

## 배운 점
**MVP 의 정의는 가치 vs 위험**. "잘못된 답변을 즉석에서 수정" 만으로도 운영 가치는
크고 위험은 거의 없음 (DB 만 변경, 다음 배포에 자동 reset). Notion 동기화는
가치는 크지만 신중한 설계 필요 → 별도 티켓.
