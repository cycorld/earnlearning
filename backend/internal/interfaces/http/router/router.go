package router

import (
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"

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
	auth := api.Group("", middleware.JWTAuth(jwtSecret))
	auth.GET("/auth/me", h.Auth.GetMe)
	auth.GET("/users/:id/profile", h.Auth.GetProfile)

	// ================================================================
	// Approved routes (JWT + approved status)
	// ================================================================
	approved := auth.Group("", middleware.ApprovedOnly())

	// Wallet
	approved.GET("/wallet", h.Wallet.GetWallet)
	approved.GET("/wallet/transactions", h.Wallet.GetTransactions)
	approved.GET("/wallet/ranking", h.Wallet.GetRanking)

	// Classrooms
	approved.POST("/classrooms", h.Classroom.CreateClassroom)
	approved.POST("/classrooms/join", h.Classroom.JoinClassroom)
	approved.GET("/classrooms", h.Classroom.ListMyClassrooms)
	approved.GET("/classrooms/:id", h.Classroom.GetClassroom)

	// Companies
	approved.POST("/companies", h.Company.CreateCompany)
	approved.GET("/companies/mine", h.Company.GetMyCompanies)
	approved.GET("/companies/:id", h.Company.GetCompany)
	approved.PUT("/companies/:id", h.Company.UpdateCompany)
	approved.POST("/companies/:id/business-card", h.Company.CreateBusinessCard)
	approved.GET("/companies/:id/business-card", h.Company.GetBusinessCard)

	// Feed / Posts
	approved.GET("/classrooms/:classroomId/channels", h.Post.GetChannels)
	approved.GET("/posts", h.Post.GetPosts)
	approved.GET("/channels/:channelId/posts", h.Post.GetPosts)
	approved.POST("/channels/:channelId/posts", h.Post.CreatePost)
	approved.POST("/posts/:id/like", h.Post.LikePost)
	approved.GET("/posts/:id/comments", h.Post.GetComments)
	approved.POST("/posts/:id/comments", h.Post.CreateComment)

	// Assignments
	approved.POST("/channels/:channelId/assignments", h.Post.CreateAssignment)
	approved.POST("/assignments/:id/submit", h.Post.SubmitAssignment)
	approved.PUT("/submissions/:id/grade", h.Post.GradeAssignment)
	approved.GET("/assignments/:id/submissions", h.Post.GetSubmissions)

	// Upload
	approved.POST("/upload", h.Upload.Upload)

	// Government Grants (정부과제) — list/get/apply for all, create/approve/close for admin
	approved.GET("/grants", h.Grant.ListGrants)
	approved.GET("/grants/:id", h.Grant.GetGrant)
	approved.POST("/grants/:id/apply", h.Grant.ApplyToGrant)

	// Freelance Market
	approved.GET("/freelance/jobs", h.Freelance.ListJobs)
	approved.POST("/freelance/jobs", h.Freelance.CreateJob)
	approved.GET("/freelance/jobs/:id", h.Freelance.GetJob)
	approved.GET("/freelance/jobs/:id/applications", h.Freelance.ListApplications)
	approved.POST("/freelance/jobs/:id/apply", h.Freelance.ApplyToJob)
	approved.POST("/freelance/jobs/:id/accept", h.Freelance.AcceptApplication)
	approved.POST("/freelance/jobs/:id/complete", h.Freelance.CompleteWork)
	approved.POST("/freelance/jobs/:id/approve", h.Freelance.ApproveJob)
	approved.POST("/freelance/jobs/:id/cancel", h.Freelance.CancelJob)
	approved.POST("/freelance/jobs/:id/dispute", h.Freelance.DisputeJob)
	approved.POST("/freelance/jobs/:id/review", h.Freelance.ReviewJob)

	// Investment
	approved.POST("/investment/rounds", h.Investment.CreateRound)
	approved.POST("/investment/rounds/:id/invest", h.Investment.Invest)
	approved.GET("/investment/rounds", h.Investment.ListRounds)
	approved.GET("/investment/portfolio", h.Investment.GetPortfolio)
	approved.POST("/investment/dividends", h.Investment.ExecuteDividend)
	approved.GET("/investment/dividends", h.Investment.GetMyDividends)
	approved.POST("/investment/kpi-rules", h.Investment.CreateKpiRule)
	approved.POST("/investment/kpi-revenue", h.Investment.AddKpiRevenue)

	// Exchange
	approved.GET("/exchange/companies", h.Exchange.ListCompanies)
	approved.GET("/exchange/orderbook/:companyId", h.Exchange.GetOrderbook)
	approved.POST("/exchange/orders", h.Exchange.PlaceOrder)
	approved.DELETE("/exchange/orders/:id", h.Exchange.CancelOrder)
	approved.GET("/exchange/orders/mine", h.Exchange.GetMyOrders)

	// Loans (Bank)
	approved.POST("/loans", h.Loan.ApplyLoan)
	approved.GET("/loans/mine", h.Loan.GetMyLoans)
	approved.GET("/loans/:id/payments", h.Loan.GetLoanPayments)
	approved.POST("/loans/:id/repay", h.Loan.RepayLoan)

	// Notifications
	approved.GET("/notifications", h.Notification.GetNotifications)
	approved.PUT("/notifications/:id/read", h.Notification.MarkRead)
	approved.PUT("/notifications/read-all", h.Notification.MarkAllRead)
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
	admin.GET("/companies", h.Company.ListAllCompanies)
	admin.POST("/users/:id/impersonate", h.Admin.ImpersonateUser)
	admin.POST("/notifications/announce", h.Notification.AdminSendAnnouncement)

	// Kanban tasks (read-only, source of truth is tasks/ markdown files)
	admin.GET("/tasks", h.Task.ListTasks)

	// Grant admin routes
	admin.POST("/grants", h.Grant.CreateGrant)
	admin.POST("/grants/:id/approve/:appId", h.Grant.ApproveApplication)
	admin.POST("/grants/:id/close", h.Grant.CloseGrant)

	// ================================================================
	// WebSocket
	// ================================================================
	api.GET("/ws", func(c echo.Context) error {
		return ws.ServeWS(hub, jwtSecret, c)
	})
}
