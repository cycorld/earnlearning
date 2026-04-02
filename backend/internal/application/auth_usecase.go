package application

import (
	"fmt"
	"regexp"
	"time"

	"github.com/earnlearning/backend/internal/domain/user"
	"github.com/earnlearning/backend/internal/domain/wallet"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var studentIDRegex = regexp.MustCompile(`^\d{7,10}$`)

type AuthUseCase struct {
	userRepo   user.Repository
	walletRepo wallet.Repository
	jwtSecret  string
}

func NewAuthUseCase(repo user.Repository, walletRepo wallet.Repository, jwtSecret string) *AuthUseCase {
	return &AuthUseCase{userRepo: repo, walletRepo: walletRepo, jwtSecret: jwtSecret}
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
