package freelance

type JobFilter struct {
	Status string
	Skills string
	MinBudget int
}

type Repository interface {
	Create(job *FreelanceJob) (int, error)
	FindByID(id int) (*FreelanceJob, error)
	List(filter JobFilter, page, limit int) ([]*FreelanceJob, int, error)
	UpdateStatus(id int, status JobStatus) error
	SetFreelancer(jobID, freelancerID, agreedPrice int) error
	SetEscrow(jobID, amount int) error
	SetWorkCompleted(jobID int) error
	SetCompleted(jobID int) error
	SaveCompletionReport(jobID int, report, media string) error

	CreateApplication(app *JobApplication) (int, error)
	FindApplicationByID(id int) (*JobApplication, error)
	FindApplicationByJobAndUser(jobID, userID int) (*JobApplication, error)
	ListApplicationsByJob(jobID int) ([]*JobApplication, error)
	UpdateApplicationStatus(id int, status ApplicationStatus) error
	CountAcceptedApplications(jobID int) (int, error)
	SetApplicationEscrow(appID, amount int) error
	SetApplicationWorkCompleted(appID int, report, media string) error

	CreateReview(review *FreelanceReview) (int, error)
	FindReviewByJobAndReviewer(jobID, reviewerID int) (*FreelanceReview, error)
	ListReviewsByUser(userID int) ([]*FreelanceReview, error)
}
