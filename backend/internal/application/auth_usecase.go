package application

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/earnlearning/backend/internal/domain/notification"
	"github.com/earnlearning/backend/internal/domain/user"
	"github.com/earnlearning/backend/internal/domain/wallet"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var studentIDRegex = regexp.MustCompile(`^\d{7,10}$`)

// EmailSender는 비밀번호 재설정 메일 발송용 인터페이스 (#128).
// 프로덕션은 email.SESService, 테스트는 fake 구현을 주입한다.
type EmailSender interface {
	SendEmail(to, subject, htmlBody, textBody string) error
	IsEnabled() bool
}

type AuthUseCase struct {
	userRepo    user.Repository
	walletRepo  wallet.Repository
	jwtSecret   string
	emailSender EmailSender
	baseURL     string
	notifUC     *NotificationUseCase
}

func NewAuthUseCase(repo user.Repository, walletRepo wallet.Repository, jwtSecret string) *AuthUseCase {
	return &AuthUseCase{userRepo: repo, walletRepo: walletRepo, jwtSecret: jwtSecret}
}

// SetNotificationUseCase는 가입 승인 알림 발송용 의존성을 주입한다 (#167).
func (uc *AuthUseCase) SetNotificationUseCase(notifUC *NotificationUseCase) {
	uc.notifUC = notifUC
}

type RegisterInput struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	Name       string `json:"name"`
	Department string `json:"department"`
	StudentID  string `json:"student_id"`
}

type LoginInput struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	RememberMe bool   `json:"remember_me"`
}

type AuthResponse struct {
	Token string     `json:"token"`
	User  *user.User `json:"user"`
}

func (uc *AuthUseCase) Register(input RegisterInput) (*AuthResponse, error) {
	if len(input.Password) < 8 {
		return nil, user.ErrWeakPassword
	}
	if !studentIDRegex.MatchString(input.StudentID) {
		return nil, user.ErrInvalidStudent
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), 10)
	if err != nil {
		return nil, err
	}

	u := &user.User{
		Email:      input.Email,
		Password:   string(hash),
		Name:       input.Name,
		Department: input.Department,
		StudentID:  input.StudentID,
		Role:       user.RoleStudent,
		Status:     user.StatusPending,
	}

	id, err := uc.userRepo.Create(u)
	if err != nil {
		return nil, err
	}

	u.ID = id
	token, err := uc.generateToken(u)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{Token: token, User: u}, nil
}

func (uc *AuthUseCase) Login(input LoginInput) (*AuthResponse, error) {
	u, err := uc.userRepo.FindByEmail(input.Email)
	if err != nil {
		return nil, user.ErrInvalidCreds
	}

	if u.Status == user.StatusRejected {
		return nil, user.ErrRejected
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(input.Password)); err != nil {
		return nil, user.ErrInvalidCreds
	}

	duration := 24 * time.Hour
	if input.RememberMe {
		duration = 180 * 24 * time.Hour // 6 months
	}

	token, err := uc.generateTokenWithDuration(u, duration)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{Token: token, User: u}, nil
}

// SetEmailService는 비밀번호 재설정 메일 발송 의존성을 주입한다 (#128).
// baseURL: 재설정 링크 prefix (예: https://earnlearning.com)
func (uc *AuthUseCase) SetEmailService(sender EmailSender, baseURL string) {
	uc.emailSender = sender
	uc.baseURL = baseURL
}

const resetTokenTTL = 1 * time.Hour

// ForgotPassword는 이메일이 등록된 경우에만 재설정 토큰을 발급·발송한다.
// 이메일 존재 여부를 노출하지 않기 위해 미등록 이메일도 에러 없이 반환한다.
func (uc *AuthUseCase) ForgotPassword(email string) error {
	u, err := uc.userRepo.FindByEmail(email)
	if err != nil {
		return nil // 미등록 이메일 — 침묵 (enumeration 방지)
	}

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return err
	}
	token := hex.EncodeToString(raw)
	hash := sha256.Sum256([]byte(token))

	if err := uc.userRepo.SaveResetToken(u.ID, hex.EncodeToString(hash[:]), time.Now().Add(resetTokenTTL)); err != nil {
		return err
	}

	resetURL := uc.baseURL + "/reset-password?token=" + token

	if uc.emailSender == nil || !uc.emailSender.IsEnabled() {
		// dev 환경 (SES 미설정) — 로그로 대체
		log.Printf("password reset (email disabled): user=%d url=%s", u.ID, resetURL)
		return nil
	}

	subject := "[언러닝] 비밀번호 재설정 안내"
	text := fmt.Sprintf("안녕하세요, %s님.\n\n아래 링크에서 비밀번호를 재설정할 수 있습니다 (1시간 유효):\n%s\n\n본인이 요청하지 않았다면 이 메일을 무시하세요.", u.Name, resetURL)
	html := fmt.Sprintf(`<p>안녕하세요, %s님.</p>
<p>아래 버튼을 눌러 비밀번호를 재설정하세요. 링크는 <strong>1시간</strong> 동안 유효합니다.</p>
<p><a href="%s" style="display:inline-block;padding:12px 24px;background:#4f46e5;color:#fff;border-radius:8px;text-decoration:none">비밀번호 재설정</a></p>
<p>버튼이 동작하지 않으면 다음 주소를 브라우저에 붙여넣으세요:<br>%s</p>
<p>본인이 요청하지 않았다면 이 메일을 무시하세요.</p>`, u.Name, resetURL, resetURL)

	if err := uc.emailSender.SendEmail(u.Email, subject, html, text); err != nil {
		log.Printf("password reset email failed: user=%d err=%v", u.ID, err)
		return err
	}
	return nil
}

// ChangePassword는 로그인 사용자가 현재 비밀번호 확인 후 새 비밀번호로 교체한다 (#131).
func (uc *AuthUseCase) ChangePassword(userID int, currentPassword, newPassword string) error {
	if len(newPassword) < 8 {
		return user.ErrWeakPassword
	}

	u, err := uc.userRepo.FindByID(userID)
	if err != nil {
		return user.ErrInvalidCreds
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(currentPassword)); err != nil {
		return user.ErrInvalidCreds
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), 10)
	if err != nil {
		return err
	}
	return uc.userRepo.UpdatePassword(userID, string(hash))
}

// ResetPassword는 토큰을 검증(1회용·TTL)하고 새 비밀번호로 교체한다.
func (uc *AuthUseCase) ResetPassword(token, newPassword string) error {
	if len(newPassword) < 8 {
		return user.ErrWeakPassword
	}

	hash := sha256.Sum256([]byte(token))
	userID, err := uc.userRepo.ConsumeResetToken(hex.EncodeToString(hash[:]))
	if err != nil {
		return err
	}

	pwHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), 10)
	if err != nil {
		return err
	}
	return uc.userRepo.UpdatePassword(userID, string(pwHash))
}

func (uc *AuthUseCase) UpdateAvatar(userID int, avatarURL string) error {
	return uc.userRepo.UpdateAvatarURL(userID, avatarURL)
}

func (uc *AuthUseCase) GetUserActivity(userID int) (*user.UserActivity, error) {
	return uc.userRepo.GetUserActivity(userID)
}

func (uc *AuthUseCase) GetMe(userID int) (*user.User, error) {
	return uc.userRepo.FindByID(userID)
}

func (uc *AuthUseCase) GetProfile(userID int) (*user.User, error) {
	return uc.userRepo.FindByID(userID)
}

// SearchUsers (#132) — 멘션 자동완성용. approved 유저 이름/학번 부분일치, 빈 검색어는 빈 결과.
func (uc *AuthUseCase) SearchUsers(q string, limit int) ([]*user.User, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return []*user.User{}, nil
	}
	if limit < 1 || limit > 20 {
		limit = 10
	}
	users, err := uc.userRepo.SearchApproved(q, limit)
	if err != nil {
		return nil, err
	}
	if users == nil {
		users = []*user.User{}
	}
	return users, nil
}

func (uc *AuthUseCase) AdminGetPending() ([]*user.User, error) {
	return uc.userRepo.FindByStatus(user.StatusPending)
}

func (uc *AuthUseCase) AdminApprove(userID int) error {
	if err := uc.userRepo.UpdateStatus(userID, user.StatusApproved); err != nil {
		return err
	}

	// Create wallet for the newly approved user (balance starts at 0;
	// initial capital is granted later when the student joins a classroom).
	_, err := uc.walletRepo.FindByUserID(userID)
	if err != nil {
		// Wallet does not exist yet — create one.
		if _, createErr := uc.walletRepo.CreateWallet(userID); createErr != nil {
			fmt.Printf("wallet creation failed for approved user %d: %v\n", userID, createErr)
			// Non-fatal: the classroom-join flow will retry wallet creation.
		}
	}

	// 가입 승인 알림 (#167) — 학생이 pending 화면에서 폴링/재접속하면 바로 확인 가능.
	// 알림 실패는 승인 자체를 막지 않는다 (non-fatal).
	if uc.notifUC != nil {
		if err := uc.notifUC.CreateNotification(
			userID,
			notification.NotifUserApproved,
			"가입 승인 완료",
			"환영합니다! 이제 언러닝을 시작할 수 있어요.",
			"user",
			userID,
		); err != nil {
			log.Printf("approval notification failed for user %d: %v", userID, err)
		}
	}

	return nil
}

func (uc *AuthUseCase) AdminReject(userID int) error {
	return uc.userRepo.UpdateStatus(userID, user.StatusRejected)
}

type UserListResult struct {
	Users      []*user.User `json:"users"`
	Total      int          `json:"total"`
	TotalPages int          `json:"total_pages"`
}

func (uc *AuthUseCase) AdminListUsers(page, limit int) (*UserListResult, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 1000 {
		limit = 20
	}

	users, total, err := uc.userRepo.ListAll(page, limit)
	if err != nil {
		return nil, err
	}

	totalPages := total / limit
	if total%limit != 0 {
		totalPages++
	}

	return &UserListResult{
		Users:      users,
		Total:      total,
		TotalPages: totalPages,
	}, nil
}

type jwtClaims struct {
	UserID int    `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	Status string `json:"status"`
	jwt.RegisteredClaims
}

// RefreshToken validates an existing token (even if recently expired within grace period)
// and returns a new token with refreshed expiry. Grace period: 7 days after expiry.
func (uc *AuthUseCase) RefreshToken(tokenStr string) (*AuthResponse, error) {
	claims := &jwtClaims{}

	// Parse with a lenient clock skew to allow expired tokens within grace period
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(uc.jwtSecret), nil
	}, jwt.WithLeeway(7*24*time.Hour))

	if err != nil || !token.Valid {
		return nil, user.ErrInvalidCreds
	}

	// Look up fresh user data (status may have changed)
	u, err := uc.userRepo.FindByID(claims.UserID)
	if err != nil {
		return nil, user.ErrInvalidCreds
	}

	if u.Status == user.StatusRejected {
		return nil, user.ErrRejected
	}

	newToken, err := uc.generateToken(u)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{Token: newToken, User: u}, nil
}

func (uc *AuthUseCase) ImpersonateUser(userID int) (*AuthResponse, error) {
	u, err := uc.userRepo.FindByID(userID)
	if err != nil {
		return nil, user.ErrInvalidCreds
	}

	token, err := uc.generateToken(u)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{Token: token, User: u}, nil
}

func (uc *AuthUseCase) generateToken(u *user.User) (string, error) {
	return uc.generateTokenWithDuration(u, 24*time.Hour)
}

func (uc *AuthUseCase) generateTokenWithDuration(u *user.User, duration time.Duration) (string, error) {
	claims := jwtClaims{
		UserID: u.ID,
		Email:  u.Email,
		Role:   string(u.Role),
		Status: string(u.Status),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(uc.jwtSecret))
}
