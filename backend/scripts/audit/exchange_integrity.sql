-- ============================================================================
-- 거래소 체결 정합성 점검 (#143)  —  READ-ONLY. 쓰기 구문 없음.
--
-- 배경: #140 이전의 체결은 "돈은 이동했는데 주식 소유권은 안 넘어간" 반쪽 상태를
--   남겼을 수 있다(매도자 미차감·매수자 주주레코드 0). #142 로 앞으로의 손상은
--   차단했고, 이 스크립트는 *과거 손상분* 을 탐지한다.
--
-- 실행 (prod, 컨테이너 내부 — 반드시 -readonly 로 열어 DB 에 쓰지 않게):
--   ssh earnlearning
--   CONTAINER=$(./deploy.sh status 2>&1 | grep -oP 'earnlearning-(blue|green)' | head -1)-backend-1
--   sudo docker exec -i $CONTAINER sh -c \
--     'apk add sqlite >/dev/null 2>&1; sqlite3 -readonly -box /data/db/earnlearning.db' \
--     < exchange_integrity.sql
--
-- 실행 (stage):
--   같은 방식, CONTAINER=earnlearning-stage-backend-1, DB=/data/earnlearning.db
-- ============================================================================

.print '\n=== Q1. 주식 보존 위반 (SUM(shareholders.shares) != companies.total_shares) ==='
.print '    diff>0 = 주식 증발(어딘가 미배정),  diff<0 = 주식 누수(중복/과배정)'
SELECT c.id                              AS company_id,
       c.name                            AS company,
       c.total_shares                    AS total_shares,
       COALESCE(SUM(s.shares), 0)        AS held_shares,
       c.total_shares - COALESCE(SUM(s.shares), 0) AS diff
FROM companies c
LEFT JOIN shareholders s ON s.company_id = c.id
GROUP BY c.id
HAVING diff != 0
ORDER BY ABS(diff) DESC;

.print '\n=== Q2. 유령 매수 (stock_trades 엔 매수자로 있으나 주주 레코드 전무) ==='
.print '    #140 시그니처(일부): 체결은 기록됐는데 매수자 주주 레코드 자체가 없음.'
.print '    ⚠ 맹점: 매수자가 그 회사에 이미 investment 주주 레코드를 갖고 있으면'
.print '    (UpsertShareholder 가 기존 행에 합산하므로) 미배달이어도 여기 안 잡힘 → Q5 로 보완.'
SELECT t.company_id,
       c.name              AS company,
       t.buyer_id,
       COUNT(*)            AS phantom_trades,
       SUM(t.shares)       AS bought_shares,
       SUM(t.total_amount) AS paid_amount
FROM stock_trades t
JOIN companies c ON c.id = t.company_id
LEFT JOIN shareholders s
       ON s.company_id = t.company_id AND s.user_id = t.buyer_id
WHERE s.user_id IS NULL
GROUP BY t.company_id, t.buyer_id
ORDER BY t.company_id, bought_shares DESC;

.print '\n=== Q3. 거래 발생 회사별 요약 (체결 수 / 거래주식 합 / 현재 주주 보유합) ==='
.print '    수동 대조용: traded_shares 누적과 held_shares 흐름이 어긋나면 의심'
SELECT c.id                       AS company_id,
       c.name                     AS company,
       c.total_shares,
       (SELECT COUNT(*)  FROM stock_trades t WHERE t.company_id = c.id) AS trade_count,
       (SELECT COALESCE(SUM(t.shares),0) FROM stock_trades t WHERE t.company_id = c.id) AS traded_shares,
       (SELECT COALESCE(SUM(s.shares),0) FROM shareholders s WHERE s.company_id = c.id)  AS held_shares
FROM companies c
WHERE EXISTS (SELECT 1 FROM stock_trades t WHERE t.company_id = c.id)
ORDER BY c.id;

.print '\n=== Q4. 매도자 미차감 정황 (매도 체결이 있는데 보유 주식이 과대) ==='
.print '    seller 가 판 만큼(sold) 줄지 않았는지 — net_sold 만큼은 holdings 에서 빠졌어야 함'
SELECT t.company_id,
       c.name        AS company,
       t.seller_id,
       SUM(t.shares) AS sold_via_trades,
       COALESCE(s.shares, 0) AS current_holding
FROM stock_trades t
JOIN companies c ON c.id = t.company_id
LEFT JOIN shareholders s
       ON s.company_id = t.company_id AND s.user_id = t.seller_id
GROUP BY t.company_id, t.seller_id
ORDER BY t.company_id, sold_via_trades DESC;

.print '\n=== Q5. 매수자 지갑 차감 vs 주식 미배달 (가장 신뢰도 높은 검출) ==='
.print '    stock_buy 로 돈은 나갔는데 그 회사에 trade 로 취득한 주식이 0 인 매수자.'
.print '    (Q2 가 놓치는 "이미 investment 보유" 케이스까지 포착)'
SELECT t.buyer_id,
       t.company_id,
       c.name                AS company,
       COUNT(*)              AS trades,
       SUM(t.shares)         AS bought_shares,
       SUM(t.total_amount)   AS paid_amount
FROM stock_trades t
JOIN companies c ON c.id = t.company_id
WHERE NOT EXISTS (
        SELECT 1 FROM shareholders s
        WHERE s.company_id = t.company_id
          AND s.user_id    = t.buyer_id
          AND s.acquisition_type = 'trade'
      )
GROUP BY t.buyer_id, t.company_id
ORDER BY paid_amount DESC;
