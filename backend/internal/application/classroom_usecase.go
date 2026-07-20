package application

import (
	"crypto/rand"
	"math/big"

	"github.com/earnlearning/backend/internal/domain/classroom"
	"github.com/earnlearning/backend/internal/domain/wallet"
)

type ClassroomUseCase struct {
	classroomRepo classroom.Repository
	walletRepo    wallet.Repository
}

func NewClassroomUseCase(cr classroom.Repository, wr wallet.Repository) *ClassroomUseCase {
	return &ClassroomUseCase{classroomRepo: cr, walletRepo: wr}
}

type CreateClassroomInput struct {
	Name           string `json:"name"`
	InitialCapital int    `json:"initial_capital"`
}

type defaultChannel struct {
	Name        string
	Slug        string
	ChannelType string
	WriteRole   string
	SortOrder   int
}

var defaultChannels = []defaultChannel{
	{Name: "공지", Slug: "notice", ChannelType: "notice", WriteRole: "admin", SortOrder: 1},
	{Name: "자유", Slug: "free", ChannelType: "free", WriteRole: "all", SortOrder: 2},
	{Name: "과제", Slug: "assignment", ChannelType: "assignment", WriteRole: "admin", SortOrder: 3},
	{Name: "쇼케이스", Slug: "showcase", ChannelType: "showcase", WriteRole: "all", SortOrder: 4},
	{Name: "외주마켓", Slug: "market", ChannelType: "market", WriteRole: "all", SortOrder: 5},
	{Name: "투자라운지", Slug: "invest", ChannelType: "invest", WriteRole: "all", SortOrder: 6},
	{Name: "거래소", Slug: "exchange", ChannelType: "exchange", WriteRole: "all", SortOrder: 7},
}

func (uc *ClassroomUseCase) CreateClassroom(input CreateClassroomInput, creatorID int) (*classroom.Classroom, error) {
	code, err := generateCode(6)
	if err != nil {
		return nil, err
	}

	if input.InitialCapital <= 0 {
		input.InitialCapital = 50000000
	}

	c := &classroom.Classroom{
		Name:           input.Name,
		Code:           code,
		CreatedBy:      creatorID,
		InitialCapital: input.InitialCapital,
		Settings:       "{}",
	}

	id, err := uc.classroomRepo.Create(c)
	if err != nil {
		return nil, err
	}
	c.ID = id

	// Add creator as a member of the classroom
	if err := uc.classroomRepo.AddMember(id, creatorID); err != nil {
		return nil, err
	}

	// #159 생성한 강의실을 생성자의 활성 강의실로 전환 (관리자가 만든 지원금 등이
	// 이 강의실에 귀속되도록)
	if err := uc.classroomRepo.SetActiveClassroom(creatorID, id); err != nil {
		return nil, err
	}

	// Create default channels
	for _, ch := range defaultChannels {
		_, err := uc.classroomRepo.CreateChannel(&classroom.Channel{
			ClassroomID: id,
			Name:        ch.Name,
			Slug:        ch.Slug,
			ChannelType: ch.ChannelType,
			WriteRole:   ch.WriteRole,
			SortOrder:   ch.SortOrder,
		})
		if err != nil {
			return nil, err
		}
	}

	return c, nil
}

type JoinClassroomInput struct {
	Code string `json:"code"`
}

func (uc *ClassroomUseCase) JoinClassroom(code string, userID int) (*classroom.Classroom, error) {
	c, err := uc.classroomRepo.FindByCode(code)
	if err != nil {
		return nil, err
	}

	isMember, err := uc.classroomRepo.IsMember(c.ID, userID)
	if err != nil {
		return nil, err
	}
	if isMember {
		// already a member, idempotent — 재조인도 활성 강의실 전환으로 취급 (#159)
		if err := uc.classroomRepo.SetActiveClassroom(userID, c.ID); err != nil {
			return nil, err
		}
		return c, nil
	}

	if err := uc.classroomRepo.AddMember(c.ID, userID); err != nil {
		return nil, err
	}

	// #159 강의실별 지갑: 이 강의실 전용 지갑을 확보하고, 처음 귀속된 경우에만
	// 해당 강의실의 초기자본을 지급한다 (강의 간 잔액 혼합·중복 지급 차단).
	walletID, isNew, err := uc.walletRepo.EnsureClassroomWallet(userID, c.ID)
	if err != nil {
		return nil, err
	}
	if isNew {
		if err := uc.walletRepo.Credit(walletID, c.InitialCapital, wallet.TxInitialCapital, "초기 자본금 지급", "classroom", c.ID); err != nil {
			return nil, err
		}
	}

	// 조인한 강의실을 활성 컨텍스트로 전환
	if err := uc.classroomRepo.SetActiveClassroom(userID, c.ID); err != nil {
		return nil, err
	}

	return c, nil
}

// ActivateClassroom — 멤버인 강의실로 활성 컨텍스트 전환 (#159).
// 관리자는 멤버가 아니어도 전환 가능 (강의실 관리 컨텍스트 진입).
func (uc *ClassroomUseCase) ActivateClassroom(userID, classroomID int, isAdmin bool) (*classroom.Classroom, error) {
	c, err := uc.classroomRepo.FindByID(classroomID)
	if err != nil {
		return nil, err
	}
	if !isAdmin {
		isMember, err := uc.classroomRepo.IsMember(classroomID, userID)
		if err != nil {
			return nil, err
		}
		if !isMember {
			return nil, classroom.ErrNotMember
		}
	}
	if err := uc.classroomRepo.SetActiveClassroom(userID, classroomID); err != nil {
		return nil, err
	}
	return c, nil
}

func (uc *ClassroomUseCase) GetClassroom(id int) (*classroom.Classroom, error) {
	return uc.classroomRepo.FindByID(id)
}

func (uc *ClassroomUseCase) GetClassroomChannels(classroomID int) ([]*classroom.Channel, error) {
	return uc.classroomRepo.GetChannels(classroomID)
}

func (uc *ClassroomUseCase) GetClassroomMembers(classroomID int) ([]*classroom.ClassroomMember, error) {
	return uc.classroomRepo.GetMembers(classroomID)
}

func (uc *ClassroomUseCase) GetMemberDashboard(classroomID int) ([]*classroom.MemberDashboard, error) {
	return uc.classroomRepo.GetMemberDashboard(classroomID)
}

func (uc *ClassroomUseCase) ListMyClassrooms(userID int) ([]*classroom.Classroom, error) {
	return uc.classroomRepo.ListByUser(userID)
}

const codeChars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func generateCode(length int) (string, error) {
	b := make([]byte, length)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(codeChars))))
		if err != nil {
			return "", err
		}
		b[i] = codeChars[n.Int64()]
	}
	return string(b), nil
}
