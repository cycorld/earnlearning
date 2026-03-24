package main

import (
	"log"
	"os"

	"github.com/labstack/echo/v4"

	"github.com/earnlearning/backend/internal/application"
	"github.com/earnlearning/backend/internal/infrastructure/config"
	"github.com/earnlearning/backend/internal/infrastructure/persistence"
	"github.com/earnlearning/backend/internal/infrastructure/email"
	"github.com/earnlearning/backend/internal/infrastructure/push"
	"github.com/earnlearning/backend/internal/interfaces/http/handler"
	"github.com/earnlearning/backend/internal/interfaces/http/router"
	"github.com/earnlearning/backend/internal/interfaces/ws"
)

// @title			EarnLearning LMS API
// @version		1.0
// @description	이화여대 창업 교육 LMS API
// @host			earnlearning.com
// @BasePath		/api
// @schemes		https
// @securityDefinitions.apikey	BearerAuth
// @in							header
// @name						Authorization

// Set via -ldflags at build time
var (
	BuildNumber = "dev"
	CommitSHA   = "local"
)

func main() {
	cfg := config.Load()

	// Database
	db, err := persistence.NewDB(cfg.DBPath)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()

	if err := persistence.RunMigrations(db); err != nil {
		log.Fatalf("migrations: %v", err)
	}
	if err := persistence.SeedAdmin(db, cfg.AdminEmail, cfg.AdminPassword); err != nil {
		log.Printf("seed admin: %v", err)
	}

	// Dev seed data (SEED_DEV=1 으로 활성화)
	if os.Getenv("SEED_DEV") == "1" {
		if err := persistence.SeedDevData(db); err != nil {
			log.Printf("seed dev data: %v", err)
		}
	}

	// Ensure upload directory
	os.MkdirAll(cfg.UploadPath, 0755)

	// Repositories
	userRepo := persistence.NewUserRepo(db)
	walletRepo := persistence.NewWalletRepo(db)
	classroomRepo := persistence.NewClassroomRepo(db)
	companyRepo := persistence.NewCompanyRepo(db)
	postRepo := persistence.NewPostRepo(db)
	freelanceRepo := persistence.NewFreelanceRepo(db)
	grantRepo := persistence.NewGrantRepo(db)
	investmentRepo := persistence.NewInvestmentRepo(db)
	exchangeRepo := persistence.NewExchangeRepo(db)
	loanRepo := persistence.NewLoanRepo(db)
	notifRepo := persistence.NewNotificationRepo(db)
	shareholderUpdater := persistence.NewShareholderUpdater(db)

	// WebSocket Hub
	hub := ws.NewHub()
	go hub.Run()

	// Push service
	pushSvc := push.NewWebPushService(cfg.VAPIDPublicKey, cfg.VAPIDPrivateKey, cfg.VAPIDSubject, notifRepo)

	// Email service (SES)
	emailSvc := email.NewSESService(email.Config{
		Region:          cfg.SESRegion,
		AccessKeyID:     cfg.SESAccessKeyID,
		SecretAccessKey:  cfg.SESSecretAccessKey,
		FromEmail:       cfg.SESFromEmail,
	})

	// Use Cases
	authUC := application.NewAuthUseCase(userRepo, walletRepo, cfg.JWTSecret)
	walletUC := application.NewWalletUseCase(walletRepo, userRepo)
	classroomUC := application.NewClassroomUseCase(classroomRepo, walletRepo)
	companyUC := application.NewCompanyUsecase(companyRepo, userRepo, walletRepo)
	postUC := application.NewPostUsecase(postRepo, walletRepo, userRepo)
	uploadUC := application.NewUploadUsecase(postRepo, cfg.UploadPath)
	notifUC := application.NewNotificationUseCase(notifRepo, pushSvc, emailSvc, hub)
	notifUC.SetAutoPoster(application.NewAutoPoster(db))
	freelanceUC := application.NewFreelanceUseCase(db, freelanceRepo, walletRepo, notifUC)
	grantUC := application.NewGrantUseCase(db, grantRepo, walletRepo, notifUC)
	investmentUC := application.NewInvestmentUseCase(db, investmentRepo, companyRepo, walletRepo)
	investmentUC.SetNotificationUseCase(notifUC)
	exchangeUC := application.NewExchangeUseCase(exchangeRepo, companyRepo, walletRepo)
	exchangeUC.SetShareholderUpdater(shareholderUpdater)
	exchangeUC.SetDB(db)
	exchangeUC.SetNotificationUseCase(notifUC)
	postUC.SetNotificationUseCase(notifUC)
	loanUC := application.NewLoanUseCase(db, loanRepo, walletRepo)

	// Task repo (reads tasks/ markdown files)
	tasksPath := os.Getenv("TASKS_PATH")
	if tasksPath == "" {
		tasksPath = "./tasks"
	}
	taskRepo := persistence.NewTaskRepo(tasksPath)

	// OAuth
	oauthRepo := persistence.NewOAuthRepo(db)
	oauthUC := application.NewOAuthUseCase(oauthRepo, userRepo)

	// Docs directory (swagger.json location)
	docsDir := os.Getenv("DOCS_DIR")
	if docsDir == "" {
		docsDir = "./docs"
	}

	// Handlers
	handlers := &router.Handlers{
		Auth:         handler.NewAuthHandler(authUC),
		Admin:        handler.NewAdminHandler(authUC),
		Classroom:    handler.NewClassroomHandler(classroomUC),
		Company:      handler.NewCompanyHandler(companyUC),
		Wallet:       handler.NewWalletHandler(walletUC),
		Post:         handler.NewPostHandler(postUC),
		Upload:       handler.NewUploadHandler(uploadUC),
		Freelance:    handler.NewFreelanceHandler(freelanceUC),
		Grant:        handler.NewGrantHandler(grantUC),
		Investment:   handler.NewInvestmentHandler(investmentUC),
		Exchange:     handler.NewExchangeHandler(exchangeUC),
		Loan:         handler.NewLoanHandler(loanUC),
		Notification: handler.NewNotificationHandler(notifUC),
		Task:         handler.NewTaskHandler(taskRepo),
		Docs:         handler.NewDocsHandler(docsDir),
		OAuth:        handler.NewOAuthHandler(oauthUC),
		OAuthUC:      oauthUC,
	}

	// Echo server
	e := echo.New()
	e.HideBanner = true

	// Static uploads
	e.Static("/uploads", cfg.UploadPath)

	// Set up all routes
	router.Setup(e, handlers, hub, cfg.JWTSecret, BuildNumber, CommitSHA)

	log.Printf("EarnLearning LMS starting on :%s", cfg.Port)
	e.Logger.Fatal(e.Start(":" + cfg.Port))
}
