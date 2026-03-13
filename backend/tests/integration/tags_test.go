package integration

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestTags(t *testing.T) {
	ts := setupTestServer(t)

	// Setup: create user, classroom, join, get writable channel
	token := ts.registerAndApprove("tagger@test.com", "pass1234", "태그유저", "2026000001")
	adminToken := ts.login(testAdminEmail, testAdminPass)

	// Create classroom
	classResp := ts.post("/api/classrooms", map[string]interface{}{
		"name":            "태그테스트",
		"initial_capital": 10000000,
	}, adminToken)
	if !classResp.Success {
		t.Fatalf("create classroom: %v", classResp.Error)
	}
	var classroom struct {
		ID   int    `json:"id"`
		Code string `json:"code"`
	}
	json.Unmarshal(classResp.Data, &classroom)

	// Join classroom
	ts.post("/api/classrooms/join", map[string]string{"code": classroom.Code}, token)

	// Get channels - find writable one (free channel)
	chResp := ts.get(fmt.Sprintf("/api/classrooms/%d/channels", classroom.ID), token)
	var channels []struct {
		ID        int    `json:"id"`
		Slug      string `json:"slug"`
		WriteRole string `json:"write_role"`
	}
	json.Unmarshal(chResp.Data, &channels)

	var freeChannelID int
	for _, ch := range channels {
		if ch.Slug == "free" {
			freeChannelID = ch.ID
			break
		}
	}
	if freeChannelID == 0 {
		t.Fatalf("free channel not found")
	}

	t.Run("사용자 입력 태그가 포스트에 저장된다", func(t *testing.T) {
		resp := ts.post(fmt.Sprintf("/api/channels/%d/posts", freeChannelID), map[string]interface{}{
			"content":   "태그 테스트 포스트입니다",
			"post_type": "normal",
			"tags":      `["공지","중요"]`,
		}, token)
		if !resp.Success {
			t.Fatalf("create post failed: %v", resp.Error)
		}
		var post map[string]interface{}
		json.Unmarshal(resp.Data, &post)

		tags := post["tags"].(string)
		var tagList []string
		json.Unmarshal([]byte(tags), &tagList)

		if len(tagList) < 2 {
			t.Fatalf("expected at least 2 tags, got %d: %v", len(tagList), tagList)
		}
		found := map[string]bool{}
		for _, tag := range tagList {
			found[tag] = true
		}
		if !found["공지"] || !found["중요"] {
			t.Fatalf("expected tags to contain 공지 and 중요, got %v", tagList)
		}
	})

	t.Run("본문 해시태그가 자동 추출된다", func(t *testing.T) {
		resp := ts.post(fmt.Sprintf("/api/channels/%d/posts", freeChannelID), map[string]interface{}{
			"content":   "이것은 #프로젝트 관련 #업데이트 입니다",
			"post_type": "normal",
		}, token)
		if !resp.Success {
			t.Fatalf("create post failed: %v", resp.Error)
		}
		var post map[string]interface{}
		json.Unmarshal(resp.Data, &post)

		tags := post["tags"].(string)
		var tagList []string
		json.Unmarshal([]byte(tags), &tagList)

		found := map[string]bool{}
		for _, tag := range tagList {
			found[tag] = true
		}
		if !found["프로젝트"] || !found["업데이트"] {
			t.Fatalf("expected auto-extracted tags 프로젝트 and 업데이트, got %v", tagList)
		}
	})

	t.Run("사용자 태그와 본문 해시태그가 병합된다", func(t *testing.T) {
		resp := ts.post(fmt.Sprintf("/api/channels/%d/posts", freeChannelID), map[string]interface{}{
			"content":   "#자동태그 가 포함된 글",
			"post_type": "normal",
			"tags":      `["수동태그"]`,
		}, token)
		if !resp.Success {
			t.Fatalf("create post failed: %v", resp.Error)
		}
		var post map[string]interface{}
		json.Unmarshal(resp.Data, &post)

		tags := post["tags"].(string)
		var tagList []string
		json.Unmarshal([]byte(tags), &tagList)

		found := map[string]bool{}
		for _, tag := range tagList {
			found[tag] = true
		}
		if !found["수동태그"] || !found["자동태그"] {
			t.Fatalf("expected merged tags 수동태그 and 자동태그, got %v", tagList)
		}
	})

	t.Run("중복 태그는 제거된다", func(t *testing.T) {
		resp := ts.post(fmt.Sprintf("/api/channels/%d/posts", freeChannelID), map[string]interface{}{
			"content":   "#공통 태그가 있는 글",
			"post_type": "normal",
			"tags":      `["공통","다른태그"]`,
		}, token)
		if !resp.Success {
			t.Fatalf("create post failed: %v", resp.Error)
		}
		var post map[string]interface{}
		json.Unmarshal(resp.Data, &post)

		tags := post["tags"].(string)
		var tagList []string
		json.Unmarshal([]byte(tags), &tagList)

		count := 0
		for _, tag := range tagList {
			if tag == "공통" {
				count++
			}
		}
		if count != 1 {
			t.Fatalf("expected 공통 tag once, found %d times in %v", count, tagList)
		}
	})

	t.Run("태그로 포스트 필터링", func(t *testing.T) {
		// Create a post with unique tag
		ts.post(fmt.Sprintf("/api/channels/%d/posts", freeChannelID), map[string]interface{}{
			"content":   "유니크 태그 테스트",
			"post_type": "normal",
			"tags":      `["유니크필터"]`,
		}, token)

		// Filter by tag
		resp := ts.get(fmt.Sprintf("/api/posts?classroom_id=%d&tag=%s&page=1&limit=20", classroom.ID, "유니크필터"), token)
		if !resp.Success {
			t.Fatalf("get posts by tag failed: %v", resp.Error)
		}
		var result struct {
			Data []map[string]interface{} `json:"data"`
		}
		json.Unmarshal(resp.Data, &result)

		if len(result.Data) == 0 {
			t.Fatalf("expected at least 1 post with tag 유니크필터, got 0")
		}

		// Verify the returned post has the tag
		for _, p := range result.Data {
			tags := p["tags"].(string)
			var tagList []string
			json.Unmarshal([]byte(tags), &tagList)
			found := false
			for _, tag := range tagList {
				if tag == "유니크필터" {
					found = true
				}
			}
			if !found {
				t.Fatalf("filtered post missing tag 유니크필터: %v", tagList)
			}
		}
	})

	t.Run("존재하지 않는 태그로 필터링하면 빈 결과", func(t *testing.T) {
		resp := ts.get(fmt.Sprintf("/api/posts?classroom_id=%d&tag=%s&page=1&limit=20", classroom.ID, "존재안함태그xyz"), token)
		if !resp.Success {
			t.Fatalf("get posts by tag failed: %v", resp.Error)
		}
		var result struct {
			Data []map[string]interface{} `json:"data"`
		}
		json.Unmarshal(resp.Data, &result)

		if len(result.Data) != 0 {
			t.Fatalf("expected 0 posts for nonexistent tag, got %d", len(result.Data))
		}
	})

	t.Run("태그 없이 포스트 생성해도 정상", func(t *testing.T) {
		resp := ts.post(fmt.Sprintf("/api/channels/%d/posts", freeChannelID), map[string]interface{}{
			"content":   "태그 없는 일반 포스트",
			"post_type": "normal",
		}, token)
		if !resp.Success {
			t.Fatalf("create post without tags failed: %v", resp.Error)
		}
		var post map[string]interface{}
		json.Unmarshal(resp.Data, &post)

		tags := post["tags"].(string)
		var tagList []string
		json.Unmarshal([]byte(tags), &tagList)

		if len(tagList) != 0 {
			t.Fatalf("expected empty tags for post without tags, got %v", tagList)
		}
	})
}
