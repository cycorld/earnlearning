# 회사 서비스 목록과 안전한 URL 검증, Rybbit 연동 버튼

## 무엇을 했나요?

회사 상세 페이지에 "서비스" 섹션을 새로 만들었습니다. 회사 대표는 자신의 서비스를 여러 개 등록할 수 있고, 서비스마다 HTTPS 주소를 서버가 직접 검증한 뒤에만 분석 도구(Rybbit) 연동 버튼이 활성화됩니다. 기존의 쉼표로 구분하던 `service_url` 필드는 그대로 두고, 완전히 새로운 테이블(`company_services`)을 추가하는 방식으로 만들었습니다.

## 왜 필요한가요?

부트캠프 학생들이 자기 서비스의 방문자 통계를 보려면 각 서비스가 분석 도구에 "사이트"로 등록되어야 합니다. 그런데 아무 주소나 등록하게 두면 두 가지 문제가 생깁니다.

1. **잘못된 주소**: 오타가 있거나 접속되지 않는 주소를 연동하면 통계가 수집되지 않습니다.
2. **보안 문제 (SSRF)**: 서버가 사용자가 준 주소로 접속해 보는 기능은, 악의적인 사용자가 `http://192.168.x.x` 같은 내부 주소를 넣어 서버 내부망을 훔쳐보는 데 악용될 수 있습니다. 이런 공격을 SSRF(Server-Side Request Forgery, 서버 측 요청 위조)라고 합니다.

그래서 "등록 → 서버 검증 → 연동" 3단계로 나누고, 검증을 통과한 주소만 연동할 수 있게 했습니다.

## 어떻게 만들었나요?

**URL 정규화**: `https://App.Example.com:443/`과 `https://app.example.com`은 사실 같은 주소입니다. 대소문자, 기본 포트(443), 끝의 슬래시를 정리한 "정규형"으로 비교해 같은 회사에 같은 주소가 두 번 등록되는 것을 막았습니다. 아이디와 비밀번호가 들어간 주소(`https://user:pw@...`)나 `#조각`이 붙은 주소는 거절합니다.

**SSRF 방어**: 주소의 도메인을 DNS로 풀어서 나온 모든 IP가 공인 IP인지 확인합니다. 사설 대역(10.x, 192.168.x 등), 루프백(127.x), 링크로컬(169.254.x) 같은 내부 대역이 하나라도 나오면 거절합니다. 실제 접속 순간에도 다시 IP를 확인해서(DNS 리바인딩 방어), 검사할 때는 공인 IP였다가 접속할 때 내부 IP로 바뀌는 속임수도 막습니다. 리다이렉트는 3번까지만 따라가고, 매번 같은 검사를 다시 합니다.

**시간 제한과 상태 저장**: 접속 확인은 5초 안에 끝나야 하고, 최종 응답이 2xx(성공)일 때만 "검증됨"이 됩니다. 검증 결과와 시각은 서버가 DB에 기록하며, 클라이언트가 "검증됐다"고 주장해도 믿지 않습니다. 검증 후 24시간이 지나면 다시 검증해야 연동할 수 있습니다.

**연동 버튼과 드리프트**: 연동은 회사 대표만, 그것도 "승인된 학생" 계정이 자기 강의실 회사에 대해서만 할 수 있습니다(관리자 계정은 캠프 자격이 없습니다 — 명시 정책). 이미 연동된 서비스에 다시 눌러도 중복 생성되지 않고(멱등성), 외부 API가 실패하면 아무 상태도 바꾸지 않습니다(fail-closed). URL을 수정하면 이전 검증이 무효가 되고, 연동 상태는 "재연동 필요"로 바뀝니다.

**Rybbit 와 서명으로 대화하기 (HMAC 프로비저닝 계약)**: 처음 구현은 "관리자 토큰을 붙인 사이트 생성 API"를 가정했지만, 실제 Rybbit 쪽에는 EarnLearning 전용 엔드포인트 `POST /api/earnlearning/provision`이 만들어졌습니다. 인증은 토큰이 아니라 **서명**입니다. 양쪽이 같은 비밀 문자열(32자 이상)을 나눠 갖고, 보낼 때마다 `타임스탬프 + "." + 본문 바이트`를 HMAC-SHA256으로 서명해 헤더로 보냅니다. 서버는 받은 원시 바이트로 서명을 다시 계산해 비교하므로, 본문을 다시 직렬화하면(키 순서·공백이 바뀌면) 서명이 깨집니다. 그래서 클라이언트는 본문을 **한 번만** 만들어 그 바이트를 서명과 전송에 함께 씁니다. 요청 한 번에 회사(조직)·사이트·사용자·접근 권한이 모두 등록되고, 사이트 키는 `service:<서비스ID>`로 고정해 URL이 바뀌어도 같은 사이트를 가리킵니다. 프로비저닝의 `user.id`는 OAuth userinfo의 `sub`와 정확히 같은 문자열이어서, 나중에 그 학생이 SSO로 로그인하면 같은 계정으로 연결됩니다.

**권한은 "재조정"된다 — 그래서 항상 전체 집합을 보낸다**: Rybbit은 요청에 담긴 `grants.siteKeys`를 "추가"가 아니라 "이 목록이 전부"로 해석합니다(재조정, reconcile). 만약 두 번째 서비스를 연동하면서 그 서비스 키 하나만 보내면, 첫 번째 서비스에 대한 접근이 **회수**됩니다. 그래서 연동 버튼을 누를 때마다 그 회사에서 이미 연동된 모든 서비스 키 + 방금 누른 서비스 키를 전부 모아 보냅니다. 다른 회사의 키는 절대 섞이지 않으며(회사 범위 조회만 사용), 응답의 `grantedSiteIds`에 방금 만든 사이트가 포함됐는지까지 확인한 뒤에야 DB에 기록합니다.

전체를 TDD로 진행했습니다. 백엔드 통합 테스트(다중 서비스, 중복 URL, 사설 IP, 리다이렉트 SSRF, 타임아웃, 오래된 검증, 소유자 아님, 멱등성, 드리프트)와 프론트엔드 컴포넌트 테스트(서비스별 버튼 활성화, 배지 상태, 소유자/비소유자 화면)를 먼저 작성하고 구현했습니다.

## 사용한 프롬프트

> CompanyDetailPage must support a first-class list of registered services, each service has its own normalized HTTPS URL and its own Rybbit one-click connect button. [...] Each registered URL must be validated before connect: syntax and canonical HTTPS origin, [...] reject localhost/private/link-local/reserved IPs and DNS resolutions (SSRF-safe), then perform a bounded server-side reachability check with strict timeout, redirect limit and revalidation of every redirect target. [...] Connect is idempotent per service and fail-closed.

> Implement and verify the exact EarnLearning↔Rybbit contract. [...] Replace EarnLearning fake Bearer /api/admin/sites provisioner with HMAC POST /api/earnlearning/provision: sign timestamp.rawBody, no Origin, strict response. [...] IMPORTANT: Rybbit reconciles grants, so each click must send the owner full cumulative set of all currently connected valid services for that same company INCLUDING the clicked service, never just one key. [...] Provisioning user.id must exactly equal OAuth userinfo sub.

## 배운 점

사용자가 준 주소로 서버가 대신 접속하는 기능은 편리하지만, 검증 없이 만들면 서버 내부망으로 들어가는 문이 됩니다. "주소 형식 검사 → IP 대역 검사 → 접속 순간 재검사 → 리다이렉트마다 반복"처럼 여러 겹의 방어를 쌓아야 하고, 외부 연동은 "성공을 확인하기 전에는 아무것도 기록하지 않는다(fail-closed)"는 원칙이 상태 꼬임을 막아 줍니다.

외부 시스템과의 계약에서는 두 가지를 더 배웠습니다. 첫째, 서명 기반 API는 "무엇을 서명하는가"가 전부입니다 — 파싱된 객체가 아니라 전송한 바이트를 서명해야 하고, 그래서 본문 생성과 서명은 한 곳에서 같은 바이트로 해야 합니다. 둘째, 상대가 권한을 "재조정" 방식으로 다루면 부분 정보만 보내는 것이 곧 삭제 요청이 됩니다. 이런 의미론은 문서에 한 줄로 적혀 있어도 놓치면 실제 사용자 권한이 사라지는 사고가 되므로, "두 번째 연동 후에도 첫 연동 접근이 살아 있는지"를 테스트로 박아 두었습니다.
