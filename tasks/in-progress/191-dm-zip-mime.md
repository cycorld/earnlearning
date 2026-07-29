---
id: 191
title: DM ZIP 첨부 MIME 호환
priority: medium
type: fix
branch: fix/191-dm-zip-mime
created: 2026-07-29
---

브라우저가 ZIP을 `application/x-zip-compressed`로 전송해도 기존 보안 검증을 유지하며 첨부할 수 있게 한다.