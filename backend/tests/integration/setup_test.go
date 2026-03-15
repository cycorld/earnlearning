package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/earnlearning/backend/internal/application"
	"github.com/earnlearning/backend/internal/infrastructure/persistence"
	"github.com/earnlearning/backend/internal/infrastructure/push"
	"github.com/earnlearning/backend/internal/interfaces/http/handler"
	"github.com/earnlearning/backend/internal/interfaces/http/router"
	"github.com/earnlearning/backend/internal/interfaces/ws"
)

const (
	testJWTSecret    = "test-jwt-secret"
	testAdminEmail   = "admin@test.com"
	testAdminPass    = "admin1234"
	testUploadPath   = "/tmp/earnlearning-test-uploads"
)

type testServer struct {
	server *httptest.Server
	t      *testing.T
}

// setupTestServer creates a fresh test server with an in-memory-like temp DB.
func setupTestServer(t *testing.T) *testServer {
	t.Helper()

	// Create temp DB
	tmpFile, err := os.CreateTemp("", "earnlearning-test-*.db")
	if err != nil {
		t.Fatalf("create temp db: %v", err)
	}
	tmpFile.Close()
	t.Cleanup(func() { os.Remove(tmpFile.Name()) })

	db, err := persistence.NewDB(tmpFile.Name())
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if err := persistence.RunMigrations(db); err != nil {
		t.Fatalf("migrations: %v", err)
	}
	if err := persistence.SeedAdmin(db, testAdminEmail, testAdminPass); err != nil {
		t.Fatalf("seed admin: %v", err)
	}

	os.MkdirAll(testUploadPath, 0755)

	// Repos
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

	hub := ws.NewHub()
	go hub.Run()

	pushSvc := push.NewWebPushService("", "", "", notifRepo)

	// Use Cases
	authUC := application.NewAuthUseCase(userRepo, walletRepo, testJWTSecret)
	walletUC := application.NewWalletUseCase(walletRepo, userRepo)
	classroomUC := application.NewClassroomUseCase(classroomRepo, walletRepo)
	companyUC := application.NewCompanyUsecase(companyRepo, userRepo, walletRepo)
	postUC := application.NewPostUsecase(postRepo, walletRepo)
	uploadUC := application.NewUploadUsecase(postRepo, testUploadPath)
	notifUC := application.NewNotificationUseCase(notifRepo, pushSvc, hub)
	freelanceUC := application.NewFreelanceUseCase(db, freelanceRepo, walletRepo, notifUC)
	grantUC := application.NewGrantUseCase(db, grantRepo, walletRepo, notifUC)
	investmentUC := application.NewInvestmentUseCase(db, investmentRepo, companyRepo, walletRepo)
	exchangeUC := application.NewExchangeUseCase(exchangeRepo, companyRepo, walletRepo)
	exchangeUC.SetShareholderUpdater(shareholderUpdater)
	loanUC := application.NewLoanUseCase(db, loanRepo, walletRepo)

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
	}

	e := echo.New()
	e.HideBanner = true
	router.Setup(e, handlers, hub, testJWTSecret, "test", "abc1234")

	ts := httptest.NewServer(e)
	t.Cleanup(func() { ts.Close() })

	return &testServer{server: ts, t: t}
}

// request helpers

type apiResponse struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   *apiError       `json:"error"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (ts *testServer) url(path string) string {
	return ts.server.URL + path
}

func (ts *testServer) get(path, token string) *apiResponse {
	ts.t.Helper()
	req, _ := http.NewRequest("GET", ts.url(path), nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		ts.t.Fatalf("GET %s: %v", path, err)
	}
	defer resp.Body.Close()
	return ts.parseResponse(resp)
}

func (ts *testServer) post(path string, body interface{}, token string) *apiResponse {
	ts.t.Helper()
	var bodyReader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(b)
	}
	req, _ := http.NewRequest("POST", ts.url(path), bodyReader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		ts.t.Fatalf("POST %s: %v", path, err)
	}
	defer resp.Body.Close()
	return ts.parseResponse(resp)
}

func (ts *testServer) put(path string, body interface{}, token string) *apiResponse {
	ts.t.Helper()
	var bodyReader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(b)
	}
	req, _ := http.NewRequest("PUT", ts.url(path), bodyReader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		ts.t.Fatalf("PUT %s: %v", path, err)
	}
	defer resp.Body.Close()
	return ts.parseResponse(resp)
}

func (ts *testServer) parseResponse(resp *http.Response) *apiResponse {
	ts.t.Helper()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		ts.t.Fatalf("read response: %v", err)
	}
	var r apiResponse
	if err := json.Unmarshal(body, &r); err != nil {
		ts.t.Fatalf("parse response JSON: %v\nbody: %s", err, string(body))
	}
	return &r
}

// login returns a JWT token for the given email/password.
func (ts *testServer) login(email, password string) string {
	ts.t.Helper()
	r := ts.post("/api/auth/login", map[string]string{
		"email": email, "password": password,
	}, "")
	if !r.Success {
		ts.t.Fatalf("login failed: %s", string(r.Data))
	}
	var data struct {
		Token string `json:"token"`
	}
	json.Unmarshal(r.Data, &data)
	return data.Token
}

// register creates a new user account and returns the response.
func (ts *testServer) register(email, password, name, studentID string) *apiResponse {
	ts.t.Helper()
	return ts.post("/api/auth/register", map[string]string{
		"email":      email,
		"password":   password,
		"name":       name,
		"student_id": studentID,
	}, "")
}

// approveUser approves a user by ID using admin token.
func (ts *testServer) approveUser(adminToken string, userID int) *apiResponse {
	ts.t.Helper()
	return ts.put(fmt.Sprintf("/api/admin/users/%d/approve", userID), nil, adminToken)
}

// registerAndApprove creates a user, approves them, and returns their token.
func (ts *testServer) registerAndApprove(email, password, name, studentID string) string {
	ts.t.Helper()

	// Register
	regResp := ts.register(email, password, name, studentID)
	if !regResp.Success {
		ts.t.Fatalf("register failed: %v", regResp.Error)
	}
	var regData struct {
		User struct {
			ID int `json:"id"`
		} `json:"user"`
	}
	json.Unmarshal(regResp.Data, &regData)

	// Approve via admin
	adminToken := ts.login(testAdminEmail, testAdminPass)
	ts.approveUser(adminToken, regData.User.ID)

	// Login as new user
	return ts.login(email, password)
}
