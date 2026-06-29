---
id: 138
title: 평가지표 매트릭스 ReviewDialog 긴 에세이 스크롤 불가 수정
priority: high
type: fix
branch: fix/138-essay-eval-modal-scroll
created: 2026-06-30
---
관리 > 평가지표 매트릭스에서 마일스톤(에세이/회고) 평가 시, 제출 내용이 길면
ReviewDialog 모달이 뷰포트를 넘어 커지지만 스크롤이 안 됨. 하단 승인/반려 버튼이
화면 밖으로 밀려 접근 불가.

원인: `AdminMilestonesPage.tsx` ReviewDialog 의 Card 에 max-height/overflow 없음.
오버레이가 `items-center` 로 세로 중앙 정렬 → 긴 내용이 위아래로 잘려 버튼 접근 불가.

수정: 모달 높이를 뷰포트로 제한(`max-h-[90vh]`)하고, 헤더/푸터(코멘트+버튼)는 고정,
가운데 제출 내용 영역만 스크롤되도록 flex 컬럼 + overflow-y-auto 구성.
