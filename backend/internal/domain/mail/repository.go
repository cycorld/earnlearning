package mail

// Repository — 메일 영속성 인터페이스. 조회류는 없으면 (nil, nil) 을 반환한다.
type Repository interface {
	// 주소
	GetAddressByID(id int) (*Address, error)
	GetAddressByOwner(ownerType string, ownerID int) (*Address, error)
	GetAddressByLocalPart(localPart string) (*Address, error) // 상태 무관 (유일성 검사·수신 매칭에서 상태는 호출측이 판단)
	CreateAddress(ownerType string, ownerID, userID int, localPart string) (int, error)
	CreateSharedAddress(adminUserID int, localPart, displayName string) (int, error) // status=approved 즉시
	UpdateAddressLocalPart(id int, localPart string) error // status 를 pending 으로 리셋
	UpdateAddressStatus(id int, status string) error
	ListMailboxesForUser(userID int) ([]*MailboxItem, error)  // 개인 + 소유 회사 + 권한 있는 공용
	ListAddressesAdmin(status string) ([]*AddressAdminItem, error) // status="" 면 전체
	ListSharedAddresses() ([]*SharedAddressItem, error)            // 관리자 공용함 목록 (grant 포함)

	// 공용(shared) 접근 권한
	GrantSharedAccess(addressID, userID int) error       // upsert, revoked=0
	RevokeSharedAccess(addressID, userID int) error       // revoked=1
	HasActiveGrant(addressID, userID int) (bool, error)   // revoked=0 인 grant 존재 여부
	ListActiveGrantUserIDs(addressID int) ([]int, error)  // 수신 알림 fan-out 대상

	// 회사 소유주 조회 (companies 테이블 직접 조인)
	GetCompanyOwnerName(companyID int) (ownerID int, name string, err error)

	// 메일
	CreateEmail(e *Email) (int, error)
	GetEmailByID(id int) (*Email, error)
	ListEmails(addressID int, direction string, limit, offset int) ([]*EmailListItem, int, error)
	ListAllEmails(limit, offset int) ([]*EmailListItem, int, error) // 관리자
	MarkRead(emailID int) error

	// 첨부
	AddAttachment(a *Attachment) (int, error)
	ListAttachments(emailID int) ([]*Attachment, error)
	GetAttachmentByID(id int) (*Attachment, error) // OwnerUserID/AddressID 채워 반환

	// 발신 표시 이름/계정 이메일 조회
	GetUserNameEmail(userID int) (name, accountEmail string, err error)
}
