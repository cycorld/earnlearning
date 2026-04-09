package main

import (
	"log"
	"os"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/earnlearning/backend/internal/application"
	"github.com/earnlearning/backend/internal/infrastructure/config"
	"github.com/earnlearning/backend/internal/infrastructure/persistence"
	"github.com/earnlearning/backend/internal/infrastructure/email"
	"github.com/earnlearning/backend/internal/infrastructure/push"
	"github.com/earnlearning/backend/internal/infrastructure/userdbadmin"
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
	userDBRepo := persistence.NewUserDBRepo(db)

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
		// Default: try project root tasks/ first, fallback to ./tasks
		if _, err := os.Stat("../tasks"); err == nil {
			tasksPath = "../tasks"
		} else {
			tasksPath = "./tasks"
		}
	}
	taskRepo := persistence.NewTaskRepo(tasksPath)

	// DM
	dmRepo := persistence.NewDMRepo(db)
	dmUC := application.NewDMUseCase(dmRepo, userRepo, hub)
	dmUC.SetNotificationUseCase(notifUC)

	// OAuth
	oauthRepo := persistence.NewOAuthRepo(db)
	oauthUC := application.NewOAuthUseCase(oauthRepo, userRepo)

	// 학생 DB 프로비저너 (POSTGRES_ADMIN_URL 이 비면 NoopProvisioner 가 사용됨)
	userDBProvisioner, err := userdbadmin.New(userdbadmin.Config{
		AdminDSN:   os.Getenv("POSTGRES_ADMIN_URL"),
		PublicHost: os.Getenv("POSTGRES_PUBLIC_HOST"),
		PublicPort: atoiDefault(os.Getenv("POSTGRES_PUBLIC_PORT"), 6432),
	})
	if err != nil {
		log.Printf("userdb provisioner disabled: %v", err)
		userDBProvisioner = userdbadmin.NewNoop()
	}
	userDBUC := application.NewUserDBUseCase(
		userDBRepo,
		userDBProvisioner,
		application.NewUserRepoNameResolver(userRepo),
		atoiDefault(os.Getenv("USER_DB_MAX_PER_USER"), 3),
	)

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
		DM:           handler.NewDMHandler(dmUC),
		UserDB:       handler.NewUserDBHandler(userDBUC),
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

// atoiDefault 는 빈 문자열이나 파싱 실패 시 기본값을 돌려준다.
func atoiDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}
