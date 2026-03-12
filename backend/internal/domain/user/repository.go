package user

type Repository interface {
	Create(u *User) (int, error)
	FindByID(id int) (*User, error)
	FindByEmail(email string) (*User, error)
	FindByStatus(status Status) ([]*User, error)
	ListAll(page, limit int) ([]*User, int, error)
	UpdateStatus(id int, status Status) error
}
