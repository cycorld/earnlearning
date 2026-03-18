---
title: "브랜드 가이드 기반 UI 리디자인: 색상과 서체로 정체성 완성하기"
date: "2026-03-15"
tags: ["브랜드", "UI", "디자인시스템", "타이포그래피", "CSS"]
---

# 브랜드 가이드 기반 UI 리디자인

## 왜 필요했나?

이전 개발일지(011편)에서 브랜드 가이드를 만들고 로고/아이콘을 생성했지만, 실제 앱의 UI에는 아직 반영되지 않은 상태였습니다. CSS 색상은 보라색(purple) 계열이었고, 서체도 기본 Geist 폰트를 사용하고 있었습니다.

**브랜드 가이드 vs 실제 구현의 불일치:**
- Primary 색상: 가이드 `#005F69` (Teal) vs 실제 `purple(280도)`
- 서체: 가이드 Plus Jakarta Sans + Noto Sans KR vs 실제 Geist Variable
- 배경색: 가이드 Cloud White `#F8F9FA` vs 실제 순백색 `#FFFFFF`

디자인 시스템이 코드에 반영되지 않으면, 가이드 문서는 그냥 종이에 불과합니다.

## 무엇을 했나?

### 1. 색상 팔레트 전면 교체

CSS 커스텀 속성(변수)을 브랜드 가이드의 색상으로 교체했습니다.

```css
/* Before: 보라색 기반 */
--primary: oklch(0.5 0.17 280);  /* purple */
--background: oklch(1 0 0);      /* pure white */

/* After: 브랜드 Teal 기반 */
--primary: #005F69;               /* Deep Dive Teal */
--background: #F8F9FA;            /* Cloud White */
--foreground: #212529;            /* Graphite */
```

**60-30-10 규칙** 적용:
- **60% 중립색**: Cloud White 배경 + Light Grey 보더
- **30% 브랜드색**: Deep Dive Teal 텍스트, 헤더, 링크
- **10% 악센트**: Momentum Coral 알림 배지, CTA 강조

Coral(`#FF6B6B`) 색상을 커스텀 CSS 변수로 추가하여 알림 배지 등에 활용할 수 있게 했습니다.

### 2. 서체 시스템 구축

브랜드 가이드에서 지정한 3가지 서체를 설치하고 역할별로 배치했습니다.

```bash
npm install @fontsource-variable/plus-jakarta-sans \
            @fontsource/noto-sans-kr \
            @fontsource/jetbrains-mono
npm uninstall @fontsource-variable/geist
```

| 용도 | 서체 | 이유 |
|------|------|------|
| 헤딩 (h1~h6) | Plus Jakarta Sans | 모던하고 테크 느낌, 명확한 시각적 위계 |
| 본문 | Noto Sans KR | 한글 최적화, 긴 글 가독성 |
| 코드 | JetBrains Mono | 개발자용 서체, `l`과 `1` 구분 |

CSS `@layer base`에서 요소별 서체를 자동 적용:

```css
h1, h2, h3, h4, h5, h6 {
  font-family: var(--font-heading);  /* Plus Jakarta Sans */
}
code, pre, kbd, samp {
  font-family: var(--font-mono);     /* JetBrains Mono */
}
```

### 3. 다크 모드 색상 보정

Teal은 어두운 색상이라 다크 모드에서는 밝은 변형이 필요합니다:

```css
.dark {
  --primary: #1AA8B5;        /* 밝은 Teal */
  --background: #1A1D21;     /* Graphite 변형 */
  --coral: #FF8A8A;          /* 밝은 Coral */
}
```

### 4. Favicon 색상 통일

SVG 파비콘의 보라색(`#863bff`, `#7e14ff`)을 Teal(`#005F69`)로, 시안(`#47bfff`)을 Coral(`#FF6B6B`)로 교체하여 PWA 아이콘과 통일했습니다.

## 어떻게 가능했나? (CSS 변수의 힘)

이번 작업의 핵심은 **CSS 커스텀 속성(CSS Variables)** 덕분에 색상 변경이 매우 간단했다는 점입니다.

모든 컴포넌트가 `text-primary`, `bg-background` 같은 Tailwind 유틸리티를 사용하고, 이것들이 CSS 변수를 참조하기 때문에, **변수값만 바꾸면 전체 앱이 한번에 바뀝니다**.

```
CSS 변수 (:root)
   ↓
@theme inline (Tailwind에 변수 연결)
   ↓
text-primary, bg-primary 등 유틸리티 클래스
   ↓
모든 컴포넌트에 자동 반영
```

만약 색상을 하드코딩했다면 (`style="color: purple"`) 50개 이상의 파일을 일일이 수정해야 했을 것입니다.

## 배운 점

1. **디자인 토큰의 가치**: CSS 변수로 색상/서체를 추상화하면 브랜드 변경이 한 파일 수정으로 끝남
2. **60-30-10 규칙**: 색상 비율을 지키면 전문적이고 일관된 느낌을 줌
3. **서체는 목적별로**: 헤딩/본문/코드 각각 최적화된 서체를 쓰면 가독성이 크게 향상
4. **다크 모드는 단순 반전이 아님**: 밝기와 채도를 별도로 조정해야 자연스러움
5. **디자인 시스템 = 코드**: 문서만으로는 부족하고, CSS/코드에 반영되어야 진짜 디자인 시스템

## 사용한 프롬프트

```
브랜딩 문서 참고해서 우리 서비스 UI 디자인 개선해줘.
```

AI가 브랜드 가이드(`docs/brand-guide.md`)를 읽고, 현재 CSS와의 차이를 분석한 뒤, 색상/서체/파비콘을 일괄 업데이트했습니다.

---

## GitHub 참고 링크
- [PR #9: 브랜드 가이드 기반 UI 리디자인](https://github.com/cycorld/earnlearning/pull/9)
- [PR #8: 마크다운 GitHub 스타일 적용](https://github.com/cycorld/earnlearning/pull/8)
