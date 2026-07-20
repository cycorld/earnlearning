---
id: 171
title: 수신 메일 발신자 표시를 헤더 From 으로 (SES 봉투주소 노출 수정)
priority: high
type: fix
branch: fix/171-mail-display-from
created: 2026-07-21
---

## 증상
내부 발송 메일이 수신함에서 보낸사람 `010c...@mail.earnlearning.com` (SES 봉투/VERP 주소)로 표시됨.

## 원인
Worker가 스푸핑 불가능한 봉투 발신자(message.from)만 저장. SES 발송 메일의 봉투는 반송 추적용 일회용 주소.

## 수정
- Worker: postal-mime 헤더 From(주소+이름)을 payload에 추가 (`header_from`, `header_from_name`)
- 백엔드: emails에 `header_from`/`header_from_name` 컬럼(ALTER+CREATE), 목록·상세 응답에 표시용 발신자(헤더 우선, 없으면 봉투) 제공. 봉투 주소는 계속 저장(감사·신뢰용).
- 프론트: 표시용 발신자 렌더 (이름 + 주소)
