# 068. 게임화 컴포넌트 4종: StreakBadge / LevelChip / CoinChip / SkillTree

**날짜**: 2026-04-18
**태그**: 프론트엔드, 디자인시스템, 게임화, 컴포넌트, TDD

## 무엇을 했나
이화여대 EarnLearning LMS 의 "게임감"을 페이지 단위가 아니라
**재사용 블록** 단위로 만들었다. 4개의 UI 프리미티브를 `frontend/src/components/gamification/`
아래에 신설했고, 각각에 대한 단위 테스트 27 개를 먼저 작성한 뒤(RED) 구현해
(GREEN) 커밋했다 — 프로젝트 규칙(TDD 필수)대로 진행.

1. **StreakBadge** — 🔥 + 연속 일수 pill. `days` 값에 따라 cold(0) / warm(1~6) / hot(7+) 세 톤.
2. **LevelChip** — Seed → Bronze → Silver → Gold → Diamond 다섯 단계 뱃지.
   각각 이모지 아이콘 + 팔레트 구분 (Seed=Ewha Green, Diamond=Info 푸른 보석 등).
3. **CoinChip** — 💰 + 금액 pill. `size`(sm/md/lg) · `showSign`(손익 부호 자동) 지원.
   Warm Orange 로 gain, Coral 로 loss 표시.
4. **SkillTree** — Khan Academy 식 주차별 해금 경로. 노드(완료 ✓ / available → / 🔒 locked)와
   연결선을 수직으로 배치. available 노드 클릭 시 `onSelect` 콜백.

## 왜 필요했나
#065 에서 컬러·폰트·버튼 같은 **디자인 토큰 레이어**는 Ewha Green × Warm Orange × Khan
감성으로 리셋했다. 하지만 실제 학생 화면(과제 제출, 수익률, 레벨업)에서 쓸
**고수준 컴포넌트**는 여전히 없었다 — 매번 Badge + 아이콘 조합을 페이지마다 재구성해야
했다. 재사용 블록이 있어야 게임화가 특정 페이지의 장식이 아니라 서비스 전반의
언어가 된다.

## 어떻게 만들었나
- **위치**: `frontend/src/components/gamification/` (신규 폴더)
- **스타일 전략**: shadcn `Badge` 컴포넌트의 CVA 패턴을 참고하되, 변형이 1~2축으로 작은 건
  간결한 `Record<Variant, string>` 룩업 테이블로 구현. (덜 마법적이고, 디자인 토큰만 건드리기
  쉬움.)
- **디자인 토큰**: `#FF6C00` / `#FF6B6B` / `#00643E` 같은 16 진 코드는 직접 쓰지 않고
  `var(--highlight)` / `var(--coral)` / `var(--primary)` 같이 `index.css` 에 이미
  선언된 시맨틱 토큰만 참조 — 향후 다크모드가 자동으로 따라옴.
- **3D 프레스 효과**: SkillTree 의 완료·진행 노드에 `shadow-[0_3px_0_0_var(--primary-shadow)]`
  를 적용해서 Duolingo 버튼처럼 아래쪽 그림자로 입체감. `active:translate-y-px` 로 눌림 피드백.
- **쇼케이스 페이지**: `/developer/gamification` 라우트 신설 (`GamificationShowcasePage`).
  SkillTree 는 available 노드를 클릭하면 완료로 바뀌고 다음 노드가 열리는 미니
  인터랙션을 붙여 실제 플로우를 바로 감각할 수 있게 했다.
- **테스트**: `@testing-library/react` + Vitest. jest-dom 매처 대신 기존 테스트 스타일
  (`textContent.toContain`, `getAttribute`) 을 따라 DOM 구조·데이터 속성·분기 로직을 검증.

## 설계 메모
- **`data-*` 속성을 일급 인터페이스로**: 각 컴포넌트는 상태를 `data-variant` /
  `data-level` / `data-tone` / `data-status` 로 DOM 에 노출한다. 테스트가 쉬워질 뿐
  아니라 상위 페이지에서 CSS 레벨로 추가 스타일을 걸기도 좋음.
- **"Klingons(API 연동) 는 지금 안 함"**: 티켓에 언급된 백엔드 스키마/엔드포인트는
  뒤로 미뤘다. 이번 PR 의 범위는 **프론트엔드 프리미티브**만. 실제 서비스 화면에서
  뭘 집계해야 할지는 이 블록들을 페이지에 넣어 보면서 결정하는 편이 낫다.
- **`SkillTree.onSelect`**: HTML 네이티브 `onSelect` 이벤트와 이름이 충돌해서
  `Omit<HTMLAttributes, 'onSelect'>` 로 빼고 커스텀 prop 으로 재정의. 사용자 DX 는
  그대로 유지.

## 사용 프롬프트 요약
> "#67 티켓 진행해줘"
> → 티켓(`tasks/in-progress/067-gamification-components.md`)에 나열된 StreakBadge /
>   LevelChip / SkillTree / CoinChip 4종을 TDD 로 구현 + 개발자 쇼케이스 라우트 추가.

## 배운 점
- **HTML 네이티브 이벤트 이름 주의**: `onClick`, `onSubmit`, `onSelect`, `onChange`
  같은 속성명을 커스텀 의미로 쓰고 싶으면 `HTMLAttributes` 에서 Omit 하거나 이름을
  바꿔야 한다. TypeScript 가 잡아 줘서 런타임에 당황하지 않은 것은 다행.
- **디자인 토큰 재활용 > 새 색 추가**: 다섯 레벨 팔레트를 새로 정의할까 고민했지만,
  결국 Bronze 만 미세 16 진 코드로 따로 두고 나머지는 기존 `success/warning/info/muted`
  토큰으로 해결. 새 토큰이 늘어날수록 다크모드 관리 비용이 기하급수로 늘어남.
- **쇼케이스 페이지의 가치**: 디자인 컴포넌트는 눈으로 봐야 판단이 서는데, 스토리북
  없이도 전용 라우트 하나면 충분. 앞으로 새 UI 블록을 만들 때마다 이 페이지에
  추가 섹션을 얹으면 자연스럽게 **내부 스타일 가이드**가 된다.

## 다음 단계
- 각 프리미티브를 실제 페이지에 적용 (예: 프로필 → LevelChip, 과제 피드 → StreakBadge,
  지갑/거래내역 → CoinChip, 강의 홈 → SkillTree).
- 그 과정에서 요구되는 집계 데이터(연속 접속일, 누적 수익 레벨 등)를 백엔드 스키마로
  승격 — 별도 티켓으로 분리.
