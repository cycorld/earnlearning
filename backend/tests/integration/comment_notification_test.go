package integration

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestCommentNotification(t *testing.T) {
	ts := setupTestServer(t)

	// Setup: admin + student
	adminToken := ts.login(testAdminEmail, testAdminPass)
	studentToken := ts.registerAndApprove("commenter@ewha.ac.kr", "password123", "댓글러", "2024050")

	// Admin creates a classroom and a post
	classResp := ts.post("/api/classrooms", map[string]string{"name": "알림테스트반"}, adminToken)
	var classData struct {
		ID int `json:"id"`
	}
	json.Unmarshal(classResp.Data, &classData)

	channelsResp := ts.get("/api/classrooms/"+commentItoa(classData.ID)+"/channels", adminToken)
	var channels []struct {
		ID int `json:"id"`
	}
	json.Unmarshal(channelsResp.Data, &channels)
	channelID := channels[0].ID

	postResp := ts.post("/api/channels/"+commentItoa(channelID)+"/posts", map[string]string{
		"content": "알림 테스트용 게시물",
	}, adminToken)
	var postData struct {
		ID int `json:"id"`
	}
	json.Unmarshal(postResp.Data, &postData)

	// Student joins classroom
	// Get classroom code
	classDetailResp := ts.get("/api/classrooms/"+commentItoa(classData.ID), adminToken)
	var classDetail struct {
		Classroom struct {
			Code string `json:"code"`
		} `json:"classroom"`
	}
	json.Unmarshal(classDetailResp.Data, &classDetail)
	ts.post("/api/classrooms/join", map[string]string{"code": classDetail.Classroom.Code}, studentToken)

	t.Run("댓글 작성 시 게시물 작성자에게 알림", func(t *testing.T) {
		// Student comments on admin's post
		commentResp := ts.post("/api/posts/"+commentItoa(postData.ID)+"/comments", map[string]string{
			"content": "좋은 글이네요!",
		}, studentToken)
		if !commentResp.Success {
			t.Fatalf("comment failed: %v", commentResp.Error)
		}

		// Check admin's notifications
		notifsResp := ts.get("/api/notifications?limit=5", adminToken)
		if !notifsResp.Success {
			t.Fatalf("get notifications failed: %v", notifsResp.Error)
		}

		var notifsData struct {
			Data []struct {
				NotifType     string `json:"notif_type"`
				Title         string `json:"title"`
				Body          string `json:"body"`
				ReferenceType string `json:"reference_type"`
				ReferenceID   int    `json:"reference_id"`
			} `json:"data"`
		}
		json.Unmarshal(notifsResp.Data, &notifsData)

		found := false
		for _, n := range notifsData.Data {
			if n.NotifType == "new_comment" && n.ReferenceID == postData.ID {
				found = true
				if n.ReferenceType != "post" {
					t.Errorf("expected reference_type 'post', got '%s'", n.ReferenceType)
				}
				if n.Body != "좋은 글이네요!" {
					t.Errorf("expected body '좋은 글이네요!', got '%s'", n.Body)
				}
				break
			}
		}
		if !found {
			t.Error("댓글 알림이 게시물 작성자에게 전송되지 않았습니다")
		}
	})

	t.Run("자기 글에 자기가 댓글 달면 알림 없음", func(t *testing.T) {
		// Admin comments on own post
		ts.post("/api/posts/"+commentItoa(postData.ID)+"/comments", map[string]string{
			"content": "본인 댓글",
		}, adminToken)

		notifsResp := ts.get("/api/notifications?limit=10", adminToken)
		var notifsData2 struct {
			Data []struct {
				NotifType string `json:"notif_type"`
				Body      string `json:"body"`
			} `json:"data"`
		}
		json.Unmarshal(notifsResp.Data, &notifsData2)

		for _, n := range notifsData2.Data {
			if n.NotifType == "new_comment" && n.Body == "본인 댓글" {
				t.Error("자기 글에 자기가 단 댓글은 알림이 가면 안 됩니다")
			}
		}
	})
}

func commentItoa(i int) string {
	return fmt.Sprintf("%d", i)
}
