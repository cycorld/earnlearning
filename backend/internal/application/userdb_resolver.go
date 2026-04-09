package application

import (
	"fmt"

	"github.com/earnlearning/backend/internal/domain/user"
)

// UserRepoNameResolver 는 user.Repository 를 이용해 UserNameResolver 를 구현한다.
// 이메일 local-part 를 PG slug 로 변환하는 전략을 쓴다.
type UserRepoNameResolver struct {
	users user.Repository
}

func NewUserRepoNameResolver(u user.Repository) *UserRepoNameResolver {
	return &UserRepoNameResolver{users: u}
}

func (r *UserRepoNameResolver) PGSlugByUserID(userID int) (string, error) {
	u, err := r.users.FindByID(userID)
	if err != nil {
		return "", err
	}
	slug := SlugFromEmail(u.Email)
	if slug == "" {
		return "", fmt.Errorf("빈 슬러그")
	}
	return slug, nil
}
