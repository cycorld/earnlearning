package integration

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestDM(t *testing.T) {
	ts := setupTestServer(t)
	adminToken := ts.login(testAdminEmail, testAdminPass)

	// Create classroom + 2 students
	crResp := ts.post("/api/classrooms", map[string]interface{}{
		"name": "DM테스트반", "initial_capital": 1000000,
	}, adminToken)
	var cr struct {
		Code string `json:"code"`
	}
	json.Unmarshal(crResp.Data, &cr)

	student1Token := ts.registerAndApprove("dm-s1@test.com", "pass1234", "학생1", "2024030")
	student2Token := ts.registerAndApprove("dm-s2@test.com", "pass1234", "학생2", "2024031")
	ts.post("/api/classrooms/join", map[string]string{"code": cr.Code}, student1Token)
	ts.post("/api/classrooms/join", map[string]string{"code": cr.Code}, student2Token)

	t.Run("메시지를 전송하면 성공한다", func(t *testing.T) {
		r := ts.post("/api/dm/messages", map[string]interface{}{
			"receiver_id": 3, // student2
			"content":     "안녕하세요!",
		}, student1Token)
		if !r.Success {
			t.Fatalf("send message failed: %v", r.Error)
		}
		var msg struct {
			ID        int    `json:"id"`
			SenderID  int    `json:"sender_id"`
			Content   string `json:"content"`
			CreatedAt string `json:"created_at"`
		}
		json.Unmarshal(r.Data, &msg)
		if msg.ID == 0 {
			t.Error("message ID should not be 0")
		}
		if msg.Content != "안녕하세요!" {
			t.Errorf("expected content '안녕하세요!', got '%s'", msg.Content)
		}
	})

	t.Run("자기 자신에게 메시지를 보낼 수 없다", func(t *testing.T) {
		r := ts.post("/api/dm/messages", map[string]interface{}{
			"receiver_id": 2, // student1 자신
			"content":     "셀프 DM",
		}, student1Token)
		if r.Success {
			t.Error("should not allow sending DM to self")
		}
	})

	t.Run("빈 메시지를 보낼 수 없다", func(t *testing.T) {
		r := ts.post("/api/dm/messages", map[string]interface{}{
			"receiver_id": 3,
			"content":     "",
		}, student1Token)
		if r.Success {
			t.Error("should not allow empty message")
		}
	})

	t.Run("메시지 조회가 가능하다", func(t *testing.T) {
		// Send a couple more messages
		ts.post("/api/dm/messages", map[string]interface{}{
			"receiver_id": 2, "content": "학생1에게 답장!",
		}, student2Token)
		ts.post("/api/dm/messages", map[string]interface{}{
			"receiver_id": 3, "content": "두번째 메시지",
		}, student1Token)

		r := ts.get("/api/dm/messages/3?limit=10", student1Token)
		if !r.Success {
			t.Fatalf("get messages failed: %v", r.Error)
		}
		var messages []struct {
			Content string `json:"content"`
		}
		json.Unmarshal(r.Data, &messages)
		if len(messages) < 2 {
			t.Errorf("expected ≥2 messages, got %d", len(messages))
		}
	})

	t.Run("대화 목록을 조회할 수 있다", func(t *testing.T) {
		r := ts.get("/api/dm/conversations", student1Token)
		if !r.Success {
			t.Fatalf("get conversations failed: %v", r.Error)
		}
		var convs []struct {
			PeerID      int    `json:"peer_id"`
			PeerName    string `json:"peer_name"`
			LastMessage string `json:"last_message"`
			UnreadCount int    `json:"unread_count"`
		}
		json.Unmarshal(r.Data, &convs)
		if len(convs) == 0 {
			t.Error("expected at least 1 conversation")
		}
		if convs[0].PeerName == "" {
			t.Error("peer_name should not be empty")
		}
	})

	t.Run("미읽음 수를 조회할 수 있다", func(t *testing.T) {
		r := ts.get("/api/dm/unread-count", student1Token)
		if !r.Success {
			t.Fatalf("get unread count failed: %v", r.Error)
		}
		var data struct {
			UnreadCount int `json:"unread_count"`
		}
		json.Unmarshal(r.Data, &data)
		if data.UnreadCount == 0 {
			t.Error("student1 should have unread messages from student2")
		}
	})

	t.Run("읽음 처리가 동작한다", func(t *testing.T) {
		r := ts.put(fmt.Sprintf("/api/dm/messages/%d/read", 3), nil, student1Token)
		if !r.Success {
			t.Fatalf("mark as read failed: %v", r.Error)
		}

		// Verify unread count is 0 for student2's messages
		ur := ts.get("/api/dm/unread-count", student1Token)
		var data struct {
			UnreadCount int `json:"unread_count"`
		}
		json.Unmarshal(ur.Data, &data)
		// student1 received 1 message from student2, now marked as read
		// But student2 also has a conversation — check student1's unread
		if data.UnreadCount != 0 {
			t.Errorf("expected unread_count=0 after mark as read, got %d", data.UnreadCount)
		}
	})
}
