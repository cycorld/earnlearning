---
id: 125
title: 평가지표 페이지 — 사업계획서 다중파일(비공개) + 회고 에디터 + 성적/자산 percentile
priority: high
type: feat
branch: feat/125-milestones-files-editor-grade
created: 2026-06-12
---

## 설명
/milestones 페이지 기능 업데이트.

## 요구사항
1. **사업계획서(business_plan) 다중 파일 업로드** — 다양한 포맷. 파일은 **업로더 본인 + 관리자만 접근**(비공개). 업로더 본인 내용 확인 가능.
2. **회고 발표(retrospective)** — Textarea → MarkdownEditor(위지윅, 툴바+미리보기). 세로 높이 크게.
3. **성적 평가 표시** — A/B/C/D 그레이드(승인 평가지표 개수 기반). 학생 자산가치 **상위 N%** 표시 (같은 A/B/C/D 그룹 내에서만 판단).
4. 스테이지 브라우저 테스트.
5. (검수 후) 메인 메뉴에 평가지표 메뉴 추가.

## 설계
- **비공개 파일**: `data/private_uploads/` (static 서빙 X). 새 테이블 `milestone_files`. 인증 다운로드 엔드포인트(owner OR admin).
  - `POST /api/milestones/files` (multipart) / `GET /api/milestones/files` (본인 목록) / `GET /api/milestones/files/:id` (다운로드, owner|admin) / `DELETE /api/milestones/files/:id` (본인 삭제)
  - admin 매트릭스(ListAll) business_plan milestone 에 Files 첨부 → admin 다운로드.
- **성적/percentile**: `/milestones/mine`(ListForStudent) 응답에 grade(=group), asset_total, group_size, asset_rank, asset_percentile 추가.
  - 자산 = Cash+Stock+CompanyEquity−Debt (GetAssetBreakdown 공식). 전 학생 (approved_count, total_asset) 1쿼리 → 그룹 분류 → 같은 그룹 내 순위 → 상위 N%.
- **회고 에디터**: 기존 `MarkdownEditor` 재사용, rows 크게.

## 결정 (사용자 확인)
- 회고 에디터: 기존 MarkdownEditor 재사용
- 그레이드 A/B/C/D: 승인된 평가지표 개수 기반 (기존 ClassifyGroup)
- percentile "같은 그룹": A/B/C/D 등급 그룹

## 테스트 (TDD)
- 파일 접근제어: owner 다운로드 OK / 타인 403 / admin OK / 본인 삭제 / 타인 삭제 거부
- grade + asset percentile: /milestones/mine 응답 검증 (같은 그룹 내 순위)
