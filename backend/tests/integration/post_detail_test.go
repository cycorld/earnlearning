package integration

import (
	"encoding/json"
	"fmt"
	"testing"
)

// #035 regression: GET /api/posts/:id returns a single post with viewer-specific
// is_liked field. Enables deep-linked /post/:id detail page in the frontend.

func TestPostDetail_Success_ReturnsSinglePostWithAuthor(t *testing.T) {
	ts := setupTestServer(t)

	adminToken := ts.login(testAdminEmail, testAdminPass)

	// Create classroom so default channels (including 'free') exist
	crResp := ts.post("/api/classrooms", map[string]interface{}{
		"name": "상세 테스트반", "initial_capital": 100000,
	}, adminToken)
	var cr struct{ ID int }
	_ = json.Unmarshal(crResp.Data, &cr)

	// Find the 'free' channel id under this classroom
	chResp := ts.get(fmt.Sprintf("/api/classrooms/%d/channels", cr.ID), adminToken)
	var channels []struct {
		ID   int    `json:"id"`
		Slug string `json:"slug"`
	}
	_ = json.Unmarshal(chResp.Data, &channels)
	var freeID int
	for _, c := range channels {
		if c.Slug == "free" {
			freeID = c.ID
			break
		}
	}
	if freeID == 0 {
		t.Fatalf("free channel not found")
	}

	// Approved student joins classroom and writes a post
	studentToken := ts.registerAndApprove("pd-student@test.com", "pass1234", "PD학생", "2024301")
	ts.post("/api/classrooms/join", map[string]string{"code": func() string {
		var code struct{ Code string }
		_ = json.Unmarshal(crResp.Data, &code)
		return code.Code
	}()}, studentToken)

	createResp := ts.post(fmt.Sprintf("/api/channels/%d/posts", freeID), map[string]interface{}{
		"content": "딥링크 대상 포스트 #회귀테스트",
	}, studentToken)
	if !createResp.Success {
		t.Fatalf("create post: %v", createResp.Error)
	}
	var createdPost struct{ ID int }
	_ = json.Unmarshal(createResp.Data, &createdPost)
	if createdPost.ID == 0 {
		t.Fatalf("create post returned no id: %s", string(createResp.Data))
	}

	// GET /api/posts/:id
	r := ts.get(fmt.Sprintf("/api/posts/%d", createdPost.ID), studentToken)
	if !r.Success {
		t.Fatalf("get single post: %v", r.Error)
	}
	var got struct {
		ID      int    `json:"id"`
		Content string `json:"content"`
		Author  struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"author"`
		IsLiked     bool   `json:"is_liked"`
		ChannelID   int    `json:"channel_id"`
		ChannelName string `json:"channel_name"`
	}
	if err := json.Unmarshal(r.Data, &got); err != nil {
		t.Fatalf("unmarshal: %v\nbody: %s", err, string(r.Data))
	}
	if got.ID != createdPost.ID {
		t.Errorf("id: expected %d, got %d", createdPost.ID, got.ID)
	}
	if got.Content != "딥링크 대상 포스트 #회귀테스트" {
		t.Errorf("content mismatch: %q", got.Content)
	}
	if got.Author.Name != "PD학생" {
		t.Errorf("author name: %q (full: %+v)", got.Author.Name, got)
	}
	if got.IsLiked {
		t.Error("expected is_liked=false before like")
	}

	// Like and re-fetch
	if lr := ts.post(fmt.Sprintf("/api/posts/%d/like", createdPost.ID), nil, studentToken); !lr.Success {
		t.Fatalf("like: %v", lr.Error)
	}
	r2 := ts.get(fmt.Sprintf("/api/posts/%d", createdPost.ID), studentToken)
	var got2 struct {
		IsLiked bool `json:"is_liked"`
	}
	_ = json.Unmarshal(r2.Data, &got2)
	if !got2.IsLiked {
		t.Error("expected is_liked=true after like")
	}
}

func TestPostDetail_NotFound_Returns404(t *testing.T) {
	ts := setupTestServer(t)
	token := ts.registerAndApprove("pd-nf@test.com", "pass1234", "pdnf", "2024302")

	r := ts.get("/api/posts/999999", token)
	if r.Success {
		t.Fatal("expected failure for non-existent post id")
	}
}
