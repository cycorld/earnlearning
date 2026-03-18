package handler

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/earnlearning/backend/internal/application"
	"github.com/earnlearning/backend/internal/domain/post"
	"github.com/earnlearning/backend/internal/interfaces/http/middleware"
)

type PostHandler struct {
	uc *application.PostUsecase
}

func NewPostHandler(uc *application.PostUsecase) *PostHandler {
	return &PostHandler{uc: uc}
}

func (h *PostHandler) GetChannels(c echo.Context) error {
	classroomID, err := strconv.Atoi(c.Param("classroomId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "INVALID_ID", "message": "유효하지 않은 교실 ID입니다"},
		})
	}

	channels, err := h.uc.GetChannels(classroomID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "FETCH_FAILED", "message": err.Error()},
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true, "data": channels, "error": nil,
	})
}

func (h *PostHandler) GetPosts(c echo.Context) error {
	userID := middleware.GetUserID(c)

	classroomID, _ := strconv.Atoi(c.QueryParam("classroom_id"))
	channelID, _ := strconv.Atoi(c.QueryParam("channel_id"))
	// Also support path param for /channels/:channelId/posts
	if channelID == 0 {
		channelID, _ = strconv.Atoi(c.Param("channelId"))
	}
	page, _ := strconv.Atoi(c.QueryParam("page"))
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	tag := c.QueryParam("tag")

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}

	result, err := h.uc.GetPosts(application.GetPostsInput{
		ClassroomID:   classroomID,
		ChannelID:     channelID,
		Page:          page,
		Limit:         limit,
		Tag:           tag,
		CurrentUserID: userID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "FETCH_FAILED", "message": err.Error()},
		})
	}

	totalPages := 0
	if result.Limit > 0 {
		totalPages = (result.Total + result.Limit - 1) / result.Limit
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"data": result.Posts,
			"pagination": map[string]interface{}{
				"page":        result.Page,
				"limit":       result.Limit,
				"total":       result.Total,
				"total_pages": totalPages,
			},
		},
		"error": nil,
	})
}

func (h *PostHandler) CreatePost(c echo.Context) error {
	userID := middleware.GetUserID(c)
	role := middleware.GetUserRole(c)

	var input application.CreatePostInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "INVALID_INPUT", "message": "잘못된 입력입니다"},
		})
	}

	// Support path param for /channels/:channelId/posts
	if input.ChannelID == 0 {
		input.ChannelID, _ = strconv.Atoi(c.Param("channelId"))
	}

	result, err := h.uc.CreatePost(userID, role, input)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "CREATE_FAILED", "message": err.Error()},
		})
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"success": true, "data": result, "error": nil,
	})
}

func (h *PostHandler) UpdatePost(c echo.Context) error {
	userID := middleware.GetUserID(c)
	role := middleware.GetUserRole(c)

	postID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "INVALID_ID", "message": "유효하지 않은 게시글 ID입니다"},
		})
	}

	var input application.UpdatePostInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "INVALID_INPUT", "message": "잘못된 입력입니다"},
		})
	}

	result, err := h.uc.UpdatePost(postID, userID, role, input)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "UPDATE_FAILED", "message": err.Error()},
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true, "data": result, "error": nil,
	})
}

func (h *PostHandler) LikePost(c echo.Context) error {
	userID := middleware.GetUserID(c)
	postID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "INVALID_ID", "message": "유효하지 않은 게시글 ID입니다"},
		})
	}

	liked, err := h.uc.LikePost(postID, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "LIKE_FAILED", "message": err.Error()},
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true, "data": map[string]interface{}{"liked": liked}, "error": nil,
	})
}

func (h *PostHandler) GetComments(c echo.Context) error {
	postID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "INVALID_ID", "message": "유효하지 않은 게시글 ID입니다"},
		})
	}

	comments, err := h.uc.GetComments(postID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "FETCH_FAILED", "message": err.Error()},
		})
	}

	if comments == nil {
		comments = []*post.Comment{}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"data": comments,
			"pagination": map[string]interface{}{
				"page":        1,
				"limit":       len(comments),
				"total":       len(comments),
				"total_pages": 1,
			},
		},
		"error": nil,
	})
}

func (h *PostHandler) CreateComment(c echo.Context) error {
	userID := middleware.GetUserID(c)

	var input application.CreateCommentInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "INVALID_INPUT", "message": "잘못된 입력입니다"},
		})
	}

	// Support path param for /posts/:id/comments
	if input.PostID == 0 {
		input.PostID, _ = strconv.Atoi(c.Param("id"))
	}

	result, err := h.uc.CreateComment(userID, input)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "CREATE_FAILED", "message": err.Error()},
		})
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"success": true, "data": result, "error": nil,
	})
}

func (h *PostHandler) CreateAssignment(c echo.Context) error {
	userID := middleware.GetUserID(c)

	var input application.CreateAssignmentInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "INVALID_INPUT", "message": "잘못된 입력입니다"},
		})
	}

	p, a, err := h.uc.CreateAssignment(userID, input)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "CREATE_FAILED", "message": err.Error()},
		})
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"success": true, "data": map[string]interface{}{"post": p, "assignment": a}, "error": nil,
	})
}

func (h *PostHandler) SubmitAssignment(c echo.Context) error {
	userID := middleware.GetUserID(c)

	var input application.SubmitAssignmentInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "INVALID_INPUT", "message": "잘못된 입력입니다"},
		})
	}

	result, err := h.uc.SubmitAssignment(userID, input)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "SUBMIT_FAILED", "message": err.Error()},
		})
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"success": true, "data": result, "error": nil,
	})
}

func (h *PostHandler) GradeAssignment(c echo.Context) error {
	var input application.GradeAssignmentInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "INVALID_INPUT", "message": "잘못된 입력입니다"},
		})
	}

	if err := h.uc.GradeAssignment(input); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "GRADE_FAILED", "message": err.Error()},
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true, "data": map[string]string{"message": "채점이 완료되었습니다"}, "error": nil,
	})
}

func (h *PostHandler) GetSubmissions(c echo.Context) error {
	assignmentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "INVALID_ID", "message": "유효하지 않은 과제 ID입니다"},
		})
	}

	submissions, err := h.uc.GetSubmissions(assignmentID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "FETCH_FAILED", "message": err.Error()},
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true, "data": submissions, "error": nil,
	})
}
