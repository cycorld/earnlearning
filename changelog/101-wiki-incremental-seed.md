# 101. wiki seed — 디렉토리 단위 incremental 복사

**날짜**: 2026-04-20
**태그**: 인프라, 챗봇, 위키, 운영

## 배경
#100 (강의 노트 wiki 추가) 작업 중 발견:
기존 `seedWikiDirIfEmpty(dst, src)` 는 **dst 가 완전 비어있을 때만** src 통째로 복사.
한 번 시드 후에는 새 서브디렉토리(`lecture-notes/`) 가 추가돼도 자동 복사 안 됨 →
prod volume 에 `docker cp` 수동 작업 필요했음.

## 변경
`seedWikiDir(dst, src)` 로 변경 — **incremental** 시드:
- src 의 모든 파일을 walk
- dst 에 **없는 파일만** 복사 (`os.Stat(dstPath) != nil` 체크)
- dst 에 이미 있으면 skip — 운영자 / Notion sync 가 수정한 파일은 절대 안 건드림
- 로그: `seeded N new files, skipped M existing`

## 작동 시나리오
| 케이스 | 결과 |
|---|---|
| 첫 부팅, dst 비어있음 | src 의 모든 파일 복사 (기존과 동일) |
| 부팅 후 image 에 새 파일 추가 | **새 파일만 복사 — 신규 동작** |
| 운영자가 수정한 파일 | image 에 같은 이름 있어도 skip |
| Notion sync 가 갱신한 파일 | skip |

## 회귀 안전
빈 dst → 전체 시드라는 기존 의도는 그대로. 운영자 데이터 보호도 유지.

## 호출부
`backend/cmd/server/main.go:187` — `seedWikiDirIfEmpty` → `seedWikiDir` 으로 변경.

## 후속 영향
- 향후 새 wiki 디렉토리 추가 시 `docker cp` 수동 작업 불필요
- 다음 배포 시 prod volume 에 누락된 파일이 있으면 자동 보충
