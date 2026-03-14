package integration

import (
	"encoding/json"
	"strconv"
	"testing"
	"time"
)

// TestTimeline_ContentDisplay verifies that the home timeline (GET /api/posts)
// returns posts with all expected fields correctly populated, including
// author_name, content, timestamps, and pagination metadata.
func TestTimeline_ContentDisplay(t *testing.T) {
	ts := setupTestServer(t)

	// Setup: admin creates a classroom → gets channels → posts to "자유" channel
	adminToken := ts.login(testAdminEmail, testAdminPass)

	// Create classroom
	classroomResp := ts.post("/api/classrooms", map[string]interface{}{
		"name": "테스트 교실",
	}, adminToken)
	if !classroomResp.Success {
		t.Fatalf("create classroom failed: %s", string(classroomResp.Data))
	}
	var classroom struct {
		ID   int    `json:"id"`
		Code string `json:"code"`
	}
	json.Unmarshal(classroomResp.Data, &classroom)

	// Get channels for the classroom
	channelsResp := ts.get("/api/classrooms/"+itoa(classroom.ID)+"/channels", adminToken)
	if !channelsResp.Success {
		t.Fatalf("get channels failed: %s", string(channelsResp.Data))
	}
	var channels []struct {
		ID          int    `json:"id"`
		Slug        string `json:"slug"`
		ChannelType string `json:"channel_type"`
	}
	json.Unmarshal(channelsResp.Data, &channels)

	// Find the "free" channel (자유)
	var freeChannelID int
	for _, ch := range channels {
		if ch.Slug == "free" {
			freeChannelID = ch.ID
			break
		}
	}
	if freeChannelID == 0 {
		t.Fatal("free channel not found in classroom")
	}

	// Register a regular user, approve, and join the classroom
	userToken := ts.registerAndApprove("student@test.com", "pass1234", "김학생", "2026000001")
	ts.post("/api/classrooms/join", map[string]string{"code": classroom.Code}, userToken)

	t.Run("admin creates post and author_name is displayed", func(t *testing.T) {
		// Admin creates a post
		createResp := ts.post("/api/channels/"+itoa(freeChannelID)+"/posts", map[string]interface{}{
			"content": "첫 번째 공지입니다 #테스트",
		}, adminToken)
		if !createResp.Success {
			t.Fatalf("create post failed: %s", string(createResp.Data))
		}

		// Fetch posts via timeline
		postsResp := ts.get("/api/posts?channel_id="+itoa(freeChannelID), userToken)
		if !postsResp.Success {
			t.Fatalf("get posts failed: %s", string(postsResp.Data))
		}

		var result struct {
			Data []struct {
				ID           int    `json:"id"`
				ChannelID    int    `json:"channel_id"`
				AuthorID     int    `json:"author_id"`
				AuthorName   string `json:"author_name"`
				AuthorAvatar string `json:"author_avatar"`
				Content      string `json:"content"`
				PostType     string `json:"post_type"`
				LikeCount    int    `json:"like_count"`
				CommentCount int    `json:"comment_count"`
				IsLiked      bool   `json:"is_liked"`
				CreatedAt    string `json:"created_at"`
				UpdatedAt    string `json:"updated_at"`
				Tags         string `json:"tags"`
			} `json:"data"`
			Pagination struct {
				Page       int `json:"page"`
				Limit      int `json:"limit"`
				Total      int `json:"total"`
				TotalPages int `json:"total_pages"`
			} `json:"pagination"`
		}
		json.Unmarshal(postsResp.Data, &result)

		if len(result.Data) == 0 {
			t.Fatal("expected at least 1 post, got 0")
		}

		post := result.Data[0]

		// author_name must be the admin's seeded name
		if post.AuthorName == "" {
			t.Error("author_name is empty, expected admin's name")
		}
		if post.AuthorName != "최용철" {
			t.Errorf("author_name = %q, want %q", post.AuthorName, "최용철")
		}

		// content must match
		if post.Content != "첫 번째 공지입니다 #테스트" {
			t.Errorf("content = %q, want %q", post.Content, "첫 번째 공지입니다 #테스트")
		}

		// post_type defaults to "normal"
		if post.PostType != "normal" {
			t.Errorf("post_type = %q, want %q", post.PostType, "normal")
		}

		// created_at must be a valid, non-zero timestamp
		if post.CreatedAt == "" || post.CreatedAt == "0001-01-01T00:00:00Z" {
			t.Errorf("created_at is zero or empty: %q", post.CreatedAt)
		}
		parsed, err := time.Parse(time.RFC3339, post.CreatedAt)
		if err != nil {
			t.Errorf("created_at is not valid RFC3339: %q, err: %v", post.CreatedAt, err)
		}
		if time.Since(parsed) > 10*time.Second {
			t.Errorf("created_at seems too old: %v", parsed)
		}

		// channel_id must match
		if post.ChannelID != freeChannelID {
			t.Errorf("channel_id = %d, want %d", post.ChannelID, freeChannelID)
		}

		// tags should contain auto-extracted "테스트"
		if post.Tags == "" || post.Tags == "[]" {
			t.Errorf("tags should contain auto-extracted tag, got: %s", post.Tags)
		}
	})

	t.Run("student creates post and their name is displayed", func(t *testing.T) {
		createResp := ts.post("/api/channels/"+itoa(freeChannelID)+"/posts", map[string]interface{}{
			"content": "학생 게시글입니다",
		}, userToken)
		if !createResp.Success {
			t.Fatalf("create post failed: %s", string(createResp.Data))
		}

		postsResp := ts.get("/api/posts?channel_id="+itoa(freeChannelID), userToken)
		if !postsResp.Success {
			t.Fatalf("get posts failed: %s", string(postsResp.Data))
		}

		var result struct {
			Data []struct {
				AuthorName string `json:"author_name"`
				Content    string `json:"content"`
			} `json:"data"`
		}
		json.Unmarshal(postsResp.Data, &result)

		// Find the student's post (ordered by created_at DESC, so it's first)
		found := false
		for _, p := range result.Data {
			if p.Content == "학생 게시글입니다" {
				found = true
				if p.AuthorName != "김학생" {
					t.Errorf("student post author_name = %q, want %q", p.AuthorName, "김학생")
				}
			}
		}
		if !found {
			t.Error("student's post not found in timeline")
		}
	})

	t.Run("pagination metadata is correct", func(t *testing.T) {
		postsResp := ts.get("/api/posts?channel_id="+itoa(freeChannelID)+"&limit=1&page=1", userToken)
		if !postsResp.Success {
			t.Fatalf("get posts failed: %s", string(postsResp.Data))
		}

		var result struct {
			Data       []json.RawMessage `json:"data"`
			Pagination struct {
				Page       int `json:"page"`
				Limit      int `json:"limit"`
				Total      int `json:"total"`
				TotalPages int `json:"total_pages"`
			} `json:"pagination"`
		}
		json.Unmarshal(postsResp.Data, &result)

		if result.Pagination.Page != 1 {
			t.Errorf("pagination.page = %d, want 1", result.Pagination.Page)
		}
		if result.Pagination.Limit != 1 {
			t.Errorf("pagination.limit = %d, want 1", result.Pagination.Limit)
		}
		if result.Pagination.Total < 2 {
			t.Errorf("pagination.total = %d, want >= 2", result.Pagination.Total)
		}
		if result.Pagination.TotalPages < 2 {
			t.Errorf("pagination.total_pages = %d, want >= 2", result.Pagination.TotalPages)
		}
		if len(result.Data) != 1 {
			t.Errorf("returned %d posts, want 1 (limit=1)", len(result.Data))
		}
	})

	t.Run("like count and is_liked reflect user action", func(t *testing.T) {
		// Get first post ID
		postsResp := ts.get("/api/posts?channel_id="+itoa(freeChannelID), userToken)
		var result struct {
			Data []struct {
				ID int `json:"id"`
			} `json:"data"`
		}
		json.Unmarshal(postsResp.Data, &result)
		if len(result.Data) == 0 {
			t.Fatal("no posts to like")
		}
		postID := result.Data[0].ID

		// Like the post
		likeResp := ts.post("/api/posts/"+itoa(postID)+"/like", nil, userToken)
		if !likeResp.Success {
			t.Fatalf("like failed: %s", string(likeResp.Data))
		}

		// Fetch again — is_liked should be true, like_count should be 1
		postsResp2 := ts.get("/api/posts?channel_id="+itoa(freeChannelID), userToken)
		var result2 struct {
			Data []struct {
				ID        int  `json:"id"`
				LikeCount int  `json:"like_count"`
				IsLiked   bool `json:"is_liked"`
			} `json:"data"`
		}
		json.Unmarshal(postsResp2.Data, &result2)

		for _, p := range result2.Data {
			if p.ID == postID {
				if !p.IsLiked {
					t.Error("is_liked should be true after liking")
				}
				if p.LikeCount < 1 {
					t.Errorf("like_count = %d, want >= 1", p.LikeCount)
				}
				return
			}
		}
		t.Errorf("liked post %d not found in results", postID)
	})

	t.Run("empty channel returns empty array not null", func(t *testing.T) {
		// Find a channel with no posts (e.g., showcase)
		var showcaseChannelID int
		for _, ch := range channels {
			if ch.Slug == "showcase" {
				showcaseChannelID = ch.ID
				break
			}
		}
		if showcaseChannelID == 0 {
			t.Skip("showcase channel not found")
		}

		postsResp := ts.get("/api/posts?channel_id="+itoa(showcaseChannelID), userToken)
		if !postsResp.Success {
			t.Fatalf("get posts failed: %s", string(postsResp.Data))
		}

		var result struct {
			Data []json.RawMessage `json:"data"`
		}
		json.Unmarshal(postsResp.Data, &result)

		if result.Data == nil {
			t.Error("data should be empty array [], not null")
		}
	})
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
