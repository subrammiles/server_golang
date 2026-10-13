package course

import (
	"cmp"
	"context"
	"slices"
	"strings"
	"sync"
	"time"
)

// MemoryRepository is an in-memory Repository (for development and tests). Safe for concurrent use.
type MemoryRepository struct {
	mu      sync.RWMutex
	courses map[int64]Course
	nextID  int64
}

func NewMemoryRepository(seed ...Input) *MemoryRepository {
	r := &MemoryRepository{courses: map[int64]Course{}, nextID: 1}
	for _, in := range seed {
		r.Create(context.Background(), in)
	}
	return r
}

func (r *MemoryRepository) List(_ context.Context, q Query) ([]Course, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var matches []Course
	for _, c := range r.courses {
		if (q.Category == "" || c.Category == q.Category) &&
			(q.Q == "" || strings.Contains(strings.ToLower(c.Title), strings.ToLower(q.Q))) {
			matches = append(matches, c)
		}
	}
	desc := strings.HasPrefix(q.Sort, "-")
	slices.SortFunc(matches, func(a, b Course) int {
		var c int
		switch strings.TrimPrefix(q.Sort, "-") {
		case "price":
			c = cmp.Compare(a.Price, b.Price)
		case "durationHours":
			c = cmp.Compare(a.DurationHours, b.DurationHours)
		default:
			c = cmp.Compare(a.Title, b.Title)
		}
		if desc {
			return -c
		}
		return c
	})
	start := min(q.Page*q.Size, len(matches))
	end := min(start+q.Size, len(matches))
	return matches[start:end], len(matches), nil
}

func (r *MemoryRepository) Get(_ context.Context, id int64) (Course, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.courses[id]
	if !ok {
		return Course{}, ErrNotFound
	}
	return c, nil
}

func (r *MemoryRepository) GetByTitle(_ context.Context, title string) (Course, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, c := range r.courses {
		if strings.EqualFold(c.Title, title) {
			return c, nil
		}
	}
	return Course{}, ErrNotFound
}

func (r *MemoryRepository) Create(_ context.Context, in Input) (Course, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c := Course{ID: r.nextID, Title: in.Title, Category: in.Category, DurationHours: in.DurationHours,
		Price: in.Price, CreatedAt: time.Now().UTC().Truncate(time.Microsecond)}
	r.courses[c.ID] = c
	r.nextID++
	return c, nil
}

func (r *MemoryRepository) Update(_ context.Context, c Course) (Course, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.courses[c.ID]; !ok {
		return Course{}, ErrNotFound
	}
	r.courses[c.ID] = c
	return c, nil
}

func (r *MemoryRepository) Delete(_ context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.courses[id]; !ok {
		return ErrNotFound
	}
	delete(r.courses, id)
	return nil
}
