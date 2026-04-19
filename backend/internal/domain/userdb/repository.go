package userdb

type Repository interface {
	Create(db *UserDatabase) (int, error)
	FindByID(id int) (*UserDatabase, error)
	FindByUserIDAndProject(userID int, projectName string) (*UserDatabase, error)
	FindByDBName(dbName string) (*UserDatabase, error) // admin reconcile / direct delete (#016)
	ListByUserID(userID int) ([]*UserDatabase, error)
	ListAll() ([]*UserDatabase, error) // admin reconcile (#016)
	CountByUserID(userID int) (int, error)
	MarkRotated(id int) error
	Delete(id int) error
}
