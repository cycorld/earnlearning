# 🗄️ 학생용 데이터베이스 사용 가이드

EarnLearning이 제공하는 개인 PostgreSQL DB로 여러분의 바이브코딩 프로젝트를
"진짜 서버처럼" 만들 수 있어요. SQLite 파일이 아니라 원격 DB에 연결하는 경험!

## 1. DB 만들기

1. **프로필 페이지** (`https://earnlearning.com/profile`) 에서
2. **"내 데이터베이스"** 섹션의 **[+ 새 DB 만들기]** 버튼 클릭
3. 프로젝트명 입력 (예: `todoapp`, `portfolio`)
4. 생성 완료 → 접속정보가 화면에 표시됨

> ⚠️ **비밀번호는 이때 한 번만 보여요.** 안전한 곳(노션, 메모장)에 복사해두세요.
> 잊어버렸다면 **[🔄 비밀번호 재발급]** 을 누르면 새 비밀번호가 나와요.

## 2. 접속 정보

| 항목 | 값 |
|------|------|
| Host | `db.earnlearning.com` |
| Port | `6432` |
| Database | `{username}_{projname}` (예: `seowon_todoapp`) |
| User | `{username}_{projname}` (DB명과 동일) |
| Password | 생성/재발급 시 표시 |

## 3. 접속 방법

### 🖥 psql (커맨드라인)
```bash
PGPASSWORD='여기에_비밀번호' psql \
  -h db.earnlearning.com \
  -p 6432 \
  -U seowon_todoapp \
  seowon_todoapp
```

### 🖼 DBeaver (GUI)
1. 새 연결 → PostgreSQL
2. Host: `db.earnlearning.com`, Port: `6432`
3. Database: `seowon_todoapp`
4. Username: `seowon_todoapp`
5. Password: 발급받은 비밀번호

### 🟢 Node.js (`pg` 라이브러리)
```bash
npm install pg
```
```javascript
import pg from 'pg';
const client = new pg.Client({
  host: 'db.earnlearning.com',
  port: 6432,
  database: 'seowon_todoapp',
  user: 'seowon_todoapp',
  password: process.env.DB_PASSWORD,
});
await client.connect();

const result = await client.query('SELECT NOW()');
console.log(result.rows);
```

`.env` 파일:
```
DATABASE_URL=postgresql://seowon_todoapp:비밀번호@db.earnlearning.com:6432/seowon_todoapp
```

### 🐍 Python (`psycopg2` 또는 `psycopg`)
```bash
pip install psycopg2-binary
```
```python
import psycopg2
conn = psycopg2.connect(
    host='db.earnlearning.com',
    port=6432,
    dbname='seowon_todoapp',
    user='seowon_todoapp',
    password=os.environ['DB_PASSWORD'],
)
cur = conn.cursor()
cur.execute('SELECT NOW()')
print(cur.fetchone())
```

### 🦀 Supabase / Prisma / Drizzle
연결 문자열 (URL) 형태로 붙일 수 있어요:
```
postgresql://seowon_todoapp:비밀번호@db.earnlearning.com:6432/seowon_todoapp
```

## 4. 첫 테이블 만들어보기

```sql
-- 할 일 테이블
CREATE TABLE todos (
  id SERIAL PRIMARY KEY,
  title TEXT NOT NULL,
  done BOOLEAN DEFAULT false,
  created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 데이터 넣기
INSERT INTO todos (title) VALUES ('DB 연결 성공!');
INSERT INTO todos (title) VALUES ('할 일 앱 만들기');

-- 조회
SELECT * FROM todos ORDER BY created_at DESC;
```

## 5. 주의사항 / 규칙

- **무한루프 쿼리 금지**: 30초 이상 걸리는 쿼리는 자동으로 끊어져요 (`statement_timeout`)
- **커넥션 10개 제한**: 한 DB에서 동시 커넥션 10개까지. 코드에서 반드시 `client.end()` / `conn.close()` 호출!
- **사용자당 최대 5개 DB**: 프로젝트가 많아지면 오래된 것 삭제
- **격리 보장**: 친구의 DB에는 접근 못 해요 (각자 본인 DB만)
- **백업은 본인 책임**: LMS가 자동 백업 안 해요. 중요한 데이터는 `pg_dump` 로 내보내세요:
  ```bash
  PGPASSWORD='...' pg_dump -h db.earnlearning.com -p 6432 -U seowon_todoapp seowon_todoapp > backup.sql
  ```

## 6. 자주 묻는 질문

**Q. 비밀번호 잃어버렸어요!**
→ 프로필 페이지에서 **[🔄 비밀번호 재발급]** 클릭. 새 비밀번호가 나와요.

**Q. DB 용량 제한은?**
→ 명시적인 쿼터는 없지만 서버 디스크(약 20GB)를 모두가 나눠 쓰는 구조니
수 MB 수준(학습용 데이터)으로만 써주세요.

**Q. 배포한 웹앱에서 이 DB를 써도 되나요?**
→ 네! `db.earnlearning.com:6432` 는 외부에서 접속 가능해요.
다만 **비밀번호를 프론트엔드 코드에 넣지 마세요** — 반드시 백엔드(Node.js/Python)에서 접속.

**Q. 친구 DB를 구경할 수 있나요?**
→ 불가능해요. 각 DB는 본인 계정으로만 접근할 수 있도록 격리되어 있어요.

**Q. LMS의 기존 데이터(과제, 공지)에 접근할 수 있나요?**
→ 아니요. 학생 개인 DB는 LMS 내부 DB와 완전히 분리되어 있어요.

## 7. 트러블슈팅

**`connection refused`**
→ 방화벽 이슈. 회사/학교 와이파이가 TCP 6432 포트를 막았을 수 있어요. 다른 네트워크에서 시도.

**`password authentication failed`**
→ 비밀번호가 틀렸거나, 예전 비밀번호. 재발급 받으세요.

**`FATAL: database "..." does not exist`**
→ DB명 오타. 프로필 페이지에서 정확한 DB명 복사.

**`too many connections for role`**
→ 10개 커넥션 한도 초과. 코드에서 커넥션을 닫지 않고 있어요.
   `try { ... } finally { client.end() }` 형태로 감싸세요.
