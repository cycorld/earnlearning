---
title: "브랜드 디자인과 AI 이미지 생성: 서비스에 정체성 입히기"
date: "2026-03-15"
tags: ["브랜드", "디자인", "AI", "Gemini", "PWA", "파비콘"]
---

## 무엇을 했나요?

서비스의 시각적 정체성을 만들었습니다:

- **AI로 로고/아이콘 생성**: Google Gemini 3 Pro Image API로 8개 후보 생성 후 최종 선택
- **브랜드 디자인 가이드**: Gemini 2.5 Pro로 색상/타이포그래피/UI 원칙 문서 생성
- **PWA 아이콘 세트**: 512px, 192px, 180px(apple-touch), 64px, 32px 파비콘 생성
- **테마 색상 변경**: 기존 보라색(#6d28d9)에서 브랜드 틸(#005F69)로

## AI를 활용한 디자인 워크플로우

### 1단계: 후보 생성 (Gemini 3 Pro Image)

프롬프트 엔지니어링으로 다양한 컨셉의 아이콘을 생성합니다:

```bash
# Gemini API로 이미지 생성
curl -s "https://generativelanguage.googleapis.com/v1beta/models/gemini-3-pro-image-preview:generateContent?key=${GEMINI_API_KEY}" \
  -H "Content-Type: application/json" \
  -d '{
    "contents": [{
      "parts": [{"text": "Generate an image: A modern minimalist app icon..."}]
    }],
    "generationConfig": {
      "responseModalities": ["TEXT", "IMAGE"]
    }
  }'
```

핵심 포인트:
- **구체적인 프롬프트**: 스타일(flat, minimal), 색상(hex 코드), 크기(512x512), 형태(rounded corners) 명시
- **병렬 생성**: 4개씩 동시에 생성해서 비교
- **반복 개선**: 첫 라운드 결과를 보고 프롬프트를 수정해서 재생성

### 2단계: 브랜드 가이드 생성 (Gemini 2.5 Pro)

텍스트 모델에게 디자인 전문가 역할을 부여:

```
"You are a senior brand designer. Create a comprehensive brand
and design guide for EarnLearning - a gamified startup education
LMS platform for university students..."
```

AI가 생성한 가이드에는:
- **색상 팔레트**: Primary(Deep Dive Teal #005F69), Accent(Momentum Coral #FF6B6B)
- **타이포그래피**: Plus Jakarta Sans(제목), Noto Sans KR(본문), JetBrains Mono(코드)
- **60-30-10 규칙**: 중립색 60%, 브랜드색 30%, 액센트 10%

### 3단계: 에셋 생성

하나의 512px 원본에서 모든 크기를 생성합니다:

```bash
# macOS sips 명령어로 리사이징
sips -z 192 192 icon-512.png --out pwa-192x192.png
sips -z 180 180 icon-512.png --out apple-touch-icon.png
sips -z 32 32 icon-512.png --out favicon.png
```

## PWA 아이콘이 필요한 이유

PWA(Progressive Web App)는 다양한 환경에서 아이콘이 필요합니다:

```
512x512  → Android 스플래시 화면, Play Store 등록
192x192  → Android 홈 화면 아이콘
180x180  → iOS(Apple Touch Icon) 홈 화면
64x64    → 브라우저 탭 (고해상도)
32x32    → 브라우저 탭 (표준)
```

### HTML에서의 설정

```html
<!-- 파비콘 (브라우저 탭) -->
<link rel="icon" type="image/png" sizes="32x32" href="/favicon.png" />
<link rel="icon" type="image/png" sizes="64x64" href="/favicon-64.png" />

<!-- iOS 홈 화면 아이콘 -->
<link rel="apple-touch-icon" href="/apple-touch-icon.png" />

<!-- 테마 색상 (모바일 브라우저 상단바) -->
<meta name="theme-color" content="#005F69" />
```

### PWA Manifest에서의 설정

```typescript
// vite.config.ts
VitePWA({
  manifest: {
    theme_color: '#005F69',
    background_color: '#F8F9FA',
    icons: [
      { src: '/pwa-192x192.png', sizes: '192x192', type: 'image/png' },
      { src: '/pwa-512x512.png', sizes: '512x512', type: 'image/png' },
    ],
  }
})
```

## Gemini API 이미지 생성 팁

### 모델 선택
```
gemini-2.5-flash-image  → 빠르고 저렴, 간단한 이미지
gemini-3-pro-image-preview → 고퀄리티, 복잡한 디자인
```

### 프롬프트 작성법
```
❌ "앱 아이콘 만들어줘"
✅ "Generate an image: A modern minimalist app icon.
    Deep Teal (#005F69) gradient background.
    White stylized E and L combined mark with upward arrow.
    Rounded corners like iOS app icon.
    No text. 512x512."
```

좋은 프롬프트의 요소:
1. **스타일 지정**: modern, minimalist, flat, geometric
2. **색상 명시**: hex 코드로 정확하게
3. **크기/형태**: 512x512, rounded corners
4. **네거티브 프롬프트**: "No text", "No gradients"
5. **레퍼런스**: "like iOS app icon", "Apple-style simplicity"

## 배운 점

### 1. AI는 디자인 도구다
디자이너를 대체하는 것이 아니라, 빠르게 후보를 생성하고 반복하는 도구입니다. 8개 후보를 5분 만에 생성하고, 마음에 드는 방향을 골라 색상만 바꿔서 재생성할 수 있습니다.

### 2. 브랜드 가이드는 일관성의 기초
색상, 폰트, 스타일을 문서화하면 팀원이 늘어나도 일관된 디자인을 유지할 수 있습니다. "이 버튼 무슨 색이지?" 대신 가이드를 참조합니다.

### 3. 하나의 원본에서 모든 크기를
512px 고해상도 원본 하나를 만들고, `sips`(macOS)나 `sharp`(Node.js)로 나머지를 자동 생성합니다. 각 크기를 따로 만들면 미세한 차이가 생깁니다.

---

## GitHub 참고 링크
- [PR #5: 브랜드 아이콘/파비콘/PWA 에셋 교체](https://github.com/cycorld/earnlearning/pull/5)
- [PR #6: 아이콘 배경 투명 처리](https://github.com/cycorld/earnlearning/pull/6)
