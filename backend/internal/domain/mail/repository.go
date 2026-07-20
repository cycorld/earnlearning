package mail

// Repository — 메일 영속성 인터페이스. 조회류는 없으면 (nil, nil) 을 반환한다.
type Repository interface {
	// 주소
	GetAddressByUserID(userID int) (*Address, error)
	GetAddressByLocalPart(localPart string) (*Address, error)
	CreateAddress(userID int, localPart string) (int, error)

	// 메일
	CreateEmail(e *Email) (int, error)
	GetEmailByID(id int) (*Email, error)
	ListEmails(ownerUserID int, direction string, limit, offset int) ([]*EmailListItem, int, error)
	ListAllEmails(limit, offset int) ([]*EmailListItem, int, error) // 관리자
	MarkRead(emailID int) error

	// 첨부
	AddAttachment(a *Attachment) (int, error)
	ListAttachments(emailID int) ([]*Attachment, error)
	GetAttachmentByID(id int) (*Attachment, error) // OwnerUserID 채워 반환

	// 발신 표시 이름/계정 이메일 조회
	GetUserNameEmail(userID int) (name, accountEmail string, err error)
}
