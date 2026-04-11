package integration

import (
	"encoding/json"
	"fmt"
	"testing"
)

// TestEngagementReward tests the like/comment reward system.
// Rules:
// - Like: 글쓴이에게 10원, 취소 시 10원 회수
// - Comment: 글쓴이에게 100원, 삭제 시 100원 회수
// - Self like/comment: 보상 없음
func TestEngagementReward(t *testing.T) {
	ts := setupTestServer(t)

	// Setup: admin creates classroom + channel
	adminToken := ts.login(testAdminEmail, testAdminPass)

	crResp := ts.post("/api/classrooms", map[string]interface{}{
		"name": "보상테스트반", "initial_capital": 1000000,
	}, adminToken)
	if !crResp.Success {
		t.Fatalf("create classroom: %v", crResp.Error)
	}
	var cr struct {
		ID   int    `json:"id"`
		Code string `json:"code"`
	}
	json.Unmarshal(crResp.Data, &cr)

	// Get free channel
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

	// Register two students
	authorToken := ts.registerAndApprove("author@test.com", "pass1234", "글쓴이", "2024001")
	readerToken := ts.registerAndApprove("reader@test.com", "pass1234", "독자", "2024002")

	// Both join classroom
	ts.post("/api/classrooms/join", map[string]string{"code": cr.Code}, authorToken)
	ts.post("/api/classrooms/join", map[string]string{"code": cr.Code}, readerToken)

	// Author creates a post
	postResp := ts.post(fmt.Sprintf("/api/channels/%d/posts", channelID), map[string]interface{}{
		"content": "보상 테스트 게시글입니다.",
	}, authorToken)
	if !postResp.Success {
		t.Fatalf("create post: %v", postResp.Error)
	}
	var postData struct {
		ID int `json:"id"`
	}
	json.Unmarshal(postResp.Data, &postData)
	postID := postData.ID

	// Helper: get wallet balance
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

	// Helper: get transactions
	getTransactions := func(token string, txType string) []struct {
		Amount      int    `json:"amount"`
		TxType      string `json:"tx_type"`
		Description string `json:"description"`
	} {
		path := "/api/wallet/transactions?page=1&limit=50"
		if txType != "" {
			path += "&tx_type=" + txType
		}
		r := ts.get(path, token)
		if !r.Success {
			return nil
		}
		var data struct {
			Data []struct {
				Amount      int    `json:"amount"`
				TxType      string `json:"tx_type"`
				Description string `json:"description"`
			} `json:"data"`
		}
		json.Unmarshal(r.Data, &data)
		return data.Data
	}

	authorBalanceBefore := getBalance(authorToken)

	t.Run("Like by other user rewards author 10원", func(t *testing.T) {
		r := ts.post(fmt.Sprintf("/api/posts/%d/like", postID), nil, readerToken)
		if !r.Success {
			t.Fatalf("like failed: %v", r.Error)
		}
		var data struct {
			Liked  bool `json:"liked"`
			Reward int  `json:"reward"`
		}
		json.Unmarshal(r.Data, &data)

		if !data.Liked {
			t.Error("expected liked=true")
		}
		if data.Reward != 10 {
			t.Errorf("expected reward=10, got %d", data.Reward)
		}

		// Check author balance increased by 10
		newBalance := getBalance(authorToken)
		if newBalance != authorBalanceBefore+10 {
			t.Errorf("expected author balance %d, got %d", authorBalanceBefore+10, newBalance)
		}

		// Check transaction record
		txs := getTransactions(authorToken, "like_reward")
		if len(txs) == 0 {
			t.Fatal("expected like_reward transaction")
		}
		if txs[0].Amount != 10 {
			t.Errorf("expected tx amount 10, got %d", txs[0].Amount)
		}
	})

	t.Run("Unlike deducts 10원 from author", func(t *testing.T) {
		balanceBefore := getBalance(authorToken)

		// Toggle like again (unlike)
		r := ts.post(fmt.Sprintf("/api/posts/%d/like", postID), nil, readerToken)
		if !r.Success {
			t.Fatalf("unlike failed: %v", r.Error)
		}
		var data struct {
			Liked  bool `json:"liked"`
			Reward int  `json:"reward"`
		}
		json.Unmarshal(r.Data, &data)

		if data.Liked {
			t.Error("expected liked=false (unlike)")
		}
		if data.Reward != -10 {
			t.Errorf("expected reward=-10, got %d", data.Reward)
		}

		newBalance := getBalance(authorToken)
		if newBalance != balanceBefore-10 {
			t.Errorf("expected balance %d, got %d", balanceBefore-10, newBalance)
		}
	})

	t.Run("Self-like gives no reward", func(t *testing.T) {
		balanceBefore := getBalance(authorToken)

		r := ts.post(fmt.Sprintf("/api/posts/%d/like", postID), nil, authorToken)
		if !r.Success {
			t.Fatalf("self-like failed: %v", r.Error)
		}
		var data struct {
			Liked  bool `json:"liked"`
			Reward int  `json:"reward"`
		}
		json.Unmarshal(r.Data, &data)

		if data.Reward != 0 {
			t.Errorf("self-like should give reward=0, got %d", data.Reward)
		}

		newBalance := getBalance(authorToken)
		if newBalance != balanceBefore {
			t.Errorf("self-like should not change balance: expected %d, got %d", balanceBefore, newBalance)
		}

		// Cleanup: unlike
		ts.post(fmt.Sprintf("/api/posts/%d/like", postID), nil, authorToken)
	})

	t.Run("Comment by other user rewards author 100원", func(t *testing.T) {
		balanceBefore := getBalance(authorToken)

		r := ts.post(fmt.Sprintf("/api/posts/%d/comments", postID), map[string]string{
			"content": "좋은 글이네요!",
		}, readerToken)
		if !r.Success {
			t.Fatalf("create comment failed: %v", r.Error)
		}

		newBalance := getBalance(authorToken)
		if newBalance != balanceBefore+100 {
			t.Errorf("expected author balance %d, got %d", balanceBefore+100, newBalance)
		}

		// Check transaction
		txs := getTransactions(authorToken, "comment_reward")
		if len(txs) == 0 {
			t.Fatal("expected comment_reward transaction")
		}
		if txs[0].Amount != 100 {
			t.Errorf("expected tx amount 100, got %d", txs[0].Amount)
		}
	})

	t.Run("Self-comment gives no reward", func(t *testing.T) {
		balanceBefore := getBalance(authorToken)

		r := ts.post(fmt.Sprintf("/api/posts/%d/comments", postID), map[string]string{
			"content": "내 글에 내가 댓글",
		}, authorToken)
		if !r.Success {
			t.Fatalf("self-comment failed: %v", r.Error)
		}

		newBalance := getBalance(authorToken)
		if newBalance != balanceBefore {
			t.Errorf("self-comment should not change balance: expected %d, got %d", balanceBefore, newBalance)
		}
	})

	t.Run("Delete comment deducts 100원 from author", func(t *testing.T) {
		// Reader writes another comment
		r := ts.post(fmt.Sprintf("/api/posts/%d/comments", postID), map[string]string{
			"content": "삭제할 댓글입니다",
		}, readerToken)
		if !r.Success {
			t.Fatalf("create comment: %v", r.Error)
		}
		var comment struct {
			ID int `json:"id"`
		}
		json.Unmarshal(r.Data, &comment)

		balanceBefore := getBalance(authorToken)

		// Reader deletes own comment
		dr := ts.delete(fmt.Sprintf("/api/posts/%d/comments/%d", postID, comment.ID), readerToken)
		if !dr.Success {
			t.Fatalf("delete comment failed: %v", dr.Error)
		}

		newBalance := getBalance(authorToken)
		if newBalance != balanceBefore-100 {
			t.Errorf("expected balance %d after comment delete, got %d", balanceBefore-100, newBalance)
		}
	})

	t.Run("Cannot delete other user's comment", func(t *testing.T) {
		// Author writes a comment
		r := ts.post(fmt.Sprintf("/api/posts/%d/comments", postID), map[string]string{
			"content": "작성자의 댓글",
		}, authorToken)
		if !r.Success {
			t.Fatalf("create comment: %v", r.Error)
		}
		var comment struct {
			ID int `json:"id"`
		}
		json.Unmarshal(r.Data, &comment)

		// Reader tries to delete author's comment
		dr := ts.delete(fmt.Sprintf("/api/posts/%d/comments/%d", postID, comment.ID), readerToken)
		if dr.Success {
			t.Error("should not be able to delete other user's comment")
		}
	})

	t.Run("Delete self-comment on own post does not deduct", func(t *testing.T) {
		// Author writes comment on own post (no reward was given)
		r := ts.post(fmt.Sprintf("/api/posts/%d/comments", postID), map[string]string{
			"content": "내가 쓴 내 댓글 - 삭제 예정",
		}, authorToken)
		if !r.Success {
			t.Fatalf("create comment: %v", r.Error)
		}
		var comment struct {
			ID int `json:"id"`
		}
		json.Unmarshal(r.Data, &comment)

		balanceBefore := getBalance(authorToken)

		dr := ts.delete(fmt.Sprintf("/api/posts/%d/comments/%d", postID, comment.ID), authorToken)
		if !dr.Success {
			t.Fatalf("delete own comment failed: %v", dr.Error)
		}

		newBalance := getBalance(authorToken)
		if newBalance != balanceBefore {
			t.Errorf("deleting self-comment on own post should not change balance: expected %d, got %d", balanceBefore, newBalance)
		}
	})
}
