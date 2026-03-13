package integration

import (
	"encoding/json"
	"fmt"
	"testing"
)

// TestComments_RegressionSuite tests the comment feature end-to-end.
// Regression: comments returned raw arrays instead of PaginatedData format,
// CreateComment didn't set PostID from URL param, and created_at was zero.
func TestComments_RegressionSuite(t *testing.T) {
	ts := setupTestServer(t)

	// Setup: admin creates classroom + channel, student registers + joins
	adminToken := ts.login(testAdminEmail, testAdminPass)

	// Create classroom
	crResp := ts.post("/api/classrooms", map[string]interface{}{
		"name": "댓글 테스트반", "initial_capital": 1000000,
	}, adminToken)
	if !crResp.Success {
		t.Fatalf("create classroom: %v", crResp.Error)
	}
	var cr struct {
		ID   int    `json:"id"`
		Code string `json:"code"`
	}
	json.Unmarshal(crResp.Data, &cr)

	// Get channels for classroom
	chResp := ts.get(fmt.Sprintf("/api/classrooms/%d/channels", cr.ID), adminToken)
	if !chResp.Success {
		t.Fatalf("get channels: %v", chResp.Error)
	}
	var channels []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
		Slug string `json:"slug"`
	}
	json.Unmarshal(chResp.Data, &channels)
	if len(channels) == 0 {
		t.Fatal("no channels created for classroom")
	}

	// Find 자유 channel (or use first one)
	channelID := channels[0].ID
	for _, ch := range channels {
		if ch.Slug == "free" || ch.Name == "자유" {
			channelID = ch.ID
			break
		}
	}

	// Register + approve a student
	studentToken := ts.registerAndApprove("student-comment@test.com", "pass1234", "댓글학생", "2024010")

	// Student joins classroom
	ts.post("/api/classrooms/join", map[string]string{
		"code": cr.Code,
	}, studentToken)

	// Create a post
	postResp := ts.post(fmt.Sprintf("/api/channels/%d/posts", channelID), map[string]interface{}{
		"title":   "댓글 테스트 게시글",
		"content": "댓글 기능 테스트용 게시글입니다.",
	}, studentToken)
	if !postResp.Success {
		t.Fatalf("create post: %v", postResp.Error)
	}
	var postData struct {
		ID int `json:"id"`
	}
	json.Unmarshal(postResp.Data, &postData)
	postID := postData.ID

	t.Run("GetComments returns PaginatedData format", func(t *testing.T) {
		r := ts.get(fmt.Sprintf("/api/posts/%d/comments?page=1&limit=50", postID), studentToken)
		if !r.Success {
			t.Fatalf("get comments failed: %v", r.Error)
		}

		var data struct {
			Data       json.RawMessage `json:"data"`
			Pagination struct {
				Page       int `json:"page"`
				Limit      int `json:"limit"`
				Total      int `json:"total"`
				TotalPages int `json:"total_pages"`
			} `json:"pagination"`
		}
		if err := json.Unmarshal(r.Data, &data); err != nil {
			t.Fatalf("comments response not PaginatedData format: %v\nraw: %s", err, string(r.Data))
		}
		if data.Pagination.Page != 1 {
			t.Errorf("expected page=1, got %d", data.Pagination.Page)
		}

		// data.Data should be an array
		var comments []json.RawMessage
		if err := json.Unmarshal(data.Data, &comments); err != nil {
			t.Fatalf("data.data is not an array: %v", err)
		}
		if len(comments) != 0 {
			t.Errorf("expected 0 comments, got %d", len(comments))
		}
	})

	t.Run("CreateComment sets PostID from URL param", func(t *testing.T) {
		// Send comment WITHOUT post_id in body — should use URL param
		r := ts.post(fmt.Sprintf("/api/posts/%d/comments", postID), map[string]string{
			"content": "테스트 댓글입니다!",
		}, studentToken)
		if !r.Success {
			t.Fatalf("create comment failed: %v", r.Error)
		}

		var comment struct {
			ID        int    `json:"id"`
			PostID    int    `json:"post_id"`
			Content   string `json:"content"`
			CreatedAt string `json:"created_at"`
			Author    struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			} `json:"author"`
		}
		if err := json.Unmarshal(r.Data, &comment); err != nil {
			t.Fatalf("parse comment: %v", err)
		}
		if comment.ID == 0 {
			t.Error("comment ID should not be 0")
		}
		if comment.PostID != postID {
			t.Errorf("expected post_id=%d, got %d", postID, comment.PostID)
		}
	})

	t.Run("CreateComment returns valid created_at", func(t *testing.T) {
		r := ts.post(fmt.Sprintf("/api/posts/%d/comments", postID), map[string]string{
			"content": "날짜 테스트 댓글",
		}, studentToken)
		if !r.Success {
			t.Fatalf("create comment: %v", r.Error)
		}

		var comment struct {
			CreatedAt string `json:"created_at"`
		}
		json.Unmarshal(r.Data, &comment)

		if comment.CreatedAt == "" || comment.CreatedAt == "0001-01-01T00:00:00Z" {
			t.Errorf("created_at is empty or zero: %s", comment.CreatedAt)
		}
	})

	t.Run("CreateComment returns author as nested object", func(t *testing.T) {
		r := ts.post(fmt.Sprintf("/api/posts/%d/comments", postID), map[string]string{
			"content": "작성자 테스트 댓글",
		}, studentToken)
		if !r.Success {
			t.Fatalf("create comment: %v", r.Error)
		}

		// Check that author is an object with name, not flat author_name
		var raw map[string]interface{}
		json.Unmarshal(r.Data, &raw)

		author, ok := raw["author"]
		if !ok {
			t.Fatal("comment response missing 'author' field")
		}
		authorMap, ok := author.(map[string]interface{})
		if !ok {
			t.Fatalf("'author' is not an object: %T", author)
		}
		if _, ok := authorMap["name"]; !ok {
			t.Error("author object missing 'name' field")
		}
	})

	t.Run("GetComments returns created comments", func(t *testing.T) {
		r := ts.get(fmt.Sprintf("/api/posts/%d/comments?page=1&limit=50", postID), studentToken)
		if !r.Success {
			t.Fatalf("get comments: %v", r.Error)
		}

		var data struct {
			Data []struct {
				ID      int `json:"id"`
				Author  struct {
					Name string `json:"name"`
				} `json:"author"`
				CreatedAt string `json:"created_at"`
			} `json:"data"`
			Pagination struct {
				Total int `json:"total"`
			} `json:"pagination"`
		}
		json.Unmarshal(r.Data, &data)

		if len(data.Data) < 3 {
			t.Errorf("expected ≥3 comments, got %d", len(data.Data))
		}
		if data.Pagination.Total < 3 {
			t.Errorf("expected total ≥3, got %d", data.Pagination.Total)
		}

		// All comments should have valid author and timestamp
		for i, c := range data.Data {
			if c.Author.Name == "" {
				t.Errorf("comment[%d] missing author.name", i)
			}
			if c.CreatedAt == "" || c.CreatedAt == "0001-01-01T00:00:00Z" {
				t.Errorf("comment[%d] has invalid created_at: %s", i, c.CreatedAt)
			}
		}
	})
}
