# EarnLearning Task Board

## 구조
```
tasks/
├── backlog/       # 아이디어, 나중에 할 것
├── todo/          # 다음에 할 작업
├── in-progress/   # 진행 중
├── done/          # 완료
└── README.md
```

## 티켓 파일 형식
`NNN-slug.md` (예: `001-email-system.md`)

## 티켓 프론트매터
```yaml
---
id: 001
title: 작업 제목
priority: high | medium | low
type: feat | fix | chore | content
created: 2026-03-18
updated: 2026-03-18
---
```

## 사용법
- `/task list` — 전체 보드 현황
- `/task add [제목]` — 새 티켓 생성 (backlog)
- `/task move [id] [상태]` — 티켓 이동
- `/task show [id]` — 티켓 상세 보기
