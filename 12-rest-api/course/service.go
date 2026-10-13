package course

import (
	"context"
	"errors"
	"fmt"
)

// Service holds the business rules. It depends on the Repository INTERFACE, not on a database.
type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service { return &Service{repo: repo} }

type Page struct {
	Items         []Course `json:"items"`
	Page          int      `json:"page"`
	Size          int      `json:"size"`
	TotalElements int      `json:"totalElements"`
	TotalPages    int      `json:"totalPages"`
}

func (s *Service) List(ctx context.Context, q Query) (Page, error) {
	if err := q.validate(); err != nil {
		return Page{}, err
	}
	items, total, err := s.repo.List(ctx, q)
	if err != nil {
		return Page{}, fmt.Errorf("list courses: %w", err)
	}
	if items == nil {
		items = []Course{}
	}
	return Page{Items: items, Page: q.Page, Size: q.Size, TotalElements: total, TotalPages: (total + q.Size - 1) / q.Size}, nil
}

func (s *Service) Get(ctx context.Context, id int64) (Course, error) {
	return s.repo.Get(ctx, id)
}

func (s *Service) Create(ctx context.Context, in Input) (Course, error) {
	in = in.normalize()
	if err := in.validate(); err != nil {
		return Course{}, err
	}
	if err := s.ensureTitleFree(ctx, in.Title, 0); err != nil {
		return Course{}, err
	}
	return s.repo.Create(ctx, in)
}

// Replace (PUT) overwrites every field; Update (PATCH) only the ones sent.
func (s *Service) Replace(ctx context.Context, id int64, in Input) (Course, error) {
	return s.Update(ctx, id, Patch{Title: &in.Title, Category: &in.Category, DurationHours: &in.DurationHours, Price: &in.Price})
}

func (s *Service) Update(ctx context.Context, id int64, p Patch) (Course, error) {
	current, err := s.repo.Get(ctx, id)
	if err != nil {
		return Course{}, err
	}
	in := Input{Title: current.Title, Category: current.Category, DurationHours: current.DurationHours, Price: current.Price}
	if p.Title != nil {
		in.Title = *p.Title
	}
	if p.Category != nil {
		in.Category = *p.Category
	}
	if p.DurationHours != nil {
		in.DurationHours = *p.DurationHours
	}
	if p.Price != nil {
		in.Price = *p.Price
	}
	in = in.normalize()
	if err := in.validate(); err != nil {
		return Course{}, err
	}
	if err := s.ensureTitleFree(ctx, in.Title, id); err != nil {
		return Course{}, err
	}
	current.Title, current.Category, current.DurationHours, current.Price = in.Title, in.Category, in.DurationHours, in.Price
	return s.repo.Update(ctx, current)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func (s *Service) ensureTitleFree(ctx context.Context, title string, exceptID int64) error {
	existing, err := s.repo.GetByTitle(ctx, title)
	switch {
	case errors.Is(err, ErrNotFound):
		return nil
	case err != nil:
		return err
	case existing.ID != exceptID:
		return ErrDuplicate
	}
	return nil
}
