package proposal

import "errors"

var (
	ErrNotFound = errors.New("제안을 찾을 수 없습니다")
)

type Repository interface {
	Create(p *Proposal) (int, error)
	FindByID(id int) (*Proposal, error)
	List(filter Filter) ([]*Proposal, error)
	Count(filter Filter) (int, error)
	UpdateStatus(id int, status Status, adminNote, ticketLink string) error
}
