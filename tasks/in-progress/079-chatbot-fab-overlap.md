---
id: 079
title: 챗봇 FAB 가 모바일 BottomNav 에 가려지는 이슈 수정
priority: high
type: fix
branch: fix/chatbot-fab-overlap
created: 2026-04-19
---

## 증상
모바일에서 우하단 챗봇 말풍선(FAB) 이 하단 메뉴에 일부 가려져 클릭이 어려움.

## 원인
- BottomNav: `fixed bottom-0`, 높이 `h-16`(64px), `z-50`
- FAB: `fixed bottom-20`(80px), `z-40` — 80-64=16px 만 떠 있음
- iOS safe-area-inset-bottom (homebar ~34px) 까지 고려하면 사실상 가려짐
- z-40 < z-50 이라 겹치면 BottomNav 가 위로 올라옴

## 수정
- 모바일 `bottom-[calc(5rem+env(safe-area-inset-bottom))]` (~80px + safe-area)
- 바닥 여유 더 주기 위해 `bottom-24`(96px) 로 베이스 상향
- z-index 를 `z-50` 으로 상향해 (BottomNav 와 동일) 시각적으로 항상 위에
- 데스크탑은 기존 `sm:bottom-6` 유지
