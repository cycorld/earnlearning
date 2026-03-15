package persistence

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// SeedDevData populates the local DB with diverse test data for manual testing.
// Triggered by SEED_DEV=1 environment variable.
func SeedDevData(db *sql.DB) error {
	// Check if dev data already seeded (use a marker)
	var count int
	db.QueryRow("SELECT COUNT(*) FROM users WHERE email = 'student1@ewha.ac.kr'").Scan(&count)
	if count > 0 {
		log.Println("[seed-dev] dev data already exists, skipping")
		return nil
	}

	log.Println("[seed-dev] seeding development data...")

	rng := rand.New(rand.NewSource(42)) // deterministic for reproducibility

	hash, err := bcrypt.GenerateFromPassword([]byte("test1234"), 8)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	pw := string(hash)

	// ─── Students (30명: 다양한 상태) ──────────────────────────────
	type student struct {
		id         int
		email      string
		name       string
		department string
		studentID  string
		status     string
	}

	names := []string{
		"김민수", "이서연", "박지훈", "최수아", "정우진",
		"강예린", "조현우", "윤서윤", "임도현", "한지은",
		"송민재", "오하영", "배준서", "홍서진", "류시우",
		"권나연", "남도윤", "문하은", "장서준", "신유진",
		"김태희", "이준혁", "박소연", "최영호", "정다은",
		"강민서", "조은비", "윤재현", "임지수", "한도현",
	}
	departments := []string{
		"컴퓨터공학과", "경영학과", "디자인학과", "산업공학과", "전자공학과",
		"미디어학과", "국제학부", "소프트웨어학과", "통계학과", "경제학과",
	}
	statuses := []string{
		"approved", "approved", "approved", "approved", "approved",
		"approved", "approved", "approved", "approved", "approved",
		"approved", "approved", "approved", "approved", "approved",
		"approved", "approved", "approved", "approved", "approved",
		"pending", "pending", "pending", "pending", "pending",
		"pending", "pending", "pending", "rejected", "rejected",
	}

	students := make([]student, len(names))
	for i, name := range names {
		email := fmt.Sprintf("student%d@ewha.ac.kr", i+1)
		sid := fmt.Sprintf("20260%05d", i+1)
		status := statuses[i]
		bio := ""
		if i%3 == 0 {
			bio = fmt.Sprintf("%s 전공, 스타트업에 관심이 많습니다.", departments[i%len(departments)])
		}

		_, err := db.Exec(`INSERT INTO users (email, password, name, department, student_id, role, status, bio)
			VALUES (?, ?, ?, ?, ?, 'student', ?, ?)`,
			email, pw, name, departments[i%len(departments)], sid, status, bio)
		if err != nil {
			return fmt.Errorf("insert student %d: %w", i, err)
		}

		var id int
		db.QueryRow("SELECT id FROM users WHERE email = ?", email).Scan(&id)
		students[i] = student{id: id, email: email, name: name, department: departments[i%len(departments)], studentID: sid, status: status}
	}

	// Create wallets for approved students
	for _, s := range students {
		if s.status == "approved" {
			db.Exec("INSERT OR IGNORE INTO wallets (user_id, balance) VALUES (?, ?)", s.id, 50000000)
		}
	}

	// ─── Get admin ID ──────────────────────────────────────────────
	var adminID int
	err = db.QueryRow("SELECT id FROM users WHERE role = 'admin' LIMIT 1").Scan(&adminID)
	if err != nil {
		return fmt.Errorf("find admin: %w", err)
	}

	// ─── Classroom ─────────────────────────────────────────────────
	res, err := db.Exec(`INSERT INTO classrooms (name, code, created_by, initial_capital)
		VALUES ('스타트업을 위한 코딩입문 A반', 'ABC123', ?, 50000000)`, adminID)
	if err != nil {
		return fmt.Errorf("create classroom: %w", err)
	}
	classroomID, _ := res.LastInsertId()

	// Add admin as member
	db.Exec("INSERT INTO classroom_members (classroom_id, user_id) VALUES (?, ?)", classroomID, adminID)

	// Default channels
	type ch struct {
		name, slug, chType, writeRole string
		sortOrder                     int
	}
	channels := []ch{
		{"공지", "notice", "notice", "admin", 1},
		{"자유", "free", "free", "all", 2},
		{"과제", "assignment", "assignment", "admin", 3},
		{"쇼케이스", "showcase", "showcase", "all", 4},
		{"외주마켓", "market", "market", "all", 5},
		{"투자라운지", "invest", "invest", "all", 6},
		{"거래소", "exchange", "exchange", "all", 7},
	}
	channelIDs := make(map[string]int64)
	for _, c := range channels {
		r, err := db.Exec(`INSERT INTO channels (classroom_id, name, slug, channel_type, write_role, sort_order)
			VALUES (?, ?, ?, ?, ?, ?)`, classroomID, c.name, c.slug, c.chType, c.writeRole, c.sortOrder)
		if err != nil {
			return fmt.Errorf("create channel %s: %w", c.slug, err)
		}
		id, _ := r.LastInsertId()
		channelIDs[c.slug] = id
	}

	// Add approved students as classroom members
	approvedStudents := []student{}
	for _, s := range students {
		if s.status == "approved" {
			db.Exec("INSERT INTO classroom_members (classroom_id, user_id) VALUES (?, ?)", classroomID, s.id)
			approvedStudents = append(approvedStudents, s)
		}
	}

	// ─── Posts (다양한 채널에 30+개) ────────────────────────────────
	now := time.Now()

	// 공지 채널 posts
	noticePosts := []struct {
		content string
		tags    string
		pinned  int
	}{
		{"📌 **수업 안내**\n\n안녕하세요, '스타트업을 위한 코딩입문' 수업에 오신 것을 환영합니다!\n\n- 수업 시간: 매주 화/목 14:00~15:30\n- 장소: 신공학관 B102\n- 준비물: 노트북\n\n궁금한 점은 자유 채널에 질문해주세요.", `["공지","필독"]`, 1},
		{"📋 **1주차 과제 안내**\n\n이번 주 과제는 팀 아이디어 브레인스토밍입니다.\n\n1. 3~4인 팀 구성\n2. 문제 정의 → 솔루션 아이디어 정리\n3. 제출 마감: 3/20(금) 23:59\n\n과제 채널에 제출해주세요.", `["과제","1주차"]`, 0},
		{"🎤 **게스트 스피커 안내**\n\n3/25(화) 수업에 '당근마켓' 초기 멤버 김OO님이 오십니다.\n질문 미리 준비해오세요!", `["공지","특강"]`, 0},
	}
	for i, p := range noticePosts {
		createdAt := now.Add(-time.Duration(len(noticePosts)-i) * 24 * time.Hour)
		db.Exec(`INSERT INTO posts (channel_id, author_id, content, post_type, tags, pinned, like_count, comment_count, created_at)
			VALUES (?, ?, ?, 'normal', ?, ?, ?, ?, ?)`,
			channelIDs["notice"], adminID, p.content, p.tags, p.pinned, rng.Intn(15)+5, rng.Intn(5), createdAt)
	}

	// 자유 채널 posts (15개 — 다양한 학생들이 작성)
	freePosts := []string{
		"혹시 같이 스터디 하실 분 계신가요? React 공부하고 싶어요 🙋‍♀️",
		"오늘 수업 내용 정리해봤습니다. 궁금한 점 있으시면 댓글 남겨주세요!\n\n## HTML 기초\n- `<div>`: 컨테이너\n- `<p>`: 문단\n- `<a>`: 링크",
		"팀 프로젝트 주제 고민 중인데... '대학생 밀키트 배달' 어떻게 생각하세요?",
		"VS Code 단축키 정리\n\n| 단축키 | 기능 |\n|--------|------|\n| Cmd+P | 파일 검색 |\n| Cmd+Shift+P | 명령 팔레트 |\n| Cmd+D | 동일 단어 선택 |",
		"첫 코딩 수업이라 긴장되네요 ㅠㅠ 다들 파이팅!",
		"JavaScript에서 `let`이랑 `const` 차이가 뭔가요?",
		"팀원 모집합니다! 배달 앱 만들어보고 싶은 분?\n- 필요 역할: 디자이너 1명, 개발자 2명\n- DM 주세요!",
		"오늘 GitHub 특강 너무 유익했어요. 감사합니다 교수님!",
		"```python\nprint('Hello, World!')\n```\n첫 번째 프로그램 작성 완료! 🎉",
		"다음 주 과제 마감이 언제인지 아시는 분?",
		"피그마로 와이어프레임 만들어봤는데 피드백 부탁드려요!\n\n로그인 → 대시보드 → 주문 플로우로 구성했습니다.",
		"CSS Flexbox 진짜 헷갈리네요... justify-content랑 align-items 차이를 모르겠어요 😭",
		"오늘 점심 학식 추천합니다. 돈까스 맛있었어요 🍽",
		"Git merge conflict 해결하는 법 공유합니다!\n\n1. `git status`로 충돌 파일 확인\n2. 파일 열어서 `<<<<<<` 부분 수정\n3. `git add .` → `git commit`",
		"스타트업 아이디어 투표해주세요!\n\n1. 캠퍼스 중고거래\n2. 스터디 매칭\n3. 식단 관리 앱",
	}
	freeTagSets := []string{
		`["스터디","모집"]`, `["수업정리"]`, `["아이디어"]`, `["팁","개발"]`, `[]`,
		`["질문"]`, `["팀원모집"]`, `[]`, `["코딩"]`, `["질문"]`,
		`["디자인","피드백"]`, `["질문","CSS"]`, `[]`, `["Git","팁"]`, `["투표"]`,
	}

	type postRef struct {
		id       int64
		authorID int
	}
	var freePostRefs []postRef
	for i, content := range freePosts {
		author := approvedStudents[i%len(approvedStudents)]
		createdAt := now.Add(-time.Duration(len(freePosts)-i)*3*time.Hour - time.Duration(rng.Intn(60))*time.Minute)
		likeCount := rng.Intn(20)
		commentCount := rng.Intn(8)
		r, _ := db.Exec(`INSERT INTO posts (channel_id, author_id, content, post_type, tags, like_count, comment_count, created_at)
			VALUES (?, ?, ?, 'normal', ?, ?, ?, ?)`,
			channelIDs["free"], author.id, content, freeTagSets[i], likeCount, commentCount, createdAt)
		pid, _ := r.LastInsertId()
		freePostRefs = append(freePostRefs, postRef{id: pid, authorID: author.id})
	}

	// 쇼케이스 채널 posts (5개 — 프로젝트 소개)
	showcasePosts := []struct {
		content string
		tags    string
	}{
		{"# 🚀 CampusBite\n\n대학생 맞춤 밀키트 구독 서비스\n\n## 문제\n- 자취생 식사 해결이 어려움\n- 기존 밀키트는 1인분이 없음\n\n## 솔루션\n- 주 3회 1인분 밀키트 배달\n- 학교 근처 식재료 업체 연계\n\n## 팀\n- PM: 김민수\n- 개발: 이서연, 박지훈\n- 디자인: 최수아", `["프로젝트","밀키트"]`},
		{"# 📚 StudyMatch\n\n스터디 매칭 플랫폼\n\n같은 과목, 같은 시간대 학생을 매칭해주는 서비스입니다.\n\n- 과목별 매칭\n- 카페 추천\n- 출석 관리", `["프로젝트","스터디"]`},
		{"# 🎨 DesignFolio\n\n디자인 포트폴리오 공유 플랫폼\n\n학생들이 자유롭게 작품을 올리고 피드백 받을 수 있습니다.", `["프로젝트","디자인"]`},
		{"# 💰 PennyWise\n\n대학생 가계부 & 절약 챌린지 앱\n\n- AI 기반 지출 분석\n- 친구들과 절약 챌린지\n- 월말 리포트", `["프로젝트","핀테크"]`},
		{"# 🏃 FitCampus\n\n캠퍼스 운동 매칭 서비스\n\n- 배드민턴, 농구 등 종목별 매칭\n- 교내 시설 예약 연동\n- 운동 기록 관리", `["프로젝트","헬스케어"]`},
	}
	for i, p := range showcasePosts {
		author := approvedStudents[(i*3)%len(approvedStudents)]
		createdAt := now.Add(-time.Duration(len(showcasePosts)-i) * 48 * time.Hour)
		db.Exec(`INSERT INTO posts (channel_id, author_id, content, post_type, tags, like_count, comment_count, created_at)
			VALUES (?, ?, ?, 'normal', ?, ?, ?, ?)`,
			channelIDs["showcase"], author.id, p.content, p.tags, rng.Intn(30)+10, rng.Intn(10)+2, createdAt)
	}

	// 과제 채널 posts (3개)
	assignmentPosts := []struct {
		content  string
		tags     string
		deadline time.Time
		reward   int
	}{
		{"# 📝 1주차 과제: 팀 아이디어 제안서\n\n팀별로 스타트업 아이디어를 정리하여 제출하세요.\n\n## 제출 형식\n1. 문제 정의\n2. 타겟 고객\n3. 솔루션 요약\n4. 경쟁 분석", `["과제","1주차"]`, now.Add(5 * 24 * time.Hour), 1000000},
		{"# 📝 2주차 과제: 프로토타입 UI\n\nFigma 또는 손그림으로 앱의 주요 화면 3개를 디자인하세요.", `["과제","2주차"]`, now.Add(12 * 24 * time.Hour), 1500000},
		{"# 📝 3주차 과제: 랜딩 페이지\n\nHTML/CSS로 팀 프로젝트 소개 랜딩 페이지를 만드세요.", `["과제","3주차"]`, now.Add(19 * 24 * time.Hour), 2000000},
	}
	for i, p := range assignmentPosts {
		createdAt := now.Add(-time.Duration(len(assignmentPosts)-i) * 7 * 24 * time.Hour)
		r, _ := db.Exec(`INSERT INTO posts (channel_id, author_id, content, post_type, tags, like_count, comment_count, created_at)
			VALUES (?, ?, ?, 'assignment', ?, 0, ?, ?)`,
			channelIDs["assignment"], adminID, p.content, p.tags, rng.Intn(5), createdAt)
		pid, _ := r.LastInsertId()
		db.Exec(`INSERT INTO assignments (post_id, deadline, reward_amount, max_score)
			VALUES (?, ?, ?, 100)`, pid, p.deadline, p.reward)
	}

	// ─── Comments (자유 채널 포스트들에 댓글 30+개) ─────────────────
	commentTexts := []string{
		"좋은 정보 감사합니다!",
		"저도 같은 생각이에요 ㅎㅎ",
		"질문 있는데요, 좀 더 자세히 설명해주실 수 있나요?",
		"와 진짜 도움이 많이 됐어요 👏",
		"저도 참여하고 싶어요!",
		"혹시 참고하신 자료 있으실까요?",
		"맞아요, 저도 그 부분이 헷갈렸어요",
		"수업 때 배운 내용이랑 연결되네요",
		"우와 정리 너무 잘 하셨다 ✨",
		"다음에도 이런 내용 공유해주세요!",
		"팀플 하실 때 연락 주세요~",
		"이거 진짜 유용해요!",
		"저번에 이거 때문에 삽질했는데ㅠ 미리 알았으면...",
		"화이팅입니다! 💪",
		"공감합니다 ㅋㅋ",
	}

	commentIdx := 0
	for _, pr := range freePostRefs {
		numComments := rng.Intn(5) + 1 // 1~5 comments per post
		for j := 0; j < numComments; j++ {
			author := approvedStudents[rng.Intn(len(approvedStudents))]
			if author.id == pr.authorID && len(approvedStudents) > 1 {
				author = approvedStudents[(rng.Intn(len(approvedStudents)-1)+1)%len(approvedStudents)]
			}
			text := commentTexts[commentIdx%len(commentTexts)]
			commentIdx++
			createdAt := now.Add(-time.Duration(rng.Intn(72)) * time.Hour)
			db.Exec(`INSERT INTO comments (post_id, author_id, content, created_at)
				VALUES (?, ?, ?, ?)`, pr.id, author.id, text, createdAt)
		}
	}

	// Update comment_count to match actual counts
	db.Exec(`UPDATE posts SET comment_count = (SELECT COUNT(*) FROM comments WHERE comments.post_id = posts.id)`)

	// ─── Likes (자유 채널 포스트에 좋아요) ──────────────────────────
	for _, pr := range freePostRefs {
		numLikes := rng.Intn(8) + 1
		used := map[int]bool{}
		for j := 0; j < numLikes && j < len(approvedStudents); j++ {
			s := approvedStudents[rng.Intn(len(approvedStudents))]
			if used[s.id] {
				continue
			}
			used[s.id] = true
			db.Exec(`INSERT OR IGNORE INTO post_likes (post_id, user_id) VALUES (?, ?)`, pr.id, s.id)
		}
	}
	// Update like_count to match actual counts
	db.Exec(`UPDATE posts SET like_count = (SELECT COUNT(*) FROM post_likes WHERE post_likes.post_id = posts.id)`)

	// ─── Companies (3개) ───────────────────────────────────────────
	companyData := []struct {
		ownerIdx    int
		name        string
		description string
		capital     int
	}{
		{0, "CampusBite", "대학생 맞춤 밀키트 구독 서비스", 5000000},
		{3, "StudyMatch", "스터디 매칭 플랫폼", 3000000},
		{6, "PennyWise", "대학생 가계부 & 절약 챌린지 앱", 4000000},
	}

	type companyRef struct {
		id      int64
		ownerID int
	}
	var companies []companyRef
	for _, c := range companyData {
		owner := approvedStudents[c.ownerIdx]
		r, err := db.Exec(`INSERT INTO companies (owner_id, name, description, initial_capital, total_capital, total_shares, valuation, listed)
			VALUES (?, ?, ?, ?, ?, 10000, ?, 0)`, owner.id, c.name, c.description, c.capital, c.capital, c.capital*2)
		if err != nil {
			log.Printf("[seed-dev] company %s: %v", c.name, err)
			continue
		}
		cid, _ := r.LastInsertId()
		companies = append(companies, companyRef{id: cid, ownerID: owner.id})

		// Company wallet
		db.Exec(`INSERT INTO company_wallets (company_id, balance) VALUES (?, ?)`, cid, c.capital)

		// Founder shares (100%)
		db.Exec(`INSERT INTO shareholders (company_id, user_id, shares, acquisition_type) VALUES (?, ?, 10000, 'founding')`,
			cid, owner.id)
	}

	// ─── Freelance Jobs (5개) ──────────────────────────────────────
	jobData := []struct {
		clientIdx int
		title     string
		desc      string
		budget    int
		skills    string
		status    string
	}{
		{0, "랜딩 페이지 디자인", "CampusBite 서비스 소개 랜딩 페이지 디자인이 필요합니다.\n- 반응형 웹\n- 밝은 톤", 500000, `["Figma","UI/UX"]`, "open"},
		{3, "앱 로고 제작", "StudyMatch 로고 디자인해주실 분 구합니다.\n심플하고 학문적인 느낌", 300000, `["일러스트","브랜딩"]`, "open"},
		{1, "데이터 크롤링", "학교 공지사항 자동 수집 스크립트 개발", 400000, `["Python","크롤링"]`, "in_progress"},
		{6, "소개 영상 제작", "30초 앱 소개 영상 편집", 600000, `["영상편집","모션그래픽"]`, "open"},
		{9, "카카오톡 챗봇", "주문 접수용 카카오톡 챗봇 개발", 800000, `["Python","API"]`, "completed"},
	}
	for _, j := range jobData {
		client := approvedStudents[j.clientIdx]
		deadline := now.Add(14 * 24 * time.Hour)
		freelancerID := sql.NullInt64{}
		if j.status == "in_progress" || j.status == "completed" {
			f := approvedStudents[(j.clientIdx+5)%len(approvedStudents)]
			freelancerID = sql.NullInt64{Int64: int64(f.id), Valid: true}
		}
		db.Exec(`INSERT INTO freelance_jobs (client_id, title, description, budget, deadline, required_skills, status, freelancer_id)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			client.id, j.title, j.desc, j.budget, deadline, j.skills, j.status, freelancerID)
	}

	// ─── Transactions (approved 학생들에게 거래 내역) ───────────────
	for i, s := range approvedStudents {
		if i >= 5 {
			break
		}
		var walletID int
		err := db.QueryRow("SELECT id FROM wallets WHERE user_id = ?", s.id).Scan(&walletID)
		if err != nil {
			continue
		}
		// Initial capital transaction
		db.Exec(`INSERT INTO transactions (wallet_id, amount, balance_after, tx_type, description, reference_type, reference_id)
			VALUES (?, 50000000, 50000000, 'initial_capital', '초기 자본금 지급', 'classroom', ?)`,
			walletID, classroomID)
	}

	// ─── Notifications (일부 학생에게 알림) ─────────────────────────
	notifData := []struct {
		userIdx   int
		notifType string
		title     string
		body      string
	}{
		{0, "post_like", "좋아요 알림", "이서연님이 게시글에 좋아요를 눌렀습니다."},
		{0, "comment", "댓글 알림", "박지훈님이 댓글을 남겼습니다."},
		{1, "post_like", "좋아요 알림", "김민수님이 게시글에 좋아요를 눌렀습니다."},
		{2, "assignment", "과제 알림", "새로운 과제가 등록되었습니다."},
		{3, "system", "시스템 알림", "회원 승인이 완료되었습니다."},
	}
	for _, n := range notifData {
		s := approvedStudents[n.userIdx%len(approvedStudents)]
		db.Exec(`INSERT INTO notifications (user_id, notif_type, title, body)
			VALUES (?, ?, ?, ?)`, s.id, n.notifType, n.title, n.body)
	}

	log.Printf("[seed-dev] done! Created %d students, 1 classroom, %d channels, 20+ posts, comments, likes, 3 companies, 5 jobs",
		len(students), len(channels))
	log.Println("[seed-dev] Test accounts: student1@ewha.ac.kr ~ student30@ewha.ac.kr (password: test1234)")
	log.Println("[seed-dev] Pending users: student21~27, Rejected: student29~30")

	return nil
}
