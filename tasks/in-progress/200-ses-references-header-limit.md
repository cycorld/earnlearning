---
id: 200
title: SES References 헤더 길이 제한
priority: high
type: fix
branch: fix/ses-references-header
created: 2026-08-06
---

SES v2 custom header의 995자 제한을 넘는 긴 References 체인을 최신 Message-ID 우선으로 축약하고 회귀 테스트를 추가한다.
