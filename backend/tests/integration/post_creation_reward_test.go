package integration

import (
	"encoding/json"
	"fmt"
	"testing"
)

// TestPostCreationReward tests the post-creation reward system (#123).
// Rules:
// - 게시글 작성: 작성자에게 +10,000원 (post_reward tx)
// - 본인 글 삭제: 작성자에게서 -10,000원 회수
// - 관리자가 학생 글 삭제: 작성자에게서 -10,000원 회수 (어뷰징 방지)
// - 과제 생성: 게시글 작성 보상 없음 (CreateAssignment 경로, 제외)
func TestPostCreationReward(t *testing.T) {
	ts := setupTestServer(t)

	adminToken := ts.login(testAdminEmail, testAdminPass)

	crResp := ts.post("/api/classrooms", map[string]interface{}{
		"name": "작성보상테스트반", "initial_capital": 1000000,
	}, adminToken)
	if !crResp.Success {
		t.Fatalf("create classroom: %v", crResp.Error)
	}
	var cr struct {
		ID   int    `json:"id"`
		Code string `json:"code"`
	}
	json.Unmarshal(crResp.Data, &cr)

	chResp := ts.get(fmt.Sprintf("/api/classrooms/%d/channels", cr.ID), adminToken)
	var channels []struct {
		ID   int    `json:"id"`
		Slug string `json:"slug"`
	}
	json.Unmarshal(chResp.Data, &channels)
	channelID := channels[0].ID
	for _, ch := range channels {
		if ch.Slug == "free" {
			channelID = ch.ID
			break
		}
	}

	authorToken := ts.registerAndApprove("postauthor@test.com", "pass1234", "글작성자", "2024010")
	ts.post("/api/classrooms/join", map[string]string{"code": cr.Code}, authorToken)

	getBalance := func(token string) int {
		r := ts.get("/api/wallet", token)
		if !r.Success {
			return 0
		}
		var w struct {
			Wallet struct {
				Balance int `json:"balance"`
			} `json:"wallet"`
		}
		json.Unmarshal(r.Data, &w)
		return w.Wallet.Balance
	}

	createPost := func(token, content string) int {
		r := ts.post(fmt.Sprintf("/api/channels/%d/posts", channelID), map[string]interface{}{
			"content": content,
		}, token)
		if !r.Success {
			t.Fatalf("create post failed: %v", r.Error)
		}
		var p struct {
			ID int `json:"id"`
		}
		json.Unmarshal(r.Data, &p)
		return p.ID
	}

	t.Run("Creating a post rewards author 10000원", func(t *testing.T) {
		before := getBalance(authorToken)
		postID := createPost(authorToken, "첫 게시글 작성 보상 테스트")
		if postID == 0 {
			t.Fatal("expected valid post id")
		}
		after := getBalance(authorToken)
		if after != before+10000 {
			t.Errorf("expected balance %d, got %d", before+10000, after)
		}

		// Check post_reward transaction recorded
		r := ts.get("/api/wallet/transactions?page=1&limit=50&tx_type=post_reward", authorToken)
		var data struct {
			Data []struct {
				Amount int    `json:"amount"`
				TxType string `json:"tx_type"`
			} `json:"data"`
		}
		json.Unmarshal(r.Data, &data)
		if len(data.Data) == 0 {
			t.Fatal("expected post_reward transaction")
		}
		if data.Data[0].Amount != 10000 {
			t.Errorf("expected tx amount 10000, got %d", data.Data[0].Amount)
		}
	})

	t.Run("Deleting own post reclaims 10000원", func(t *testing.T) {
		postID := createPost(authorToken, "삭제할 게시글")
		before := getBalance(authorToken)
		dr := ts.delete(fmt.Sprintf("/api/posts/%d", postID), authorToken)
		if !dr.Success {
			t.Fatalf("delete post failed: %v", dr.Error)
		}
		after := getBalance(authorToken)
		if after != before-10000 {
			t.Errorf("expected balance %d after delete, got %d", before-10000, after)
		}
	})

	t.Run("Admin deleting student post reclaims 10000원 from author", func(t *testing.T) {
		postID := createPost(authorToken, "관리자가 삭제할 게시글")
		before := getBalance(authorToken)
		dr := ts.delete(fmt.Sprintf("/api/posts/%d", postID), adminToken)
		if !dr.Success {
			t.Fatalf("admin delete post failed: %v", dr.Error)
		}
		after := getBalance(authorToken)
		if after != before-10000 {
			t.Errorf("expected author balance %d after admin delete, got %d", before-10000, after)
		}
	})

	t.Run("Creating an assignment gives NO post reward", func(t *testing.T) {
		before := getBalance(authorToken)
		r := ts.post(fmt.Sprintf("/api/channels/%d/assignments", channelID), map[string]interface{}{
			"channel_id":    channelID,
			"content":       "과제입니다",
			"reward_amount": 0,
			"max_score":     100,
		}, authorToken)
		if !r.Success {
			t.Fatalf("create assignment failed: %v", r.Error)
		}
		after := getBalance(authorToken)
		if after != before {
			t.Errorf("assignment creation should not give post reward: before %d after %d", before, after)
		}
	})
}
