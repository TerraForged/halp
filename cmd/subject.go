package cmd

type Subject interface {
	Perms() []string
}

type SimpleSubject struct {
	ID string
}

func (s *SimpleSubject) Perms() []string {
	return []string{s.ID}
}

func NewSubject(id string) Subject {
	return &SimpleSubject{ID: id}
}
