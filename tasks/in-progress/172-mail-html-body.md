---
id: 172
title: HTML 전용 메일 본문 렌더 (샌드박스 iframe) + 스니펫 폴백
priority: high
type: fix
branch: fix/172-mail-html-body
created: 2026-07-21
---

## 증상
Anthropic 로그인 링크 등 HTML 전용 메일(text/plain 파트 없음)이 수신함에서 본문 빈 칸.

## 원인
body_html 은 저장되지만 화면이 XSS 방지로 body_text 만 렌더.

## 수정
- 상세: body_html 있으면 sandbox iframe 렌더 (allow-scripts 없음 → JS 전면 차단,
  allow-popups 로 링크만 새 탭, allow-same-origin 은 스크립트 차단 상태라 안전 — 높이 측정용)
- 목록 스니펫: body_text 비면 body_html 태그 제거 텍스트로 폴백 (백엔드)
