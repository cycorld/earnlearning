---
slug: 057-investment-post-link-fix
title: 투자 라운드 공지 글 링크 수정 — /investment → /invest/:id
date: 2026-04-17
tags: [fix, 투자, 링크, 회귀테스트, 데이터마이그레이션]
---

# 투자 라운드 공지 글 링크 수정 — `/investment` → `/invest/:id`

## 왜 필요했는가

프로덕션에서 학생들이 "피드에 뜬 투자 공지 글의 '투자하러 가기' 버튼이 안 먹힌다"는 신고를 해줬습니다.

투자 라운드가 오픈되면 서버가 자동으로 `투자라운지` 채널에 공지 글을 올려줍니다. 이 글 끝에는 상세 페이지로 가는 링크가 붙어있죠:

```markdown
👉 [투자하러 가기](/investment)
```

문제는 **프론트엔드 라우트가 `/investment`가 아니라 `/invest/:id`** 라는 것. 아예 존재하지 않는 주소로 걸려있었습니다. 클릭하면 404 페이지가 뜨는 거죠.

이걸 다음 3가지를 동시에 고쳐야 완전한 수복이 됩니다:
1. **소스코드**: 새 라운드 오픈 시 올라가는 공지 글의 링크 경로를 올바르게
2. **회귀 테스트**: 이 실수가 다시는 반복되지 않도록 자동 검증
3. **프로덕션 데이터**: 이미 잘못 박혀 있는 기존 공지 글 10개도 고쳐주기

## 무엇을 했는가

### 1. 소스코드 수정

`backend/internal/application/investment_usecase.go` 의 `CreateRound` 안에서 공지 글 컨텐츠를 조립하는 부분을 고쳤습니다.

Before:
```go
content := fmt.Sprintf("## 📈 투자 라운드 오픈: %s\n\n... 👉 [투자하러 가기](/investment)",
    c.Name, ...)
```

After:
```go
content := fmt.Sprintf("## 📈 투자 라운드 오픈: %s\n\n... 👉 [투자하러 가기](/invest/%d)",
    c.Name, ..., id)  // id = 방금 만들어진 round의 ID
```

`CreateRound` 는 방금 생성한 라운드의 ID를 `id` 변수에 담고 있으니, 이걸 링크에 끼워 넣으면 됩니다. 이제 클릭하면 해당 라운드의 상세 페이지 (`/invest/1`, `/invest/2`, ...)로 바로 이동합니다.

### 2. 회귀 테스트 — 실패 → 통과로 증명

TDD 규칙에 따라, 회귀를 잡는 테스트를 먼저 작성하고 **실제로 실패하는지** 확인한 뒤 고치는 흐름을 지켰습니다.

```go
// #032 회귀: /invest/<round_id> 로 링크가 박혀야 한다
func TestInvestment_OpenRound_AutoPost_LinkPointsToDetail(t *testing.T) {
    // ... 라운드 오픈 후 ...
    var content string
    ts.db.QueryRow(`
        SELECT p.content FROM posts p
        JOIN channels c ON c.id = p.channel_id
        WHERE c.slug = 'invest'
        ORDER BY p.id DESC LIMIT 1`).Scan(&content)

    // 올바른 경로 포함?
    if !strings.Contains(content, fmt.Sprintf("/invest/%d", round.ID)) {
        t.Errorf("...")
    }
    // 옛 깨진 경로 재등장 방지
    if strings.Contains(content, "(/investment)") {
        t.Errorf("...")
    }
}
```

테스트 검증 과정:
1. 수정 전 코드로 실행 → **FAIL** (올바른 경로 없음, 옛 경로 남아있음)
2. 수정 후 코드로 실행 → **PASS**

이렇게 "버그가 살아있으면 테스트가 실패한다"는 걸 직접 눈으로 확인한 뒤에야 회귀 테스트로 인정합니다.

### 3. 프로덕션 데이터 마이그레이션

이미 잘못된 링크로 올라가 있는 공지 글들이 10개 있었습니다. 새 코드를 배포해도 **기존 글은 저절로 고쳐지지 않으니** DB 직접 수정이 필요했죠.

각 공지 글이 어느 라운드에 대한 것인지 매칭해야 했는데, 다행히 **공지 글과 라운드가 거의 같은 순간에 생성**되기 때문에 `created_at` 기준으로 쉽게 매칭됐습니다:

```sql
-- 매칭 예시: 공지 글 87 = Momuk 회사의 라운드 1
SELECT p.id, ir.id
FROM posts p
JOIN investment_rounds ir
  ON ABS(strftime('%s',ir.created_at) - strftime('%s',p.created_at)) < 5
JOIN companies co ON co.id = ir.company_id
WHERE p.content LIKE '%(/investment)%'
  AND p.content LIKE '%' || co.name || '%';
```

매칭 결과 10건(post_id → round_id):
```
87→1, 88→2, 89→3, 90→4, 92→6, 93→7, 95→9, 97→10, 98→11, 99→12
```

백업 후 개별 UPDATE 문으로 안전하게 교체:
```sql
BEGIN;
UPDATE posts SET content = replace(content, '(/investment)', '(/invest/1)')  WHERE id=87;
-- ... 10개 반복 ...
SELECT COUNT(*) FROM posts WHERE content LIKE '%(/investment)%';  -- 0 확인
COMMIT;
```

## 배운 점

### 링크 경로는 "문자열"이 아니라 "약속"이다

프론트엔드가 `/invest/:id`, 백엔드가 `/investment` — 숫자로 치면 이건 그냥 **약속 불일치**입니다. 이걸 막는 방법은 두 가지가 있어요:

1. **타입으로 묶기**: 라우트를 상수로 만들어서 여기저기 복붙되지 않게 (예: `routes.investDetail(id)`)
2. **테스트로 묶기**: 양쪽이 같은 규칙을 따른다는 걸 자동 검증

이번엔 2번(회귀 테스트)으로 방어막을 쳤습니다. 근본적으로는 1번이 더 튼튼하지만, 다음 번에 개선할 포인트로 남겨두죠.

### 프로덕션 데이터 수정 전엔 반드시 백업

```bash
sudo cp earnlearning.db earnlearning.db.bak-$(date +%Y%m%d-%H%M%S)
```

`UPDATE` 한 줄이 잘못 나가면 100명 학생의 데이터가 뒤틀릴 수 있어요. 백업 → 검증 쿼리(SELECT) → UPDATE → 재검증의 4단계를 지키면 99%의 사고는 막힙니다. 나머지 1%는… 아예 트랜잭션으로 감싸서 롤백 가능하게:

```sql
BEGIN;
UPDATE ...;
SELECT ...;   -- 결과 맘에 드나?
COMMIT;       -- 아니면 ROLLBACK;
```

### 시간 근접 매칭은 "같은 이벤트로 생겼다"의 증거

서로 다른 테이블에 있는 두 레코드(공지 글 vs 라운드)가 **같은 순간에 만들어졌다**는 건 "이 둘은 하나의 이벤트"라는 강력한 힌트입니다. 외래키가 없어도 `created_at` 시간 근접성 + 본문 내 회사 이름 매칭으로 정확히 이어붙일 수 있었어요.

데이터베이스 설계 관점에서는 **앞으로 이런 auto-post에는 `reference_id` 컬럼을 두어 라운드 ID를 직접 저장**하는 게 정답입니다. 이번엔 타임스탬프 매칭으로 해결했지만, 같은 패턴(공지/알림/이벤트 ↔ 원본 객체)이 반복되면 언젠가 꼭 뒷탈이 나죠.

## 사용한 프롬프트

> 프러덕션에서 투자유치 공지 글들의 링크가 /investment 라서 깨지는거 같아. 소스코드 수정하고 프러덕션 데이터도 고쳐줘.
