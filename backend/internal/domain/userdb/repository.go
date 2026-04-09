package userdb

type Repository interface {
	Create(db *UserDatabase) (int, error)
	FindByID(id int) (*UserDatabase, error)
	FindByUserIDAndProject(userID int, projectName string) (*UserDatabase, error)
	ListByUserID(userID int) ([]*UserDatabase, error)
	CountByUserID(userID int) (int, error)
	MarkRotated(id int) error
	Delete(id int) error
}
