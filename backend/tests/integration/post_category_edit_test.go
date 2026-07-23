package integration

import (
	"encoding/json"
	"fmt"
	"testing"
)

// TestAdminPostCategoryEdit — 게시글의 카테고리(채널) 변경은 관리자만 가능 (#175).
//
// 규칙:
//   - 관리자는 같은 강의실 안의 다른 채널로 게시글을 이동시킬 수 있다.
//   - 일반 작성자(학생)는 자기 글이라도 채널을 변경할 수 없다 (본문/태그 수정은 여전히 가능).
//   - 존재하지 않는 채널 / 다른 강의실 채널로는 이동할 수 없다.
//   - channel_id 를 아예 보내지 않으면 채널은 그대로 유지된다 (하위 호환).
func TestAdminPostCategoryEdit(t *testing.T) {
	ts := setupTestServer(t)
	adminToken := ts.login(testAdminEmail, testAdminPass)

	cr := ts.createClassroom(adminToken, "카테고리테스트반", 1000000)
	other := ts.createClassroom(adminToken, "다른강의실반", 1000000)

	// 채널 조회 헬퍼 — slug 로 채널 ID 찾기.
	channelsOf := func(classroomID int) map[string]int {
		r := ts.get(fmt.Sprintf("/api/classrooms/%d/channels", classroomID), adminToken)
		if !r.Success {
			t.Fatalf("get channels: %v", r.Error)
		}
		var chs []struct {
			ID   int    `json:"id"`
			Slug string `json:"slug"`
		}
		json.Unmarshal(r.Data, &chs)
		m := make(map[string]int)
		for _, ch := range chs {
			m[ch.Slug] = ch.ID
		}
		return m
	}

	crCh := channelsOf(cr.ID)
	otherCh := channelsOf(other.ID)
	freeCh := crCh["free"]         // 학생 작성 가능 (write_role all) — 원본 채널
	showcaseCh := crCh["showcase"] // 같은 강의실 다른 채널 (write_role all) — 이동 대상
	otherFreeCh := otherCh["free"] // 다른 강의실 채널 — 경계 위반 테스트용
	if freeCh == 0 || showcaseCh == 0 || otherFreeCh == 0 {
		t.Fatalf("expected seeded channels, got cr=%v other=%v", crCh, otherCh)
	}

	authorToken := ts.registerAndApprove("catedit-author@test.com", "pass1234", "카테고리작성자", "20270101")
	ts.joinClassroom(authorToken, cr.Code)

	// 학생이 free 채널에 글 작성 후 그 postID 반환.
	createPost := func(content string) int {
		r := ts.post(fmt.Sprintf("/api/channels/%d/posts", freeCh), map[string]interface{}{
			"content": content,
		}, authorToken)
		if !r.Success {
			t.Fatalf("create post failed: %v", r.Error)
		}
		var p struct {
			ID int `json:"id"`
		}
		json.Unmarshal(r.Data, &p)
		if p.ID == 0 {
			t.Fatalf("expected valid post id, got body %s", string(r.Data))
		}
		return p.ID
	}

	// GET /api/posts/{id} 로 현재 channel_id 재조회.
	currentChannel := func(postID int, token string) int {
		r := ts.get(fmt.Sprintf("/api/posts/%d", postID), token)
		if !r.Success {
			t.Fatalf("refetch post %d failed: %v", postID, r.Error)
		}
		var p struct {
			ChannelID int    `json:"channel_id"`
			Content   string `json:"content"`
		}
		json.Unmarshal(r.Data, &p)
		return p.ChannelID
	}

	t.Run("admin moves student post to another channel in same classroom", func(t *testing.T) {
		postID := createPost("관리자가 이동시킬 학생 글")
		r := ts.put(fmt.Sprintf("/api/posts/%d", postID), map[string]interface{}{
			"content":    "관리자가 이동시킬 학생 글", // 본문은 항상 함께 전송 (수정 폼 동작 반영)
			"channel_id": showcaseCh,
		}, adminToken)
		if !r.Success {
			t.Fatalf("admin move should succeed: %v", r.Error)
		}
		var p struct {
			ChannelID int    `json:"channel_id"`
			Content   string `json:"content"`
		}
		json.Unmarshal(r.Data, &p)
		if p.ChannelID != showcaseCh {
			t.Errorf("expected channel_id %d after move, got %d", showcaseCh, p.ChannelID)
		}
		if p.Content != "관리자가 이동시킬 학생 글" {
			t.Errorf("content should be preserved, got %q", p.Content)
		}
		if got := currentChannel(postID, adminToken); got != showcaseCh {
			t.Errorf("refetch: expected channel %d, got %d", showcaseCh, got)
		}
	})

	t.Run("author cannot change channel of own post", func(t *testing.T) {
		postID := createPost("학생이 채널 바꾸려는 글")
		r := ts.put(fmt.Sprintf("/api/posts/%d", postID), map[string]interface{}{
			"content":    "학생이 채널 바꾸려는 글",
			"channel_id": showcaseCh,
		}, authorToken)
		if r.Success {
			t.Fatalf("author changing channel should fail, but succeeded")
		}
		if got := currentChannel(postID, authorToken); got != freeCh {
			t.Errorf("channel should be unchanged (%d), got %d", freeCh, got)
		}
	})

	t.Run("author can still edit content and tags without channel_id (backward compat)", func(t *testing.T) {
		postID := createPost("원본 본문")
		r := ts.put(fmt.Sprintf("/api/posts/%d", postID), map[string]interface{}{
			"content": "수정된 본문",
			"tags":    `["태그1","태그2"]`,
		}, authorToken)
		if !r.Success {
			t.Fatalf("author content/tags edit should succeed: %v", r.Error)
		}
		var p struct {
			ChannelID int    `json:"channel_id"`
			Content   string `json:"content"`
		}
		json.Unmarshal(r.Data, &p)
		if p.Content != "수정된 본문" {
			t.Errorf("expected updated content, got %q", p.Content)
		}
		if p.ChannelID != freeCh {
			t.Errorf("channel should stay %d, got %d", freeCh, p.ChannelID)
		}
	})

	t.Run("author sending own current channel_id is a no-op (allowed)", func(t *testing.T) {
		postID := createPost("현재채널 재전송 글")
		r := ts.put(fmt.Sprintf("/api/posts/%d", postID), map[string]interface{}{
			"content":    "현재채널 재전송 글 수정",
			"channel_id": freeCh, // 자기 글의 현재 채널 그대로 → 이동 아님, 허용
		}, authorToken)
		if !r.Success {
			t.Fatalf("author resending current channel should succeed: %v", r.Error)
		}
		if got := currentChannel(postID, authorToken); got != freeCh {
			t.Errorf("channel should stay %d, got %d", freeCh, got)
		}
	})

	t.Run("admin cannot move to nonexistent channel", func(t *testing.T) {
		postID := createPost("존재하지 않는 채널로 이동 시도")
		r := ts.put(fmt.Sprintf("/api/posts/%d", postID), map[string]interface{}{
			"content":    "존재하지 않는 채널로 이동 시도",
			"channel_id": 999999,
		}, adminToken)
		if r.Success {
			t.Fatalf("moving to nonexistent channel should fail, but succeeded")
		}
		if got := currentChannel(postID, adminToken); got != freeCh {
			t.Errorf("channel should be unchanged (%d), got %d", freeCh, got)
		}
	})

	t.Run("admin cannot move to channel in a different classroom", func(t *testing.T) {
		postID := createPost("다른 강의실 채널로 이동 시도")
		r := ts.put(fmt.Sprintf("/api/posts/%d", postID), map[string]interface{}{
			"content":    "다른 강의실 채널로 이동 시도",
			"channel_id": otherFreeCh,
		}, adminToken)
		if r.Success {
			t.Fatalf("cross-classroom move should fail, but succeeded")
		}
		if got := currentChannel(postID, adminToken); got != freeCh {
			t.Errorf("channel should be unchanged (%d), got %d", freeCh, got)
		}
	})

	t.Run("admin updates content only (no channel_id key) keeps channel", func(t *testing.T) {
		postID := createPost("본문만 수정")
		r := ts.put(fmt.Sprintf("/api/posts/%d", postID), map[string]interface{}{
			"content": "본문만 수정됨",
		}, adminToken)
		if !r.Success {
			t.Fatalf("content-only update should succeed: %v", r.Error)
		}
		if got := currentChannel(postID, adminToken); got != freeCh {
			t.Errorf("channel should stay %d, got %d", freeCh, got)
		}
	})
}
