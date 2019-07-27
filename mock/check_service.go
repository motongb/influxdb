package mock

import (
	"context"

	platform "github.com/influxdata/influxdb"
	"go.uber.org/zap"
)

// CheckService is a mock implementation of a retention.CheckService, which
// also makes it a suitable mock to use wherever an platform.CheckService is required.
type CheckService struct {
	// Methods for a retention.CheckService
	OpenFn       func() error
	CloseFn      func() error
	WithLoggerFn func(l *zap.Logger)

	// Methods for an platform.CheckService
	FindCheckByIDFn func(context.Context, platform.ID) (*platform.Check, error)
	FindCheckFn     func(context.Context, platform.CheckFilter) (*platform.Check, error)
	FindChecksFn    func(context.Context, platform.CheckFilter, ...platform.FindOptions) ([]*platform.Check, int, error)
	CreateCheckFn   func(context.Context, *platform.Check) error
	UpdateCheckFn   func(context.Context, platform.ID, platform.CheckUpdate) (*platform.Check, error)
	DeleteCheckFn   func(context.Context, platform.ID) error
}

// NewCheckService returns a mock CheckService where its methods will return
// zero values.
func NewCheckService() *CheckService {
	return &CheckService{
		OpenFn:          func() error { return nil },
		CloseFn:         func() error { return nil },
		WithLoggerFn:    func(l *zap.Logger) {},
		FindCheckByIDFn: func(context.Context, platform.ID) (*platform.Check, error) { return nil, nil },
		FindCheckFn:     func(context.Context, platform.CheckFilter) (*platform.Check, error) { return nil, nil },
		FindChecksFn: func(context.Context, platform.CheckFilter, ...platform.FindOptions) ([]*platform.Check, int, error) {
			return nil, 0, nil
		},
		CreateCheckFn: func(context.Context, *platform.Check) error { return nil },
		UpdateCheckFn: func(context.Context, platform.ID, platform.CheckUpdate) (*platform.Check, error) { return nil, nil },
		DeleteCheckFn: func(context.Context, platform.ID) error { return nil },
	}
}

// Open opens the CheckService.
func (s *CheckService) Open() error { return s.OpenFn() }

// Close closes the CheckService.
func (s *CheckService) Close() error { return s.CloseFn() }

// WithLogger sets the logger on the CheckService.
func (s *CheckService) WithLogger(l *zap.Logger) { s.WithLoggerFn(l) }

// FindCheckByID returns a single check by ID.
func (s *CheckService) FindCheckByID(ctx context.Context, id platform.ID) (*platform.Check, error) {
	return s.FindCheckByIDFn(ctx, id)
}

// FindCheck returns the first check that matches filter.
func (s *CheckService) FindCheck(ctx context.Context, filter platform.CheckFilter) (*platform.Check, error) {
	return s.FindCheckFn(ctx, filter)
}

// FindChecks returns a list of checks that match filter and the total count of matching checks.
func (s *CheckService) FindChecks(ctx context.Context, filter platform.CheckFilter, opts ...platform.FindOptions) ([]*platform.Check, int, error) {
	return s.FindChecksFn(ctx, filter, opts...)
}

// CreateCheck creates a new check and sets b.ID with the new identifier.
func (s *CheckService) CreateCheck(ctx context.Context, check *platform.Check) error {
	return s.CreateCheckFn(ctx, check)
}

// UpdateCheck updates a single check with changeset.
func (s *CheckService) UpdateCheck(ctx context.Context, id platform.ID, upd platform.CheckUpdate) (*platform.Check, error) {
	return s.UpdateCheckFn(ctx, id, upd)
}

// DeleteCheck removes a check by ID.
func (s *CheckService) DeleteCheck(ctx context.Context, id platform.ID) error {
	return s.DeleteCheckFn(ctx, id)
}
