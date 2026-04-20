# 104. tasks/in-progress → done 일괄 정리 (housekeeping)

**날짜**: 2026-04-20
**태그**: housekeeping, 운영

## 배경
audit 결과 21개 in-progress 티켓 중 20개가 changelog 존재 + PR 머지 완료된 상태에서 `tasks/done/` 으로 이동만 안 된 채 남아있었음. 매 PR 마다 done 이동을 깜박하거나, #103 history rewrite 후 EC2/cycorld 재 clone + 로컬 reset 으로 일부 상태 reset 영향도 있음.

## 이동
20개 ticket → `tasks/done/`:

| ID | 티켓 | 매칭 changelog |
|---|---|---|
| 016 | userdb-cli-orphan-rows | 016-auto-post-notifications, 096-userdb-admin-reconcile |
| 072 | chatbot-sse-streaming | 072-chatbot-ta |
| 074 | chatbot-admin-enhancements | 074-chatbot-service-key |
| 075 | chatbot-wiki-editor | 075-chatbot-context7, 082 |
| 080 | housekeeping-073-078 | 080 |
| 082 | chatbot-wiki-notion-sync | 082, 094 |
| 084 | chatbot-stream-first-hop-text | 084 |
| 085 | chatbot-fetch-failed-mixed-content | 085 |
| 086 | chatbot-wiki-citation-broken | 086 |
| 087 | llm-concurrency-cap | 087 |
| 088 | chatbot-queue-progress | 088 |
| 089 | llm-concurrency-metrics | **092**-llm-stats-endpoint |
| 090 | chatbot-faq-cache | **093**-chatbot-faq-cache |
| 091 | queue-progress-panic | 091 |
| 095 | wiki-slug-decode | 095 |
| 099 | markdown-internal-link | 099 |
| 100 | wiki-lecture-notes | 100 |
| 101 | wiki-incremental-seed | 101 |
| 102 | public-repo-security-audit | 102 |
| 103 | history-rewrite-pii | 103 |

## 추가 정리
- 097-oauth-bounty-followup.md 가 in-progress + done 양쪽에 중복 → in-progress 쪽 삭제

## 검증
in-progress 에는 본 housekeeping ticket (#104) 만 남음.

## 향후 재발 방지
PR 머지 직후 done 이동을 한 commit 으로 묶어 처리하는 습관 / 또는 PR merge hook 자동화 (별도 티켓 권장).
