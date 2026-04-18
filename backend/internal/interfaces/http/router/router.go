package router

import (
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"

	"github.com/earnlearning/backend/internal/application"
	"github.com/earnlearning/backend/internal/interfaces/http/handler"
	"github.com/earnlearning/backend/internal/interfaces/http/middleware"
	"github.com/earnlearning/backend/internal/interfaces/ws"
)

// Handlers holds all handler references for route registration.
type Handlers struct {
	Auth         *handler.AuthHandler
	Admin        *handler.AdminHandler
	Classroom    *handler.ClassroomHandler
	Company      *handler.CompanyHandler
	Wallet       *handler.WalletHandler
	Post         *handler.PostHandler
	Upload       *handler.UploadHandler
	Freelance    *handler.FreelanceHandler
	Grant        *handler.GrantHandler
	Investment   *handler.InvestmentHandler
	Exchange     *handler.ExchangeHandler
	Loan         *handler.LoanHandler
	Notification *handler.NotificationHandler
	Task         *handler.TaskHandler
	Docs         *handler.DocsHandler
	OAuth        *handler.OAuthHandler
	OAuthUC      *application.OAuthUseCase // needed for middleware
	DM           *handler.DMHandler
	UserDB       *handler.UserDBHandler
	LLM          *handler.LLMHandler
	Chat         *handler.ChatHandler
}

// Setup registers all routes on the given Echo instance.
func Setup(e *echo.Echo, h *Handlers, hub *ws.Hub, jwtSecret string, buildNumber string, commitSHA string) {
	// Request Logger - errors and slow requests
	e.Use(echomw.LoggerWithConfig(echomw.LoggerConfig{
		Format: "${time_rfc3339} ${status} ${method} ${uri} ${latency_human} ${error}\n",
		Skipper: func(c echo.Context) bool {
			// Skip health checks and successful WebSocket upgrades
			return c.Path() == "/api/health"
		},
	}))
	e.Use(echomw.Recover())

	// CORS
	e.Use(middleware.CORS())

	// API Documentation (public, outside /api group)
	if h.Docs != nil {
		e.GET("/docs", h.Docs.ServeUI)
		e.GET("/docs/openapi.json", h.Docs.ServeSpec)
	}

	api := e.Group("/api")

	// ================================================================
	// Public routes (no auth required)
	// ================================================================
	api.GET("/health", func(c echo.Context) error {
		return c.JSON(200, map[string]interface{}{"success": true, "data": "ok", "error": nil})
	})
	api.GET("/version", func(c echo.Context) error {
		c.Response().Header().Set("Cache-Control", "no-store")
		return c.JSON(200, map[string]interface{}{
			"success": true,
			"data": map[string]string{
				"build_number": buildNumber,
				"commit_sha":   commitSHA,
			},
			"error": nil,
		})
	})
	api.POST("/auth/register", h.Auth.Register)
	api.POST("/auth/login", h.Auth.Login)
	api.POST("/auth/refresh", h.Auth.Refresh)
	api.GET("/push/vapid-public-key", h.Notification.GetVAPIDPublicKey)

	// ================================================================
	// Auth routes (JWT required, any status)
	// ================================================================
	auth := api.Group("", middleware.JWTAuth(jwtSecret, h.OAuthUC))
	auth.GET("/auth/me", h.Auth.GetMe)
	auth.PUT("/auth/avatar", h.Auth.UpdateAvatar)
	auth.GET("/users/:id/profile", h.Auth.GetProfile)
	auth.GET("/users/:id/activity", h.Auth.GetUserActivity)

	// ================================================================
	// Approved routes (JWT + approved status)
	// ================================================================
	approved := auth.Group("", middleware.ApprovedOnly())

	// Wallet (OAuth: read:wallet / write:wallet)
	approved.GET("/wallet", h.Wallet.GetWallet, middleware.RequireScope("read:wallet"))
	approved.GET("/wallet/transactions", h.Wallet.GetTransactions, middleware.RequireScope("read:wallet"))
	approved.GET("/wallet/ranking", h.Wallet.GetRanking, middleware.RequireScope("read:wallet"))
	approved.GET("/wallet/recipients", h.Wallet.SearchRecipients, middleware.RequireScope("read:wallet"))
	approved.POST("/wallet/transfer", h.Wallet.Transfer, middleware.RequireScope("write:wallet"))

	// Classrooms (no OAuth scope — internal only)
	approved.POST("/classrooms", h.Classroom.CreateClassroom)
	approved.POST("/classrooms/join", h.Classroom.JoinClassroom)
	approved.GET("/classrooms", h.Classroom.ListMyClassrooms)
	approved.GET("/classrooms/:id", h.Classroom.GetClassroom)

	// Companies (OAuth: read:company / write:company)
	approved.POST("/companies", h.Company.CreateCompany, middleware.RequireScope("write:company"))
	approved.GET("/companies", h.Company.ListCompaniesPublic, middleware.RequireScope("read:company"))
	approved.GET("/companies/mine", h.Company.GetMyCompanies, middleware.RequireScope("read:company"))
	approved.GET("/companies/:id", h.Company.GetCompany, middleware.RequireScope("read:company"))
	approved.PUT("/companies/:id", h.Company.UpdateCompany, middleware.RequireScope("write:company"))
	approved.POST("/companies/:id/business-card", h.Company.CreateBusinessCard, middleware.RequireScope("write:company"))
	approved.GET("/companies/:id/business-card", h.Company.GetBusinessCard, middleware.RequireScope("read:company"))
	approved.POST("/companies/:id/disclosures", h.Company.CreateDisclosure, middleware.RequireScope("write:company"))
	approved.GET("/companies/:id/disclosures", h.Company.GetDisclosures, middleware.RequireScope("read:company"))

	// Company wallet (#031)
	approved.GET("/companies/:id/wallet", h.Company.GetCompanyWallet, middleware.RequireScope("read:company"))
	approved.GET("/companies/:id/transactions", h.Company.GetCompanyTransactions, middleware.RequireScope("read:company"))
	approved.POST("/companies/:id/transfer", h.Company.TransferFromCompany, middleware.RequireScope("write:company"))

	// Shareholder proposals (주주총회 투표) — #022
	approved.POST("/companies/:id/proposals", h.Company.CreateProposal, middleware.RequireScope("write:company"))
	approved.GET("/companies/:id/proposals", h.Company.GetProposals, middleware.RequireScope("read:company"))
	approved.GET("/proposals/:pid", h.Company.GetProposal, middleware.RequireScope("read:company"))
	approved.POST("/proposals/:pid/vote", h.Company.CastVote, middleware.RequireScope("write:company"))
	approved.POST("/proposals/:pid/cancel", h.Company.CancelProposal, middleware.RequireScope("write:company"))
	approved.POST("/proposals/:pid/execute", h.Company.ExecuteLiquidation, middleware.RequireScope("write:company"))

	// Feed / Posts (OAuth: read:posts / write:posts)
	approved.GET("/classrooms/:classroomId/channels", h.Post.GetChannels, middleware.RequireScope("read:posts"))
	approved.GET("/posts", h.Post.GetPosts, middleware.RequireScope("read:posts"))
	approved.GET("/posts/:id", h.Post.GetPost, middleware.RequireScope("read:posts"))
	approved.GET("/channels/:channelId/posts", h.Post.GetPosts, middleware.RequireScope("read:posts"))
	approved.POST("/channels/:channelId/posts", h.Post.CreatePost, middleware.RequireScope("write:posts"))
	approved.PUT("/posts/:id", h.Post.UpdatePost, middleware.RequireScope("write:posts"))
	approved.DELETE("/posts/:id", h.Post.DeletePost, middleware.RequireScope("write:posts"))
	approved.POST("/posts/:id/like", h.Post.LikePost, middleware.RequireScope("write:posts"))
	approved.GET("/posts/:id/comments", h.Post.GetComments, middleware.RequireScope("read:posts"))
	approved.POST("/posts/:id/comments", h.Post.CreateComment, middleware.RequireScope("write:posts"))
	approved.DELETE("/posts/:id/comments/:commentId", h.Post.DeleteComment, middleware.RequireScope("write:posts"))

	// Assignments (write:posts scope)
	approved.POST("/channels/:channelId/assignments", h.Post.CreateAssignment, middleware.RequireScope("write:posts"))
	approved.POST("/assignments/:id/submit", h.Post.SubmitAssignment, middleware.RequireScope("write:posts"))
	approved.PUT("/submissions/:id/grade", h.Post.GradeAssignment, middleware.RequireScope("write:posts"))
	approved.GET("/assignments/:id/submissions", h.Post.GetSubmissions, middleware.RequireScope("read:posts"))

	// Upload (write:posts scope)
	approved.POST("/upload", h.Upload.Upload, middleware.RequireScope("write:posts"))

	// Government Grants (OAuth: read:market / write:market)
	approved.GET("/grants", h.Grant.ListGrants, middleware.RequireScope("read:market"))
	approved.GET("/grants/:id", h.Grant.GetGrant, middleware.RequireScope("read:market"))
	approved.POST("/grants/:id/apply", h.Grant.ApplyToGrant, middleware.RequireScope("write:market"))
	approved.PUT("/grants/:id/applications/:appId", h.Grant.UpdateApplication, middleware.RequireScope("write:market"))
	approved.DELETE("/grants/:id/applications/:appId", h.Grant.DeleteApplication, middleware.RequireScope("write:market"))

	// DM (no OAuth scope — internal only)
	approved.POST("/dm/messages", h.DM.SendMessage)
	approved.GET("/dm/conversations", h.DM.GetConversations)
	approved.GET("/dm/messages/:userId", h.DM.GetMessages)
	approved.PUT("/dm/messages/:userId/read", h.DM.MarkAsRead)
	approved.GET("/dm/unread-count", h.DM.GetUnreadCount)

	// Freelance Market (OAuth: read:market / write:market)
	approved.GET("/freelance/jobs", h.Freelance.ListJobs, middleware.RequireScope("read:market"))
	approved.POST("/freelance/jobs", h.Freelance.CreateJob, middleware.RequireScope("write:market"))
	approved.GET("/freelance/jobs/:id", h.Freelance.GetJob, middleware.RequireScope("read:market"))
	approved.GET("/freelance/jobs/:id/applications", h.Freelance.ListApplications, middleware.RequireScope("read:market"))
	approved.POST("/freelance/jobs/:id/apply", h.Freelance.ApplyToJob, middleware.RequireScope("write:market"))
	approved.POST("/freelance/jobs/:id/accept", h.Freelance.AcceptApplication, middleware.RequireScope("write:market"))
	approved.POST("/freelance/jobs/:id/complete", h.Freelance.CompleteWork, middleware.RequireScope("write:market"))
	approved.POST("/freelance/jobs/:id/approve", h.Freelance.ApproveJob, middleware.RequireScope("write:market"))
	approved.POST("/freelance/jobs/:id/cancel", h.Freelance.CancelJob, middleware.RequireScope("write:market"))
	approved.POST("/freelance/jobs/:id/dispute", h.Freelance.DisputeJob, middleware.RequireScope("write:market"))
	approved.POST("/freelance/jobs/:id/review", h.Freelance.ReviewJob, middleware.RequireScope("write:market"))

	// Investment (OAuth: read:market / write:market)
	approved.POST("/investment/rounds", h.Investment.CreateRound, middleware.RequireScope("write:market"))
	approved.POST("/investment/rounds/:id/invest", h.Investment.Invest, middleware.RequireScope("write:market"))
	approved.GET("/investment/rounds", h.Investment.ListRounds, middleware.RequireScope("read:market"))
	approved.GET("/investment/rounds/:id", h.Investment.GetRound, middleware.RequireScope("read:market"))
	approved.POST("/investment/rounds/:id/close", h.Investment.CloseRoundEarly, middleware.RequireScope("write:market"))
	approved.POST("/investment/rounds/:id/cancel", h.Investment.CancelRound, middleware.RequireScope("write:market"))
	approved.GET("/investment/portfolio", h.Investment.GetPortfolio, middleware.RequireScope("read:market"))
	approved.POST("/investment/dividends", h.Investment.ExecuteDividend, middleware.RequireScope("write:market"))
	approved.GET("/investment/dividends", h.Investment.GetMyDividends, middleware.RequireScope("read:market"))
	approved.POST("/investment/kpi-rules", h.Investment.CreateKpiRule, middleware.RequireScope("write:market"))
	approved.POST("/investment/kpi-revenue", h.Investment.AddKpiRevenue, middleware.RequireScope("write:market"))

	// Exchange (OAuth: read:market / write:market)
	approved.GET("/exchange/companies", h.Exchange.ListCompanies, middleware.RequireScope("read:market"))
	approved.GET("/exchange/orderbook/:companyId", h.Exchange.GetOrderbook, middleware.RequireScope("read:market"))
	approved.POST("/exchange/orders", h.Exchange.PlaceOrder, middleware.RequireScope("write:market"))
	approved.DELETE("/exchange/orders/:id", h.Exchange.CancelOrder, middleware.RequireScope("write:market"))
	approved.GET("/exchange/orders/mine", h.Exchange.GetMyOrders, middleware.RequireScope("read:market"))

	// Loans (OAuth: read:wallet / write:wallet)
	approved.POST("/loans", h.Loan.ApplyLoan, middleware.RequireScope("write:wallet"))
	approved.GET("/loans/mine", h.Loan.GetMyLoans, middleware.RequireScope("read:wallet"))
	approved.GET("/loans/:id/payments", h.Loan.GetLoanPayments, middleware.RequireScope("read:wallet"))
	approved.POST("/loans/:id/repay", h.Loan.RepayLoan, middleware.RequireScope("write:wallet"))

	// User Databases (학생 개인 PostgreSQL 프로비저닝) - 프로필 섹션에서 사용
	if h.UserDB != nil {
		approved.GET("/users/me/databases", h.UserDB.ListMyDatabases)
		approved.POST("/users/me/databases", h.UserDB.CreateMyDatabase)
		approved.POST("/users/me/databases/:id/rotate", h.UserDB.RotateMyDatabasePassword)
		approved.DELETE("/users/me/databases/:id", h.UserDB.DeleteMyDatabase)
	}

	// LLM API Keys (#068) — llm.cycorld.com 프록시 키 발급 + 자정 과금
	if h.LLM != nil {
		approved.GET("/llm/me", h.LLM.GetMyKey)
		approved.POST("/llm/me/rotate", h.LLM.RotateMyKey)
		approved.GET("/llm/me/usage", h.LLM.GetMyUsage)
		approved.GET("/llm/status", h.LLM.GetStatus)
	}

	// Chatbot TA (#071) — in-app chatbot with RAG + skills
	if h.Chat != nil {
		approved.POST("/chat/sessions", h.Chat.CreateSession)
		approved.GET("/chat/sessions", h.Chat.ListSessions)
		approved.GET("/chat/sessions/:id", h.Chat.GetSession)
		approved.POST("/chat/sessions/:id/ask", h.Chat.Ask)
		approved.DELETE("/chat/sessions/:id", h.Chat.DeleteSession)
		approved.GET("/chat/skills", h.Chat.ListSkills)
	}

	// Notifications (OAuth: read:notifications)
	approved.GET("/notifications", h.Notification.GetNotifications, middleware.RequireScope("read:notifications"))
	approved.PUT("/notifications/:id/read", h.Notification.MarkRead, middleware.RequireScope("read:notifications"))
	approved.PUT("/notifications/read-all", h.Notification.MarkAllRead, middleware.RequireScope("read:notifications"))
	approved.POST("/notifications/push/subscribe", h.Notification.SubscribePush)
	approved.DELETE("/notifications/push/subscribe", h.Notification.UnsubscribePush)
	approved.GET("/notifications/push/vapid-key", h.Notification.GetVAPIDPublicKey)
	approved.GET("/notifications/email/preference", h.Notification.GetEmailPreference)
	approved.PUT("/notifications/email/preference", h.Notification.UpdateEmailPreference)

	// ================================================================
	// Admin routes (JWT + approved + admin)
	// ================================================================
	admin := approved.Group("/admin", middleware.AdminOnly())
	admin.GET("/users/pending", h.Admin.GetPendingUsers)
	admin.PUT("/users/:id/approve", h.Admin.ApproveUser)
	admin.PUT("/users/:id/reject", h.Admin.RejectUser)
	admin.GET("/users", h.Admin.ListUsers)
	admin.POST("/wallet/transfer", h.Wallet.AdminTransfer)
	admin.PUT("/loans/:id/approve", h.Loan.ApproveLoan)
	admin.PUT("/loans/:id/reject", h.Loan.RejectLoan)
	admin.POST("/loans/weekly-interest", h.Loan.ProcessWeeklyInterest)
	admin.GET("/loans", h.Loan.AdminListLoans)
	admin.GET("/classrooms/:id/dashboard", h.Classroom.GetClassroomDashboard)
	admin.GET("/companies", h.Company.ListAllCompanies)
	admin.POST("/users/:id/impersonate", h.Admin.ImpersonateUser)
	admin.POST("/notifications/announce", h.Notification.AdminSendAnnouncement)
	admin.POST("/force-reload", h.Admin.ForceReload)

	// Kanban tasks (read-only, source of truth is tasks/ markdown files)
	admin.GET("/tasks", h.Task.ListTasks)

	// Grant admin routes
	admin.POST("/grants", h.Grant.CreateGrant)
	admin.POST("/grants/:id/approve/:appId", h.Grant.ApproveApplication)
	admin.POST("/grants/:id/revoke/:appId", h.Grant.RevokeApplication)
	admin.POST("/grants/:id/close", h.Grant.CloseGrant)
	admin.GET("/disclosures", h.Company.GetAllDisclosures)
	admin.POST("/disclosures/:did/approve", h.Company.ApproveDisclosure)
	admin.POST("/disclosures/:did/reject", h.Company.RejectDisclosure)

	// Admin chat skill / wiki management
	if h.Chat != nil {
		admin.POST("/chat/skills", h.Chat.AdminCreateSkill)
		admin.PUT("/chat/skills/:id", h.Chat.AdminUpdateSkill)
		admin.DELETE("/chat/skills/:id", h.Chat.AdminDeleteSkill)
		admin.GET("/chat/wiki", h.Chat.AdminListWiki)
		admin.POST("/chat/wiki/reindex", h.Chat.AdminReindexWiki)
	}

	// ================================================================
	// OAuth routes
	// ================================================================
	if h.OAuth != nil {
		// Public OAuth endpoint (token exchange)
		api.POST("/oauth/token", h.OAuth.Token)

		// OAuth endpoints requiring JWT login (for app management & authorization)
		approved.POST("/oauth/clients", h.OAuth.RegisterClient)
		approved.GET("/oauth/clients", h.OAuth.ListClients)
		approved.DELETE("/oauth/clients/:id", h.OAuth.DeleteClient)
		approved.GET("/oauth/authorize", h.OAuth.AuthorizePage)
		approved.POST("/oauth/authorize", h.OAuth.Authorize)
		approved.POST("/oauth/revoke", h.OAuth.Revoke)

		// OAuth Bearer protected endpoint
		if h.OAuthUC != nil {
			oauthGroup := api.Group("", middleware.OAuthBearerAuth(h.OAuthUC))
			oauthGroup.GET("/oauth/userinfo", h.OAuth.UserInfo)
		}
	}

	// ================================================================
	// WebSocket
	// ================================================================
	api.GET("/ws", func(c echo.Context) error {
		return ws.ServeWS(hub, jwtSecret, c)
	})
}
