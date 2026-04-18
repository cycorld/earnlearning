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

// GetChannels godoc
//
//	@Summary		채널 목록 조회
//	@Description	클래스룸 내 채널 목록 조회
//	@Tags			Feed
//	@Produce		json
//	@Security		BearerAuth
//	@Param			classroomId	path		int	true	"클래스룸 ID"
//	@Success		200			{object}	APIResponse
//	@Router			/classrooms/{classroomId}/channels [get]
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

// GetPosts godoc
//
//	@Summary		게시물 목록 조회
//	@Description	채널 또는 클래스룸의 게시물 목록 (페이지네이션, 태그 필터)
//	@Tags			Feed
//	@Produce		json
//	@Security		BearerAuth
//	@Param			classroom_id	query		int		false	"클래스룸 ID"
//	@Param			channel_id		query		int		false	"채널 ID"
//	@Param			page			query		int		false	"페이지 번호"	default(1)
//	@Param			limit			query		int		false	"페이지 크기"	default(20)
//	@Param			tag				query		string	false	"태그 필터"
//	@Success		200				{object}	APIResponse
//	@Router			/posts [get]
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

	if classroomID == 0 && channelID == 0 {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "MISSING_PARAM", "message": "classroom_id 또는 channel_id 파라미터가 필요합니다"},
		})
	}

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

// GetPost godoc
//
//	@Summary		게시물 단건 조회 (딥링크용)
//	@Description	ID로 단일 게시물을 조회한다. 알림·외부 공유 링크에서 진입하는 /post/:id 에 대응.
//	@Tags			Feed
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"게시물 ID"
//	@Success		200	{object}	APIResponse
//	@Failure		404	{object}	APIResponse
//	@Router			/posts/{id} [get]
func (h *PostHandler) GetPost(c echo.Context) error {
	userID := middleware.GetUserID(c)

	postID, err := strconv.Atoi(c.Param("id"))
	if err != nil || postID <= 0 {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "INVALID_PARAM", "message": "유효한 게시물 ID가 필요합니다"},
		})
	}

	p, err := h.uc.GetPost(postID, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "FETCH_FAILED", "message": err.Error()},
		})
	}
	if p == nil {
		return c.JSON(http.StatusNotFound, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "POST_NOT_FOUND", "message": "게시물을 찾을 수 없습니다"},
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    p,
		"error":   nil,
	})
}

// CreatePost godoc
//
//	@Summary		게시물 작성
//	@Description	채널에 새 게시물 작성
//	@Tags			Feed
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			channelId	path		int				true	"채널 ID"
//	@Param			body		body		CreatePostRequest	true	"게시물 내용"
//	@Success		201			{object}	APIResponse
//	@Router			/channels/{channelId}/posts [post]
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

// UpdatePost godoc
//
//	@Summary		게시물 수정
//	@Description	게시물 내용 수정
//	@Tags			Feed
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int				true	"게시물 ID"
//	@Param			body	body		UpdatePostRequest	true	"수정 내용"
//	@Success		200		{object}	APIResponse
//	@Router			/posts/{id} [put]
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

// DeletePost godoc
//
//	@Summary		게시물 삭제
//	@Description	본인 또는 관리자가 게시물 삭제
//	@Tags			Feed
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"게시물 ID"
//	@Success		200	{object}	APIResponse
//	@Router			/posts/{id} [delete]
func (h *PostHandler) DeletePost(c echo.Context) error {
	userID := middleware.GetUserID(c)
	role := middleware.GetUserRole(c)
	postID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "INVALID_ID", "message": "유효하지 않은 게시글 ID입니다"},
		})
	}
	if err := h.uc.DeletePost(postID, userID, role); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "DELETE_FAILED", "message": err.Error()},
		})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true, "data": map[string]string{"message": "게시글이 삭제되었습니다"}, "error": nil,
	})
}

// LikePost godoc
//
//	@Summary		게시물 좋아요
//	@Description	게시물에 좋아요 토글
//	@Tags			Feed
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"게시물 ID"
//	@Success		200	{object}	APIResponse
//	@Router			/posts/{id}/like [post]
func (h *PostHandler) LikePost(c echo.Context) error {
	userID := middleware.GetUserID(c)
	postID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "INVALID_ID", "message": "유효하지 않은 게시글 ID입니다"},
		})
	}

	result, err := h.uc.LikePost(postID, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "LIKE_FAILED", "message": err.Error()},
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true, "data": map[string]interface{}{"liked": result.Liked, "reward": result.Reward}, "error": nil,
	})
}

// GetComments godoc
//
//	@Summary		댓글 목록 조회
//	@Description	게시물의 댓글 목록 조회
//	@Tags			Feed
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"게시물 ID"
//	@Success		200	{object}	APIResponse
//	@Router			/posts/{id}/comments [get]
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

// CreateComment godoc
//
//	@Summary		댓글 작성
//	@Description	게시물에 댓글 작성
//	@Tags			Feed
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int					true	"게시물 ID"
//	@Param			body	body		CreateCommentRequest	true	"댓글 내용"
//	@Success		201		{object}	APIResponse
//	@Router			/posts/{id}/comments [post]
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

// DeleteComment godoc
//
//	@Summary		댓글 삭제
//	@Description	본인 또는 관리자가 댓글 삭제 (보상 회수 포함)
//	@Tags			Feed
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id			path		int	true	"게시물 ID"
//	@Param			commentId	path		int	true	"댓글 ID"
//	@Success		200			{object}	APIResponse
//	@Router			/posts/{id}/comments/{commentId} [delete]
func (h *PostHandler) DeleteComment(c echo.Context) error {
	userID := middleware.GetUserID(c)
	role := middleware.GetUserRole(c)

	commentID, err := strconv.Atoi(c.Param("commentId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "INVALID_ID", "message": "유효하지 않은 댓글 ID입니다"},
		})
	}

	if err := h.uc.DeleteComment(commentID, userID, role); err != nil {
		status := http.StatusBadRequest
		if err.Error() == "댓글을 찾을 수 없습니다" {
			status = http.StatusNotFound
		} else if err.Error() == "본인이 작성한 댓글만 삭제할 수 있습니다" {
			status = http.StatusForbidden
		}
		return c.JSON(status, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "DELETE_FAILED", "message": err.Error()},
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true, "data": map[string]string{"message": "댓글이 삭제되었습니다"}, "error": nil,
	})
}

// CreateAssignment godoc
//
//	@Summary		과제 생성
//	@Description	채널에 새 과제 생성 (관리자)
//	@Tags			Assignment
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		CreateAssignmentRequest	true	"과제 정보"
//	@Success		201		{object}	APIResponse
//	@Router			/channels/{channelId}/assignments [post]
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

// SubmitAssignment godoc
//
//	@Summary		과제 제출
//	@Description	과제에 대한 제출물 작성
//	@Tags			Assignment
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int						true	"과제 ID"
//	@Param			body	body		SubmitAssignmentRequest	true	"제출 내용"
//	@Success		201		{object}	APIResponse
//	@Router			/assignments/{id}/submit [post]
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

// GradeAssignment godoc
//
//	@Summary		과제 채점
//	@Description	제출된 과제 채점 (관리자)
//	@Tags			Assignment
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int						true	"제출 ID"
//	@Param			body	body		GradeAssignmentRequest	true	"채점 정보"
//	@Success		200		{object}	APIResponse
//	@Router			/submissions/{id}/grade [put]
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

// GetSubmissions godoc
//
//	@Summary		제출 목록 조회
//	@Description	과제의 제출물 목록 조회
//	@Tags			Assignment
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"과제 ID"
//	@Success		200	{object}	APIResponse
//	@Router			/assignments/{id}/submissions [get]
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
