package grant

type GrantFilter struct {
	Status      string
	ClassroomID int // #159 강의실 스코프
}

type Repository interface {
	Create(g *Grant) (int, error)
	FindByID(id int) (*Grant, error)
	List(filter GrantFilter, page, limit int) ([]*Grant, int, error)
	UpdateStatus(id int, status GrantStatus) error

	CreateApplication(app *GrantApplication) (int, error)
	FindApplicationByID(id int) (*GrantApplication, error)
	FindApplicationByGrantAndUser(grantID, userID int) (*GrantApplication, error)
	ListApplicationsByGrant(grantID int) ([]*GrantApplication, error)
	ListApplicationsByUserID(userID int) ([]*GrantApplication, error)
	UpdateApplicationStatus(id int, status ApplicationStatus) error
	UpdateApplicationProposal(id int, proposal string) error
	DeleteApplication(id int) error
}
