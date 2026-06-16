package integration

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"unicode/utf8"
)

// #132 @멘션 기능 통합 테스트
// - GET /api/users/search?q= : approved 유저 이름/학번 부분일치 검색
// - 게시글/댓글 본문의 @[이름](user:ID) 멘션 → 멘션 유저에게 notif_type=mention 알림
// - 댓글 멘션 알림은 reference_type=post + anchor=comment-<id>
// - GET /api/notifications?type=mention 필터

type mentionUser struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Department string `json:"department"`
	StudentID  string `json:"student_id"`
}

type mentionNotif struct {
	NotifType     string `json:"notif_type"`
	Title         string `json:"title"`
	Body          string `json:"body"`
	ReferenceType string `json:"reference_type"`
	ReferenceID   int    `json:"reference_id"`
	Anchor        string `json:"anchor"`
}

func (ts *testServer) myID(token string) int {
	ts.t.Helper()
	resp := ts.get("/api/auth/me", token)
	if !resp.Success {
		ts.t.Fatalf("auth/me failed: %v", resp.Error)
	}
	var me struct {
		ID int `json:"id"`
	}
	json.Unmarshal(resp.Data, &me)
	if me.ID == 0 {
		ts.t.Fatalf("auth/me returned id=0: %s", string(resp.Data))
	}
	return me.ID
}

func (ts *testServer) searchUsers(token, q string) []mentionUser {
	ts.t.Helper()
	resp := ts.get("/api/users/search?q="+url.QueryEscape(q), token)
	if !resp.Success {
		ts.t.Fatalf("user search %q failed: %v", q, resp.Error)
	}
	var users []mentionUser
	json.Unmarshal(resp.Data, &users)
	return users
}

func (ts *testServer) notifs(token, query string) []mentionNotif {
	ts.t.Helper()
	resp := ts.get("/api/notifications?"+query, token)
	if !resp.Success {
		ts.t.Fatalf("get notifications failed: %v", resp.Error)
	}
	var data struct {
		Data []mentionNotif `json:"data"`
	}
	json.Unmarshal(resp.Data, &data)
	return data.Data
}

func TestUserSearch(t *testing.T) {
	ts := setupTestServer(t)

	searcherToken := ts.registerAndApprove("searcher@ewha.ac.kr", "password123", "검색러", "2024060")
	ts.registerAndApprove("target1@ewha.ac.kr", "password123", "김멘션", "2024061")
	ts.registerAndApprove("target2@ewha.ac.kr", "password123", "김멘션", "2024062") // 동명이인
	ts.register("pending@ewha.ac.kr", "password123", "김펜딩", "2024063")           // 미승인

	// 동명이인 구분용 전공 포함 가입 (#132 — 드롭다운에 이름+전공 표시)
	regResp := ts.post("/api/auth/register", map[string]string{
		"email":      "dept@ewha.ac.kr",
		"password":   "password123",
		"name":       "전공있음",
		"student_id": "2024064",
		"department": "컴퓨터공학과",
	}, "")
	var regData struct {
		User struct {
			ID int `json:"id"`
		} `json:"user"`
	}
	json.Unmarshal(regResp.Data, &regData)
	adminToken := ts.login(testAdminEmail, testAdminPass)
	ts.approveUser(adminToken, regData.User.ID)

	t.Run("이름 부분일치 — 동명이인 둘 다 반환", func(t *testing.T) {
		users := ts.searchUsers(searcherToken, "김멘")
		if len(users) != 2 {
			t.Fatalf("expected 2 users, got %d: %+v", len(users), users)
		}
		if users[0].ID == users[1].ID {
			t.Error("동명이인은 서로 다른 id여야 합니다")
		}
		for _, u := range users {
			if u.Name != "김멘션" {
				t.Errorf("expected name 김멘션, got %s", u.Name)
			}
		}
	})

	t.Run("학번 부분일치", func(t *testing.T) {
		users := ts.searchUsers(searcherToken, "2024061")
		if len(users) != 1 {
			t.Fatalf("expected 1 user, got %d", len(users))
		}
		if users[0].Name != "김멘션" {
			t.Errorf("expected 김멘션, got %s", users[0].Name)
		}
	})

	t.Run("미승인 유저 제외", func(t *testing.T) {
		users := ts.searchUsers(searcherToken, "김펜딩")
		if len(users) != 0 {
			t.Errorf("pending 유저가 검색되면 안 됩니다: %+v", users)
		}
	})

	t.Run("검색 결과에 전공 포함 (동명이인 구분용)", func(t *testing.T) {
		users := ts.searchUsers(searcherToken, "전공있음")
		if len(users) != 1 {
			t.Fatalf("expected 1 user, got %d", len(users))
		}
		if users[0].Department != "컴퓨터공학과" {
			t.Errorf("expected department 컴퓨터공학과, got %q", users[0].Department)
		}
	})

	t.Run("빈 검색어는 빈 결과", func(t *testing.T) {
		users := ts.searchUsers(searcherToken, "")
		if len(users) != 0 {
			t.Errorf("expected 0 users for empty query, got %d", len(users))
		}
	})
}

// setupMentionFixture: 교실+채널 만들고 작성자/멘션대상 학생 가입, (channelID, writerToken, targetToken, targetID) 반환
func setupMentionFixture(t *testing.T, ts *testServer) (int, string, string, int) {
	t.Helper()
	adminToken := ts.login(testAdminEmail, testAdminPass)

	classResp := ts.post("/api/classrooms", map[string]string{"name": "멘션테스트반"}, adminToken)
	var classData struct {
		ID int `json:"id"`
	}
	json.Unmarshal(classResp.Data, &classData)

	channelsResp := ts.get(fmt.Sprintf("/api/classrooms/%d/channels", classData.ID), adminToken)
	var channels []struct {
		ID        int    `json:"id"`
		WriteRole string `json:"write_role"`
	}
	json.Unmarshal(channelsResp.Data, &channels)
	// 학생이 글 쓸 수 있는 채널 선택 (공지는 write_role=admin)
	channelID := 0
	for _, ch := range channels {
		if ch.WriteRole != "admin" {
			channelID = ch.ID
			break
		}
	}
	if channelID == 0 {
		t.Fatal("학생 작성 가능 채널이 없습니다")
	}

	classDetailResp := ts.get(fmt.Sprintf("/api/classrooms/%d", classData.ID), adminToken)
	var classDetail struct {
		Classroom struct {
			Code string `json:"code"`
		} `json:"classroom"`
	}
	json.Unmarshal(classDetailResp.Data, &classDetail)

	writerToken := ts.registerAndApprove("writer@ewha.ac.kr", "password123", "글쓴이", "2024070")
	targetToken := ts.registerAndApprove("target@ewha.ac.kr", "password123", "멘션대상", "2024071")
	ts.post("/api/classrooms/join", map[string]string{"code": classDetail.Classroom.Code}, writerToken)
	ts.post("/api/classrooms/join", map[string]string{"code": classDetail.Classroom.Code}, targetToken)

	return channelID, writerToken, targetToken, ts.myID(targetToken)
}

func TestPostMentionNotification(t *testing.T) {
	ts := setupTestServer(t)
	channelID, writerToken, targetToken, targetID := setupMentionFixture(t, ts)

	postResp := ts.post(fmt.Sprintf("/api/channels/%d/posts", channelID), map[string]string{
		"content": fmt.Sprintf("안녕하세요 @[멘션대상](user:%d) 확인 부탁해요", targetID),
	}, writerToken)
	if !postResp.Success {
		t.Fatalf("create post failed: %v", postResp.Error)
	}
	var postData struct {
		ID int `json:"id"`
	}
	json.Unmarshal(postResp.Data, &postData)

	t.Run("게시글 멘션 → mention 알림 (refType=post, anchor 없음)", func(t *testing.T) {
		found := false
		for _, n := range ts.notifs(targetToken, "limit=10") {
			if n.NotifType == "mention" && n.ReferenceID == postData.ID {
				found = true
				if n.ReferenceType != "post" {
					t.Errorf("expected reference_type post, got %s", n.ReferenceType)
				}
				if n.Anchor != "" {
					t.Errorf("게시글 멘션은 anchor가 비어야 합니다, got %s", n.Anchor)
				}
			}
		}
		if !found {
			t.Error("멘션 알림이 전송되지 않았습니다")
		}
	})

	t.Run("긴 한글 본문 미리보기 — UTF-8 안 깨짐", func(t *testing.T) {
		long := strings.Repeat("가나다라마바사아자차", 12) // 120 runes (360 bytes)
		ts.post(fmt.Sprintf("/api/channels/%d/posts", channelID), map[string]string{
			"content": fmt.Sprintf("@[멘션대상](user:%d) %s", targetID, long),
		}, writerToken)

		for _, n := range ts.notifs(targetToken, "type=mention&limit=5") {
			if !utf8.ValidString(n.Body) || strings.ContainsRune(n.Body, '�') {
				t.Errorf("알림 미리보기 UTF-8 깨짐: %q", n.Body)
			}
		}
	})

	t.Run("자기 멘션은 알림 없음", func(t *testing.T) {
		writerID := ts.myID(writerToken)
		ts.post(fmt.Sprintf("/api/channels/%d/posts", channelID), map[string]string{
			"content": fmt.Sprintf("셀프멘션 @[글쓴이](user:%d)", writerID),
		}, writerToken)
		for _, n := range ts.notifs(writerToken, "limit=10") {
			if n.NotifType == "mention" {
				t.Error("자기 자신 멘션은 알림이 가면 안 됩니다")
			}
		}
	})
}

func TestCommentMentionNotification(t *testing.T) {
	ts := setupTestServer(t)
	channelID, writerToken, targetToken, targetID := setupMentionFixture(t, ts)
	writerID := ts.myID(writerToken)

	// writer가 게시글 작성, target이 댓글에서 제3자가 아닌 writer 멘션 검증을 위해
	// 글: writer / 댓글: target
	postResp := ts.post(fmt.Sprintf("/api/channels/%d/posts", channelID), map[string]string{
		"content": "댓글 멘션 테스트 글",
	}, writerToken)
	var postData struct {
		ID int `json:"id"`
	}
	json.Unmarshal(postResp.Data, &postData)

	t.Run("댓글 멘션 → anchor=comment-<id>, refType=post(refID=postID)", func(t *testing.T) {
		// writer가 자기 글 댓글에서 target 멘션
		commentResp := ts.post(fmt.Sprintf("/api/posts/%d/comments", postData.ID), map[string]string{
			"content": fmt.Sprintf("@[멘션대상](user:%d) 댓글에서 불러요", targetID),
		}, writerToken)
		if !commentResp.Success {
			t.Fatalf("comment failed: %v", commentResp.Error)
		}
		var commentData struct {
			ID int `json:"id"`
		}
		json.Unmarshal(commentResp.Data, &commentData)

		found := false
		for _, n := range ts.notifs(targetToken, "limit=10") {
			if n.NotifType == "mention" && n.ReferenceID == postData.ID {
				found = true
				if n.ReferenceType != "post" {
					t.Errorf("expected reference_type post, got %s", n.ReferenceType)
				}
				wantAnchor := fmt.Sprintf("comment-%d", commentData.ID)
				if n.Anchor != wantAnchor {
					t.Errorf("expected anchor %s, got %s", wantAnchor, n.Anchor)
				}
			}
		}
		if !found {
			t.Error("댓글 멘션 알림이 전송되지 않았습니다")
		}
	})

	t.Run("글 작성자가 멘션된 댓글 → mention만 받고 new_comment 중복 없음", func(t *testing.T) {
		// target이 writer의 글에 writer를 멘션한 댓글 작성
		content := fmt.Sprintf("@[글쓴이](user:%d) 글 잘 봤어요", writerID)
		commentResp := ts.post(fmt.Sprintf("/api/posts/%d/comments", postData.ID), map[string]string{
			"content": content,
		}, targetToken)
		if !commentResp.Success {
			t.Fatalf("comment failed: %v", commentResp.Error)
		}

		// 알림 미리보기는 마크업이 벗겨진 형태 (@이름)
		strippedBody := "@글쓴이 글 잘 봤어요"
		mentionCount, newCommentCount := 0, 0
		for _, n := range ts.notifs(writerToken, "limit=20") {
			if n.Body != strippedBody {
				continue
			}
			switch n.NotifType {
			case "mention":
				mentionCount++
			case "new_comment":
				newCommentCount++
			}
		}
		if mentionCount != 1 {
			t.Errorf("expected 1 mention notification, got %d", mentionCount)
		}
		if newCommentCount != 0 {
			t.Errorf("멘션된 글 작성자에게 new_comment 중복 알림이 가면 안 됩니다, got %d", newCommentCount)
		}
	})

	t.Run("멘션 없는 댓글은 기존 new_comment 알림 유지", func(t *testing.T) {
		body := "멘션 없는 일반 댓글"
		ts.post(fmt.Sprintf("/api/posts/%d/comments", postData.ID), map[string]string{
			"content": body,
		}, targetToken)

		found := false
		for _, n := range ts.notifs(writerToken, "limit=20") {
			if n.NotifType == "new_comment" && n.Body == body {
				found = true
			}
		}
		if !found {
			t.Error("일반 댓글의 new_comment 알림이 사라졌습니다 (회귀)")
		}
	})
}

func TestNotificationTypeFilter(t *testing.T) {
	ts := setupTestServer(t)
	channelID, writerToken, targetToken, targetID := setupMentionFixture(t, ts)

	// target에게 mention 1건 + new_comment 1건 만들기
	postResp := ts.post(fmt.Sprintf("/api/channels/%d/posts", channelID), map[string]string{
		"content": fmt.Sprintf("필터테스트 @[멘션대상](user:%d)", targetID),
	}, writerToken)
	var postData struct {
		ID int `json:"id"`
	}
	json.Unmarshal(postResp.Data, &postData)

	// target이 글 작성 → writer가 댓글 → target에게 new_comment
	targetPostResp := ts.post(fmt.Sprintf("/api/channels/%d/posts", channelID), map[string]string{
		"content": "타겟의 글",
	}, targetToken)
	var targetPostData struct {
		ID int `json:"id"`
	}
	json.Unmarshal(targetPostResp.Data, &targetPostData)
	ts.post(fmt.Sprintf("/api/posts/%d/comments", targetPostData.ID), map[string]string{
		"content": "일반 댓글",
	}, writerToken)

	t.Run("type=mention은 멘션만 반환", func(t *testing.T) {
		notifs := ts.notifs(targetToken, "type=mention&limit=20")
		if len(notifs) == 0 {
			t.Fatal("멘션 알림이 없습니다")
		}
		for _, n := range notifs {
			if n.NotifType != "mention" {
				t.Errorf("type=mention 필터에 %s가 섞여 있습니다", n.NotifType)
			}
		}
	})

	t.Run("필터 없으면 전체 반환", func(t *testing.T) {
		types := map[string]bool{}
		for _, n := range ts.notifs(targetToken, "limit=20") {
			types[n.NotifType] = true
		}
		if !types["mention"] || !types["new_comment"] {
			t.Errorf("전체 조회에 mention+new_comment 둘 다 있어야 합니다: %v", types)
		}
	})
}

// TestUpdatePostMentionNotification — #134
// 게시글 수정 시 "새로 추가된" 멘션만 알림. 기존 멘션 재알림 금지(수정 스팸 방지).
func TestUpdatePostMentionNotification(t *testing.T) {
	ts := setupTestServer(t)
	channelID, writerToken, targetToken, targetID := setupMentionFixture(t, ts)

	// 수정 때 새로 추가될 두 번째 멘션 대상 (채널 가입 불필요 — 멘션 알림은 유저 존재만 검증)
	target2Token := ts.registerAndApprove("target2@ewha.ac.kr", "password123", "멘션대상2", "2024072")
	target2ID := ts.myID(target2Token)

	// 글 생성: targetA만 멘션
	postResp := ts.post(fmt.Sprintf("/api/channels/%d/posts", channelID), map[string]string{
		"content": fmt.Sprintf("처음 글 @[멘션대상](user:%d)", targetID),
	}, writerToken)
	if !postResp.Success {
		t.Fatalf("create post failed: %v", postResp.Error)
	}
	var postData struct {
		ID int `json:"id"`
	}
	json.Unmarshal(postResp.Data, &postData)

	mentionCount := func(token string) int {
		n := 0
		for _, x := range ts.notifs(token, "type=mention&limit=50") {
			if x.ReferenceID == postData.ID {
				n++
			}
		}
		return n
	}

	// 생성 직후 targetA 멘션 알림 1건 (#132 기존 동작)
	if c := mentionCount(targetToken); c != 1 {
		t.Fatalf("생성 직후 targetA 멘션 알림 1건이어야 함, got %d", c)
	}

	// 수정: targetA 유지 + targetB 신규 추가
	updResp := ts.put(fmt.Sprintf("/api/posts/%d", postData.ID), map[string]string{
		"content": fmt.Sprintf("수정 글 @[멘션대상](user:%d) @[멘션대상2](user:%d)", targetID, target2ID),
	}, writerToken)
	if !updResp.Success {
		t.Fatalf("update post failed: %v", updResp.Error)
	}

	t.Run("수정으로 추가된 멘션(targetB) → 알림 1건", func(t *testing.T) {
		if c := mentionCount(target2Token); c != 1 {
			t.Errorf("신규 멘션 알림 1건이어야 함, got %d", c)
		}
	})

	t.Run("구본문에 이미 있던 멘션(targetA) → 재알림 없음", func(t *testing.T) {
		if c := mentionCount(targetToken); c != 1 {
			t.Errorf("기존 멘션은 재알림 금지(여전히 1건이어야 함), got %d", c)
		}
	})

	t.Run("동일 본문 재저장 → 신규 알림 없음", func(t *testing.T) {
		ts.put(fmt.Sprintf("/api/posts/%d", postData.ID), map[string]string{
			"content": fmt.Sprintf("수정 글 @[멘션대상](user:%d) @[멘션대상2](user:%d)", targetID, target2ID),
		}, writerToken)
		if c := mentionCount(targetToken); c != 1 {
			t.Errorf("targetA 재알림 금지, got %d", c)
		}
		if c := mentionCount(target2Token); c != 1 {
			t.Errorf("targetB 재알림 금지, got %d", c)
		}
	})
}
