---
title: "캐시 문제 해결: 새 버전이 안 보일 때"
date: "2026-03-15"
tags: ["캐시", "Service Worker", "PWA", "Nginx", "배포"]
---

## 무엇을 했나요?

새 버전을 배포해도 사용자 브라우저에 이전 버전이 계속 보이는 문제를 해결했습니다:

- **index.html 캐시 방지**: Nginx에서 `no-cache` 헤더 설정
- **버전 API 캐시 방지**: `no-store` 헤더 + fetch `cache: 'no-store'`
- **강제 새로고침 개선**: SW 캐시 클리어 + SW 해제 + 캐시 버스트 URL

## 왜 문제가 발생했나요?

### 웹 캐싱의 3가지 레이어

```
사용자 요청 → ① 브라우저 캐시 → ② CDN 캐시 → ③ 서버
                   ↑                    ↑
              여기서 막힘          여기서도 막힘
```

**레이어 1: 브라우저 캐시**

브라우저는 한번 받은 파일을 저장해두고, 같은 URL 요청 시 서버에 묻지 않고 저장된 파일을 사용합니다:

```
첫 방문: GET /index.html → 서버에서 받음 → 브라우저에 저장
재방문: GET /index.html → 브라우저 캐시에서 바로 로드 (서버 요청 안함!)
```

**레이어 2: Service Worker (PWA)**

PWA 앱은 Service Worker가 네트워크 요청을 가로채서 캐시된 파일을 서빙합니다:

```
요청 → Service Worker가 가로챔 → 캐시에 있으면 캐시에서 반환
                                  → 캐시에 없으면 네트워크 요청
```

`window.location.reload()`를 해도 Service Worker가 캐시된 파일을 줍니다!

**레이어 3: Vite의 콘텐츠 해시 (이미 해결된 부분)**

Vite는 빌드 시 파일 내용을 기반으로 해시를 파일명에 넣습니다:

```
코드 변경 전: index-qRwCKAMx.js  (해시: qRwCKAMx)
코드 변경 후: index-Bp7tXz3K.js  (해시: Bp7tXz3K)
→ 파일명이 달라지므로 브라우저가 새 파일로 인식!
```

이것이 Next.js도 사용하는 "핑거프린팅" 기법입니다. **하지만** 이 해시된 파일명은 `index.html` 안에 적혀 있습니다:

```html
<!-- index.html -->
<script src="/assets/index-qRwCKAMx.js"></script>
```

`index.html` 자체가 캐시되면? → 옛날 해시 파일명 → 옛날 JS 로드!

## 어떻게 해결했나요?

### 해결 전략

```
핑거프린팅 (Vite 기본 제공):
  JS, CSS 파일 → 콘텐츠 해시 → 영구 캐시 OK ✅

캐시 방지 (우리가 추가):
  index.html → no-cache → 항상 최신 HTML → 새 해시 JS 참조 ✅
  /api/version → no-store → 항상 서버에서 가져옴 ✅
  sw.js → no-cache → Service Worker 업데이트 감지 ✅
```

### 1. Nginx 캐시 정책

```nginx
server {
    # HTML: 항상 서버에 확인 (캐시하되 매번 revalidate)
    location / {
        try_files $uri $uri/ /index.html;
        add_header Cache-Control "no-cache";
    }

    # JS/CSS/이미지: 영구 캐시 (파일명에 해시가 있으므로 안전)
    location ~* \.(js|css|png|jpg|svg|woff2)$ {
        expires 1y;
        add_header Cache-Control "public, immutable";
    }

    # Service Worker: 절대 캐시하지 않음
    location = /sw.js {
        add_header Cache-Control "no-cache, no-store, must-revalidate";
    }
}
```

`no-cache` vs `no-store`:
```
no-cache: 캐시는 하되, 사용 전에 서버에 "아직 유효해?" 확인
         → index.html에 적합 (변경 안됐으면 304 응답으로 빠르게)

no-store: 캐시 자체를 하지 않음. 매번 서버에서 가져옴
         → 버전 API에 적합 (항상 최신 값 필요)

immutable: 절대 변경되지 않음. 서버에 확인도 안 함
         → 해시된 JS/CSS에 적합 (내용이 바뀌면 파일명이 바뀌니까)
```

### 2. 강제 새로고침 (Service Worker 우회)

```typescript
async function forceRefresh(): Promise<void> {
  // 1. Service Worker가 캐시한 모든 파일 삭제
  if ('caches' in window) {
    const cacheNames = await caches.keys()
    await Promise.all(cacheNames.map(name => caches.delete(name)))
  }

  // 2. Service Worker 해제 (더 이상 요청을 가로채지 않음)
  if ('serviceWorker' in navigator) {
    const registrations = await navigator.serviceWorker.getRegistrations()
    await Promise.all(registrations.map(r => r.unregister()))
  }

  // 3. 캐시 버스트 URL로 이동 (브라우저 캐시도 우회)
  const url = new URL(window.location.href)
  url.searchParams.set('_v', Date.now().toString())
  window.location.replace(url.toString())
  // ?_v=1710468000000 ← 이 파라미터 때문에 브라우저가 새 요청으로 인식
  // SPA이므로 서버는 이 파라미터를 무시하고 index.html을 반환
}
```

왜 `window.location.reload()` 대신 이 방법을 쓰나:
```
reload()의 한계:
1. Service Worker가 가로채서 캐시된 파일 반환 가능
2. 브라우저가 disk cache에서 로드 가능
3. reload(true)는 deprecated (비표준)

forceRefresh()의 장점:
1. SW 캐시 삭제 → SW가 가로챌 파일 없음
2. SW 해제 → 가로채기 자체를 중단
3. URL 변경 → 브라우저가 완전히 새 요청으로 처리
4. 다음 페이지 로드 시 SW가 새로 등록됨 → 최신 파일 캐시
```

## 배운 점

### 1. 캐시는 양날의 검
캐시를 잘 쓰면 성능이 빨라지고, 잘못 쓰면 업데이트가 반영 안 됩니다. 파일 유형별로 다른 캐시 전략이 필요합니다.

### 2. "안 되는 이유"를 레이어별로 분석하자
문제가 있을 때 "캐시 때문이야"로 끝내지 말고, **어느 레이어의 캐시인지** 정확히 파악해야 합니다. 브라우저? CDN? Service Worker? 각각 해결 방법이 다릅니다.

### 3. 핑거프린팅은 업계 표준
Vite, Webpack, Next.js 모두 콘텐츠 해시 기반 캐시 전략을 사용합니다. 원리를 이해하면 어떤 프레임워크에서든 적용할 수 있습니다.
