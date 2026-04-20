# 105. 챗봇 input/전송 버튼 폴리시 + iOS auto-zoom 방지

**날짜**: 2026-04-20
**태그**: 챗봇, UI, 모바일, 접근성

## 배경
스크린샷 점검 결과 챗봇 입력 영역 약점:
- placeholder "질문을 입력하세요… (Enter 전송, Shift+Enter 줄바꿈)" 가 모바일에서 잘림
- rows=2 + 버튼 옆 배치 → 입력 시작 전 빈 공간 낭비, 버튼이 떠 있음
- `text-sm` (14px) → iOS 에서 focus 시 자동 zoom 발생
- `pb` 가 일정해서 iPhone 홈 indicator 영역과 시각적 충돌
- 활성/비활성 색상 대비 미묘 — disabled 인지 알기 어려움

## 수정 (`ChatDock.tsx`)
| 변경 | 의도 |
|---|---|
| `text-sm` → `text-base` (16px) | iOS focus auto-zoom 방지 (viewport meta 손대지 않음 — WCAG 1.4.4 호환) |
| placeholder 짧게 + `Enter 전송 · Shift+Enter 줄바꿈` 은 desktop 에서만 작은 글씨로 | 모바일 잘림 해결 |
| `rows={1}` + onChange 에서 auto-grow (max 5 lines) | 빈 공간 최소화, 길게 쓰면 자연스럽게 확장 |
| 전송 버튼을 textarea 내부 우하단 absolute (`pr-12` 로 공간 확보) | 당근/토스 패턴, 시각 통합 |
| 활성: `bg-primary text-primary-foreground` / disabled: `bg-muted text-muted-foreground` | 대비 명확화 |
| 컨테이너 padding: `pb-[max(0.5rem,env(safe-area-inset-bottom))]` | iPhone 홈 indicator 영역 안전 |
| `aria-label="전송"` 추가 | 스크린리더 접근성 |

## 화면 확대 정책 (의식적 미포함)
사용자 질문 "화면 확대 막는게 좋을지" 에 대한 결론:
- **pinch-to-zoom 차단 ❌** — WCAG 1.4.4 + iOS HIG 위반. 시력 약한 학생 접근성 핵심 기능. viewport meta 의 `maximum-scale=1, user-scalable=no` 같은 옵션은 절대 추가 안 함.
- **iOS input focus auto-zoom 차단 ✅** — 별개 동작. textarea `font-size: 16px+` 로 자동 해결됨 (이번 PR 에 포함). 다른 input 들은 추후 필요 시 같은 방식.

## 검증
- vitest 136 pass
- Stage 배포 후 모바일 화면에서 챗봇 입력/전송 동작 + iOS focus 시 zoom 안 되는지 확인 필요
