-- EarnLearning Database Schema

PRAGMA journal_mode = WAL;
PRAGMA busy_timeout = 5000;
PRAGMA synchronous = NORMAL;
PRAGMA foreign_keys = ON;

-- ============================================================
-- Identity Domain
-- ============================================================
CREATE TABLE IF NOT EXISTS users (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    email       TEXT    NOT NULL UNIQUE,
    password    TEXT    NOT NULL,
    name        TEXT    NOT NULL,
    department  TEXT    NOT NULL,
    student_id  TEXT    NOT NULL,
    role        TEXT    NOT NULL DEFAULT 'student'
                CHECK (role IN ('admin', 'student')),
    status      TEXT    NOT NULL DEFAULT 'pending'
                CHECK (status IN ('pending', 'approved', 'rejected')),
    bio         TEXT    DEFAULT '',
    avatar_url  TEXT    DEFAULT '',
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Classroom Domain
-- ============================================================
CREATE TABLE IF NOT EXISTS classrooms (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    name            TEXT    NOT NULL,
    code            TEXT    NOT NULL UNIQUE,
    created_by      INTEGER NOT NULL REFERENCES users(id),
    initial_capital INTEGER NOT NULL DEFAULT 50000000,
    settings        TEXT    DEFAULT '{}',
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS classroom_members (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    classroom_id INTEGER NOT NULL REFERENCES classrooms(id),
    user_id      INTEGER NOT NULL REFERENCES users(id),
    joined_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(classroom_id, user_id)
);

-- ============================================================
-- Wallet Domain
-- ============================================================
CREATE TABLE IF NOT EXISTS wallets (
    id      INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL UNIQUE REFERENCES users(id),
    balance INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS transactions (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    wallet_id      INTEGER NOT NULL REFERENCES wallets(id),
    amount         INTEGER NOT NULL,
    balance_after  INTEGER NOT NULL,
    tx_type        TEXT    NOT NULL,
    description    TEXT    DEFAULT '',
    reference_type TEXT    DEFAULT '',
    reference_id   INTEGER DEFAULT 0,
    created_at     DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_transactions_wallet ON transactions(wallet_id);
CREATE INDEX IF NOT EXISTS idx_transactions_created ON transactions(created_at);

-- ============================================================
-- Company Domain
-- ============================================================
CREATE TABLE IF NOT EXISTS companies (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    owner_id         INTEGER NOT NULL REFERENCES users(id),
    name             TEXT    NOT NULL UNIQUE,
    description      TEXT    DEFAULT '',
    logo_url         TEXT    DEFAULT '',
    initial_capital  INTEGER NOT NULL CHECK (initial_capital >= 1000000),
    total_capital    INTEGER NOT NULL DEFAULT 0,
    total_shares     INTEGER NOT NULL DEFAULT 10000,
    valuation        INTEGER NOT NULL DEFAULT 0,
    listed           INTEGER NOT NULL DEFAULT 0,
    business_card    TEXT    DEFAULT '{}',
    status           TEXT    NOT NULL DEFAULT 'active'
                     CHECK (status IN ('active', 'dissolved')),
    created_at       DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS company_wallets (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    company_id INTEGER NOT NULL UNIQUE REFERENCES companies(id),
    balance    INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS company_transactions (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    company_wallet_id INTEGER NOT NULL REFERENCES company_wallets(id),
    amount            INTEGER NOT NULL,
    balance_after     INTEGER NOT NULL,
    tx_type           TEXT    NOT NULL,
    description       TEXT    DEFAULT '',
    reference_type    TEXT    DEFAULT '',
    reference_id      INTEGER DEFAULT 0,
    created_at        DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS shareholders (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    company_id       INTEGER NOT NULL REFERENCES companies(id),
    user_id          INTEGER NOT NULL REFERENCES users(id),
    shares           INTEGER NOT NULL DEFAULT 0,
    acquisition_type TEXT    NOT NULL
                     CHECK (acquisition_type IN ('founding', 'investment', 'trade')),
    acquired_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(company_id, user_id)
);

-- ============================================================
-- Feed Domain
-- ============================================================
CREATE TABLE IF NOT EXISTS channels (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    classroom_id INTEGER NOT NULL REFERENCES classrooms(id),
    name         TEXT    NOT NULL,
    slug         TEXT    NOT NULL,
    channel_type TEXT    NOT NULL,
    write_role   TEXT    NOT NULL DEFAULT 'all',
    sort_order   INTEGER NOT NULL DEFAULT 0,
    UNIQUE(classroom_id, slug)
);

CREATE TABLE IF NOT EXISTS posts (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    channel_id    INTEGER NOT NULL REFERENCES channels(id),
    author_id     INTEGER NOT NULL REFERENCES users(id),
    content       TEXT    NOT NULL,
    post_type     TEXT    NOT NULL DEFAULT 'normal',
    media         TEXT    DEFAULT '[]',
    tags          TEXT    DEFAULT '[]',
    like_count    INTEGER NOT NULL DEFAULT 0,
    comment_count INTEGER NOT NULL DEFAULT 0,
    pinned        INTEGER NOT NULL DEFAULT 0,
    created_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_posts_channel ON posts(channel_id, created_at DESC);

CREATE TABLE IF NOT EXISTS post_likes (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    post_id    INTEGER NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    user_id    INTEGER NOT NULL REFERENCES users(id),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(post_id, user_id)
);

CREATE TABLE IF NOT EXISTS comments (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    post_id    INTEGER NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    author_id  INTEGER NOT NULL REFERENCES users(id),
    content    TEXT    NOT NULL,
    media      TEXT    DEFAULT '[]',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_comments_post ON comments(post_id, created_at);

CREATE TABLE IF NOT EXISTS assignments (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    post_id       INTEGER NOT NULL UNIQUE REFERENCES posts(id),
    deadline      DATETIME NOT NULL,
    reward_amount INTEGER NOT NULL DEFAULT 0,
    max_score     INTEGER NOT NULL DEFAULT 100
);

CREATE TABLE IF NOT EXISTS submissions (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    assignment_id INTEGER NOT NULL REFERENCES assignments(id),
    student_id    INTEGER NOT NULL REFERENCES users(id),
    comment_id    INTEGER REFERENCES comments(id),
    content       TEXT    DEFAULT '',
    files         TEXT    DEFAULT '[]',
    grade         INTEGER DEFAULT NULL,
    rewarded      INTEGER NOT NULL DEFAULT 0,
    submitted_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(assignment_id, student_id)
);

CREATE TABLE IF NOT EXISTS uploads (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER NOT NULL REFERENCES users(id),
    filename    TEXT    NOT NULL,
    stored_name TEXT    NOT NULL,
    mime_type   TEXT    NOT NULL,
    size        INTEGER NOT NULL,
    path        TEXT    NOT NULL,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Freelance Market Domain
-- ============================================================
CREATE TABLE IF NOT EXISTS freelance_jobs (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    client_id       INTEGER NOT NULL REFERENCES users(id),
    title           TEXT    NOT NULL,
    description     TEXT    NOT NULL,
    budget          INTEGER NOT NULL,
    deadline        DATETIME,
    required_skills TEXT    DEFAULT '[]',
    status          TEXT    NOT NULL DEFAULT 'open'
                    CHECK (status IN ('open','in_progress','completed','disputed','cancelled')),
    freelancer_id   INTEGER REFERENCES users(id),
    escrow_amount   INTEGER NOT NULL DEFAULT 0,
    agreed_price    INTEGER DEFAULT 0,
    work_completed    INTEGER NOT NULL DEFAULT 0,
    completion_report TEXT    DEFAULT '',
    completion_media  TEXT    DEFAULT '[]',
    created_at        DATETIME DEFAULT CURRENT_TIMESTAMP,
    completed_at      DATETIME
);

CREATE TABLE IF NOT EXISTS job_applications (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id     INTEGER NOT NULL REFERENCES freelance_jobs(id),
    user_id    INTEGER NOT NULL REFERENCES users(id),
    proposal   TEXT    NOT NULL,
    price      INTEGER NOT NULL,
    status     TEXT    NOT NULL DEFAULT 'pending'
               CHECK (status IN ('pending', 'accepted', 'rejected')),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(job_id, user_id)
);

CREATE TABLE IF NOT EXISTS freelance_reviews (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id      INTEGER NOT NULL REFERENCES freelance_jobs(id),
    reviewer_id INTEGER NOT NULL REFERENCES users(id),
    reviewee_id INTEGER NOT NULL REFERENCES users(id),
    rating      INTEGER NOT NULL CHECK (rating BETWEEN 1 AND 5),
    comment     TEXT    DEFAULT '',
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(job_id, reviewer_id)
);

-- ============================================================
-- Investment Domain
-- ============================================================
CREATE TABLE IF NOT EXISTS investment_rounds (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    company_id      INTEGER NOT NULL REFERENCES companies(id),
    post_id         INTEGER REFERENCES posts(id),
    target_amount   INTEGER NOT NULL,
    offered_percent REAL    NOT NULL CHECK (offered_percent > 0 AND offered_percent < 1),
    current_amount  INTEGER NOT NULL DEFAULT 0,
    price_per_share REAL    NOT NULL,
    new_shares      INTEGER NOT NULL,
    status          TEXT    NOT NULL DEFAULT 'open'
                    CHECK (status IN ('open', 'funded', 'failed', 'cancelled')),
    expires_at      DATETIME,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    funded_at       DATETIME
);

CREATE TABLE IF NOT EXISTS investments (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    round_id   INTEGER NOT NULL REFERENCES investment_rounds(id),
    user_id    INTEGER NOT NULL REFERENCES users(id),
    amount     INTEGER NOT NULL,
    shares     INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS dividends (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    company_id   INTEGER NOT NULL REFERENCES companies(id),
    total_amount INTEGER NOT NULL,
    executed_by  INTEGER NOT NULL REFERENCES users(id),
    created_at   DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS dividend_payments (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    dividend_id INTEGER NOT NULL REFERENCES dividends(id),
    user_id     INTEGER NOT NULL REFERENCES users(id),
    shares      INTEGER NOT NULL,
    amount      INTEGER NOT NULL,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS kpi_rules (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    company_id       INTEGER NOT NULL REFERENCES companies(id),
    rule_description TEXT    NOT NULL,
    active           INTEGER NOT NULL DEFAULT 1,
    created_at       DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS kpi_revenues (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    company_id  INTEGER NOT NULL REFERENCES companies(id),
    kpi_rule_id INTEGER REFERENCES kpi_rules(id),
    amount      INTEGER NOT NULL,
    memo        TEXT    DEFAULT '',
    created_by  INTEGER NOT NULL REFERENCES users(id),
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Exchange Domain
-- ============================================================
CREATE TABLE IF NOT EXISTS stock_orders (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    company_id       INTEGER NOT NULL REFERENCES companies(id),
    user_id          INTEGER NOT NULL REFERENCES users(id),
    order_type       TEXT    NOT NULL CHECK (order_type IN ('buy', 'sell')),
    shares           INTEGER NOT NULL,
    remaining_shares INTEGER NOT NULL,
    price_per_share  INTEGER NOT NULL,
    status           TEXT    NOT NULL DEFAULT 'open'
                     CHECK (status IN ('open', 'partial', 'filled', 'cancelled')),
    created_at       DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at       DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_orders_company ON stock_orders(company_id, status, price_per_share);

CREATE TABLE IF NOT EXISTS stock_trades (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    company_id      INTEGER NOT NULL REFERENCES companies(id),
    buy_order_id    INTEGER NOT NULL REFERENCES stock_orders(id),
    sell_order_id   INTEGER NOT NULL REFERENCES stock_orders(id),
    buyer_id        INTEGER NOT NULL REFERENCES users(id),
    seller_id       INTEGER NOT NULL REFERENCES users(id),
    shares          INTEGER NOT NULL,
    price_per_share INTEGER NOT NULL,
    total_amount    INTEGER NOT NULL,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_trades_company ON stock_trades(company_id, created_at DESC);

-- ============================================================
-- Bank Domain
-- ============================================================
CREATE TABLE IF NOT EXISTS loans (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    borrower_id   INTEGER NOT NULL REFERENCES users(id),
    amount        INTEGER NOT NULL,
    remaining     INTEGER NOT NULL,
    interest_rate REAL    NOT NULL,
    penalty_rate  REAL    NOT NULL DEFAULT 0,
    purpose       TEXT    DEFAULT '',
    status        TEXT    NOT NULL DEFAULT 'pending'
                  CHECK (status IN ('pending','rejected','active','paid','overdue')),
    approved_by   INTEGER REFERENCES users(id),
    approved_at   DATETIME,
    next_payment  DATETIME,
    created_at    DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS loan_payments (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    loan_id    INTEGER NOT NULL REFERENCES loans(id),
    amount     INTEGER NOT NULL,
    principal  INTEGER NOT NULL DEFAULT 0,
    interest   INTEGER NOT NULL DEFAULT 0,
    penalty    INTEGER NOT NULL DEFAULT 0,
    pay_type   TEXT    NOT NULL
               CHECK (pay_type IN ('interest', 'repayment', 'penalty', 'auto')),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Notification Domain
-- ============================================================
CREATE TABLE IF NOT EXISTS notifications (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id        INTEGER NOT NULL REFERENCES users(id),
    notif_type     TEXT    NOT NULL,
    title          TEXT    NOT NULL,
    body           TEXT    DEFAULT '',
    reference_type TEXT    DEFAULT '',
    reference_id   INTEGER DEFAULT 0,
    is_read        INTEGER NOT NULL DEFAULT 0,
    created_at     DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_notifications_user ON notifications(user_id, is_read, created_at DESC);

CREATE TABLE IF NOT EXISTS push_subscriptions (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER NOT NULL REFERENCES users(id),
    endpoint    TEXT    NOT NULL UNIQUE,
    p256dh      TEXT    NOT NULL,
    auth        TEXT    NOT NULL,
    user_agent  TEXT    DEFAULT '',
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_push_sub_user ON push_subscriptions(user_id);


