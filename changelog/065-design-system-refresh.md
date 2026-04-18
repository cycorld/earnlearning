# 065. 디자인 시스템 리프레시: 이화 그린 + Warm Orange

**날짜**: 2026-04-18
**태그**: 디자인시스템, CSS변수, 브랜드, 이화여대, 칸아카데미

## 무엇을 했나

우리 LMS의 색·배경·질감을 전면 재정의했어요. 기존엔 "Deep Dive Teal(#005F69)
+ Cloud White(#F8F9FA)" 조합의 점잖은 핀테크 톤이었는데, 이번 리프레시로
"**Ewha Green(#00643E) + Warm Orange(#FF6C00) + Warm Paper(#F8F7F4)**" 조합의
친근하고 활기 있는 교육 플랫폼 톤으로 바꿨습니다.

### 바뀐 것

| 항목 | 전(前) | 후(後) | 의미 |
|------|--------|--------|------|
| **Primary** | `#005F69` Deep Dive Teal | **`#00643E` Ewha Green** | 이화여대 공식 CI (Pantone 336C, 1908년 지정) |
| **Background** | `#F8F9FA` Cloud White | **`#F8F7F4` Warm Paper** | 종이·양피지 질감, 눈 피로 감소 |
| **Highlight (신규)** | 없음 | **`#FF6C00` Warm Orange** | 단일 브랜드 악센트 (코인·액션·게임성) |
| **Coral** `#FF6B6B` | 일반 악센트 | 손실·마이너스 수익률 전용 | 역할 분리 |
| **Text** | `#212529` Graphite | `#2C2A24` Warm Charcoal | 배경과 결을 맞춘 따뜻한 회색 |
| **Chart 팔레트** | Teal/Coral/Green/Yellow/Teal | Green/Orange/PearLeaf/Gold/Coral | 브랜드 연속성 |

## 왜 필요했나

프로덕션 LMS가 돌아가고 있지만, 디자인이 "기본 shadcn + 중립 청록"에 머물러
있어서 "우리 서비스만의 색"이 없었어요. 게임화 창업 교육이라는 정체성에
비해 시각이 너무 점잖았죠.

이번 작업은:
1. **국내외 LMS 리서치** (Canvas, Moodle, Coursera, Duolingo, Khan, 인프런,
   클래스팅, 엘리스 등)로 공통 분모와 차별화 축을 파악하고
2. **우리만의 색을 대화로 좁힌** 결과입니다.

세 가지 원칙:
- **뿌리는 이화**: 시작이 이대였으니 이화 그린을 시드로 삼되
- **확장은 전국**: 이대에만 갇히지 않는 보편적 게임 톤
- **레퍼런스는 Khan Academy**: 친근하고 자연스러운 학습 공간 질감

## 어떻게 만들었나

### 1) CSS 변수 전면 재정의

`frontend/src/index.css`의 `:root`와 `.dark` 블록에서 **light/dark 모드 모두**
변수 값을 교체했어요. CSS 변수를 쓰고 있어서 **한 파일만 수정해도 전체 앱에
자동 반영**됩니다. 개별 컴포넌트를 건드릴 필요가 없죠.

```css
:root {
  --background: #F8F7F4;  /* Warm Paper */
  --foreground: #2C2A24;  /* Warm Charcoal */
  --primary:    #00643E;  /* Ewha Green */
  --highlight:  #FF6C00;  /* Warm Orange (신규) */
  --coral:      #FF6B6B;  /* 손실 전용 */
  /* ... */
}
```

다크 모드에선 Ewha Green을 그대로 쓰면 너무 어둡기 때문에 밝은 변형
`#4CAF7F`로, Warm Orange도 소프트한 `#FF8A42`로 조정했습니다.

### 2) `@theme inline`에 highlight 등록

Tailwind v4의 `@theme inline` 블록에 `--color-highlight`와
`--color-highlight-foreground`를 추가했어요. 이제 `bg-highlight`,
`text-highlight` 같은 Tailwind 클래스가 자동 생성됩니다.

### 3) PWA/모바일 테마 컬러 동기화

`index.html`의 `<meta name="theme-color">`와 `vite.config.ts`의
PWA manifest `theme_color`, `background_color`도 모두 새 팔레트로 교체.
홈 화면에 설치된 PWA의 스플래시·상태바 색이 이화 그린으로 바뀝니다.

## 사용한 프롬프트

```
우리 프로젝트 디자인 개선 작업 하자. 일단 국내외 유명한 LMS 서비스 디자인에
대해 리서치 후에 공통 분모를 뽑아줘. 그리고 우리만의 색을 나와 대화하면서
찾고 입히자.
```

## 배운 점

### CSS 변수 기반 디자인 시스템의 힘
shadcn/ui 전제 위에 **CSS 변수**로 브랜드 토큰을 정의해두면, 팔레트 개편이
파일 하나 수정으로 끝납니다. 만약 Tailwind arbitrary value (`bg-[#005F69]`)
를 여기저기 박아뒀다면, 이번 작업은 수십 개 컴포넌트를 일일이 고쳐야
했을 거예요.

### "악센트(accent)"의 두 가지 의미
shadcn의 `--accent` 변수는 실제로는 **hover 상태용 중립색**이라, 디자인
용어의 "브랜드 악센트"와 다릅니다. 그래서 Warm Orange는 별도
`--highlight` 변수로 추가했어요. 명명이 겹치지만 역할은 완전히 달라요.

### 공식 브랜드 컬러는 반드시 1차 출처에서 확인
"이화 그린"이라 말하긴 쉬워도, 공식 값은 `#00643E` (CMYK C100/M0/Y80/K50,
Pantone 336C)이에요. 웹 검색으로 이화여대 공식 UI/SI 페이지를 확인한 뒤
적용했습니다. 브랜드 아이덴티티 작업에선 1차 출처 확인이 필수.

### 다음 단계
- [ ] 버튼에 3D press 그림자 (Duolingo식)
- [ ] 카드 모서리 키우고 외곽선 두텁게 (Khan식)
- [ ] 스킬트리·스트릭·배지 계층 등 게임화 컴포넌트
- [ ] 로고/favicon 재디자인 (이화 그린 반영)
