package classroom

type Repository interface {
	Create(c *Classroom) (int, error)
	FindByID(id int) (*Classroom, error)
	FindByCode(code string) (*Classroom, error)
	AddMember(classroomID, userID int) error
	IsMember(classroomID, userID int) (bool, error)
	// SetActiveClassroom sets the user's current classroom context (#159).
	SetActiveClassroom(userID, classroomID int) error
	GetMembers(classroomID int) ([]*ClassroomMember, error)
	CreateChannel(ch *Channel) (int, error)
	GetChannels(classroomID int) ([]*Channel, error)
	ListByUser(userID int) ([]*Classroom, error)
	GetMemberDashboard(classroomID int) ([]*MemberDashboard, error)
}
