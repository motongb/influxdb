package testing

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	platform "github.com/influxdata/influxdb"
	"github.com/influxdata/influxdb/mock"
)

const (
	checkOneID   = "020f755c3c082000"
	checkTwoID   = "020f755c3c082001"
	checkThreeID = "020f755c3c082002"
)

var checkCmpOptions = cmp.Options{
	cmp.Comparer(func(x, y []byte) bool {
		return bytes.Equal(x, y)
	}),
	cmp.Transformer("Sort", func(in []*platform.Check) []*platform.Check {
		out := append([]*platform.Check(nil), in...) // Copy input to avoid mutating it
		sort.Slice(out, func(i, j int) bool {
			return out[i].ID.String() > out[j].ID.String()
		})
		return out
	}),
}

// CheckFields will include the IDGenerator, and checks
type CheckFields struct {
	IDGenerator   platform.IDGenerator
	TimeGenerator platform.TimeGenerator
	Checks        []*platform.Check
	Organizations []*platform.Organization
}

type checkServiceF func(
	init func(CheckFields, *testing.T) (platform.CheckService, string, func()),
	t *testing.T,
)

// CheckService tests all the service functions.
func CheckService(
	init func(CheckFields, *testing.T) (platform.CheckService, string, func()),
	t *testing.T,
) {
	tests := []struct {
		name string
		fn   checkServiceF
	}{
		{
			name: "CreateCheck",
			fn:   CreateCheck,
		},
		{
			name: "FindCheckByID",
			fn:   FindCheckByID,
		},
		{
			name: "FindChecks",
			fn:   FindChecks,
		},
		{
			name: "FindCheck",
			fn:   FindCheck,
		},
		{
			name: "UpdateCheck",
			fn:   UpdateCheck,
		},
		{
			name: "DeleteCheck",
			fn:   DeleteCheck,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fn(init, t)
		})
	}
}

// CreateCheck testing
func CreateCheck(
	init func(CheckFields, *testing.T) (platform.CheckService, string, func()),
	t *testing.T,
) {
	type args struct {
		check *platform.Check
	}
	type wants struct {
		err    error
		checks []*platform.Check
	}

	tests := []struct {
		name   string
		fields CheckFields
		args   args
		wants  wants
	}{
		{
			name: "create checks with empty set",
			fields: CheckFields{
				IDGenerator:   mock.NewIDGenerator(checkOneID, t),
				TimeGenerator: mock.TimeGenerator{FakeValue: time.Date(2006, 5, 4, 1, 2, 3, 0, time.UTC)},
				Checks:        []*platform.Check{},
				Organizations: []*platform.Organization{
					{
						Name: "theorg",
						ID:   MustIDBase16(orgOneID),
					},
				},
			},
			args: args{
				check: &platform.Check{
					Name:           "name1",
					OrganizationID: MustIDBase16(orgOneID),
					Description:    "desc1",
				},
			},
			wants: wants{
				checks: []*platform.Check{
					{
						Name:           "name1",
						ID:             MustIDBase16(checkOneID),
						OrganizationID: MustIDBase16(orgOneID),
						Description:    "desc1",
						CRUDLog: platform.CRUDLog{
							CreatedAt: time.Date(2006, 5, 4, 1, 2, 3, 0, time.UTC),
							UpdatedAt: time.Date(2006, 5, 4, 1, 2, 3, 0, time.UTC),
						},
					},
				},
			},
		},
		{
			name: "basic create check",
			fields: CheckFields{
				IDGenerator: &mock.IDGenerator{
					IDFn: func() platform.ID {
						return MustIDBase16(checkTwoID)
					},
				},
				TimeGenerator: mock.TimeGenerator{FakeValue: time.Date(2006, 5, 4, 1, 2, 3, 0, time.UTC)},
				Checks: []*platform.Check{
					{
						ID:             MustIDBase16(checkOneID),
						Name:           "check1",
						OrganizationID: MustIDBase16(orgOneID),
					},
				},
				Organizations: []*platform.Organization{
					{
						Name: "theorg",
						ID:   MustIDBase16(orgOneID),
					},
					{
						Name: "otherorg",
						ID:   MustIDBase16(orgTwoID),
					},
				},
			},
			args: args{
				check: &platform.Check{
					Name:           "check2",
					OrganizationID: MustIDBase16(orgTwoID),
				},
			},
			wants: wants{
				checks: []*platform.Check{
					{
						ID:             MustIDBase16(checkOneID),
						Name:           "check1",
						OrganizationID: MustIDBase16(orgOneID),
					},
					{
						ID:             MustIDBase16(checkTwoID),
						Name:           "check2",
						OrganizationID: MustIDBase16(orgTwoID),
						CRUDLog: platform.CRUDLog{
							CreatedAt: time.Date(2006, 5, 4, 1, 2, 3, 0, time.UTC),
							UpdatedAt: time.Date(2006, 5, 4, 1, 2, 3, 0, time.UTC),
						},
					},
				},
			},
		},
		{
			name: "names should be unique within an organization",
			fields: CheckFields{
				IDGenerator: &mock.IDGenerator{
					IDFn: func() platform.ID {
						return MustIDBase16(checkTwoID)
					},
				},
				TimeGenerator: mock.TimeGenerator{FakeValue: time.Date(2006, 5, 4, 1, 2, 3, 0, time.UTC)},
				Checks: []*platform.Check{
					{
						ID:             MustIDBase16(checkOneID),
						Name:           "check1",
						OrganizationID: MustIDBase16(orgOneID),
					},
				},
				Organizations: []*platform.Organization{
					{
						Name: "theorg",
						ID:   MustIDBase16(orgOneID),
					},
					{
						Name: "otherorg",
						ID:   MustIDBase16(orgTwoID),
					},
				},
			},
			args: args{
				check: &platform.Check{
					Name:           "check1",
					OrganizationID: MustIDBase16(orgOneID),
				},
			},
			wants: wants{
				checks: []*platform.Check{
					{
						ID:             MustIDBase16(checkOneID),
						Name:           "check1",
						OrganizationID: MustIDBase16(orgOneID),
					},
				},
				err: &platform.Error{
					Code: platform.EConflict,
					Op:   platform.OpCreateCheck,
					Msg:  fmt.Sprintf("check with name check1 already exists"),
				},
			},
		},
		{
			name: "names should not be unique across organizations",
			fields: CheckFields{
				IDGenerator: &mock.IDGenerator{
					IDFn: func() platform.ID {
						return MustIDBase16(checkTwoID)
					},
				},
				TimeGenerator: mock.TimeGenerator{FakeValue: time.Date(2006, 5, 4, 1, 2, 3, 0, time.UTC)},
				Organizations: []*platform.Organization{
					{
						Name: "theorg",
						ID:   MustIDBase16(orgOneID),
					},
					{
						Name: "otherorg",
						ID:   MustIDBase16(orgTwoID),
					},
				},
				Checks: []*platform.Check{
					{
						ID:             MustIDBase16(checkOneID),
						Name:           "check1",
						OrganizationID: MustIDBase16(orgOneID),
					},
				},
			},
			args: args{
				check: &platform.Check{
					Name:           "check1",
					OrganizationID: MustIDBase16(orgTwoID),
				},
			},
			wants: wants{
				checks: []*platform.Check{
					{
						ID:             MustIDBase16(checkOneID),
						Name:           "check1",
						OrganizationID: MustIDBase16(orgOneID),
					},
					{
						ID:             MustIDBase16(checkTwoID),
						Name:           "check1",
						OrganizationID: MustIDBase16(orgTwoID),
						CRUDLog: platform.CRUDLog{
							CreatedAt: time.Date(2006, 5, 4, 1, 2, 3, 0, time.UTC),
							UpdatedAt: time.Date(2006, 5, 4, 1, 2, 3, 0, time.UTC),
						},
					},
				},
			},
		},
		{
			name: "create check with orgID not exist",
			fields: CheckFields{
				IDGenerator:   mock.NewIDGenerator(checkOneID, t),
				TimeGenerator: mock.TimeGenerator{FakeValue: time.Date(2006, 5, 4, 1, 2, 3, 0, time.UTC)},
				Checks:        []*platform.Check{},
				Organizations: []*platform.Organization{},
			},
			args: args{
				check: &platform.Check{
					Name:           "name1",
					OrganizationID: MustIDBase16(orgOneID),
				},
			},
			wants: wants{
				checks: []*platform.Check{},
				err: &platform.Error{
					Code: platform.ENotFound,
					Msg:  "organization not found",
					Op:   platform.OpCreateCheck,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, opPrefix, done := init(tt.fields, t)
			defer done()
			ctx := context.Background()
			err := s.CreateCheck(ctx, tt.args.check)
			diffPlatformErrors(tt.name, err, tt.wants.err, opPrefix, t)

			// Delete only newly created checks - ie., with a not nil ID
			// if tt.args.check.ID.Valid() {
			defer s.DeleteCheck(ctx, tt.args.check.ID)
			// }

			checks, _, err := s.FindChecks(ctx, platform.CheckFilter{})
			if err != nil {
				t.Fatalf("failed to retrieve checks: %v", err)
			}
			if diff := cmp.Diff(checks, tt.wants.checks, checkCmpOptions...); diff != "" {
				t.Errorf("checks are different -got/+want\ndiff %s", diff)
			}
		})
	}
}

// FindCheckByID testing
func FindCheckByID(
	init func(CheckFields, *testing.T) (platform.CheckService, string, func()),
	t *testing.T,
) {
	type args struct {
		id platform.ID
	}
	type wants struct {
		err   error
		check *platform.Check
	}

	tests := []struct {
		name   string
		fields CheckFields
		args   args
		wants  wants
	}{
		{
			name: "basic find check by id",
			fields: CheckFields{
				Checks: []*platform.Check{
					{
						ID:             MustIDBase16(checkOneID),
						OrganizationID: MustIDBase16(orgOneID),
						Name:           "check1",
					},
					{
						ID:             MustIDBase16(checkTwoID),
						OrganizationID: MustIDBase16(orgOneID),
						Name:           "check2",
					},
				},
				Organizations: []*platform.Organization{
					{
						Name: "theorg",
						ID:   MustIDBase16(orgOneID),
					},
				},
			},
			args: args{
				id: MustIDBase16(checkTwoID),
			},
			wants: wants{
				check: &platform.Check{
					ID:             MustIDBase16(checkTwoID),
					OrganizationID: MustIDBase16(orgOneID),
					Name:           "check2",
				},
			},
		},
		{
			name: "find check by id not exist",
			fields: CheckFields{
				Checks: []*platform.Check{
					{
						ID:             MustIDBase16(checkOneID),
						OrganizationID: MustIDBase16(orgOneID),
						Name:           "check1",
					},
					{
						ID:             MustIDBase16(checkTwoID),
						OrganizationID: MustIDBase16(orgOneID),
						Name:           "check2",
					},
				},
				Organizations: []*platform.Organization{
					{
						Name: "theorg",
						ID:   MustIDBase16(orgOneID),
					},
				},
			},
			args: args{
				id: MustIDBase16(threeID),
			},
			wants: wants{
				err: &platform.Error{
					Code: platform.ENotFound,
					Op:   platform.OpFindCheckByID,
					Msg:  "check not found",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, opPrefix, done := init(tt.fields, t)
			defer done()
			ctx := context.Background()

			check, err := s.FindCheckByID(ctx, tt.args.id)
			diffPlatformErrors(tt.name, err, tt.wants.err, opPrefix, t)

			if diff := cmp.Diff(check, tt.wants.check, checkCmpOptions...); diff != "" {
				t.Errorf("check is different -got/+want\ndiff %s", diff)
			}
		})
	}
}

// FindChecks testing
func FindChecks(
	init func(CheckFields, *testing.T) (platform.CheckService, string, func()),
	t *testing.T,
) {
	type args struct {
		ID             platform.ID
		name           string
		organization   string
		organizationID platform.ID
		findOptions    platform.FindOptions
	}

	type wants struct {
		checks []*platform.Check
		err    error
	}
	tests := []struct {
		name   string
		fields CheckFields
		args   args
		wants  wants
	}{
		{
			name: "find all checks",
			fields: CheckFields{
				Organizations: []*platform.Organization{
					{
						Name: "theorg",
						ID:   MustIDBase16(orgOneID),
					},
					{
						Name: "otherorg",
						ID:   MustIDBase16(orgTwoID),
					},
				},
				Checks: []*platform.Check{
					{
						ID:             MustIDBase16(checkOneID),
						OrganizationID: MustIDBase16(orgOneID),
						Name:           "abc",
					},
					{
						ID:             MustIDBase16(checkTwoID),
						OrganizationID: MustIDBase16(orgTwoID),
						Name:           "xyz",
					},
				},
			},
			args: args{},
			wants: wants{
				checks: []*platform.Check{
					{
						ID:             MustIDBase16(checkOneID),
						OrganizationID: MustIDBase16(orgOneID),
						Name:           "abc",
					},
					{
						ID:             MustIDBase16(checkTwoID),
						OrganizationID: MustIDBase16(orgTwoID),
						Name:           "xyz",
					},
				},
			},
		},
		{
			name: "find all checks by offset and limit",
			fields: CheckFields{
				Organizations: []*platform.Organization{
					{
						Name: "theorg",
						ID:   MustIDBase16(orgOneID),
					},
				},
				Checks: []*platform.Check{
					{
						ID:             MustIDBase16(checkOneID),
						OrganizationID: MustIDBase16(orgOneID),
						Name:           "abc",
					},
					{
						ID:             MustIDBase16(checkTwoID),
						OrganizationID: MustIDBase16(orgOneID),
						Name:           "def",
					},
					{
						ID:             MustIDBase16(checkThreeID),
						OrganizationID: MustIDBase16(orgOneID),
						Name:           "xyz",
					},
				},
			},
			args: args{
				findOptions: platform.FindOptions{
					Offset: 1,
					Limit:  1,
				},
			},
			wants: wants{
				checks: []*platform.Check{
					{
						ID:             MustIDBase16(checkTwoID),
						OrganizationID: MustIDBase16(orgOneID),
						Name:           "def",
					},
				},
			},
		},
		{
			name: "find all checks by descending",
			fields: CheckFields{
				Organizations: []*platform.Organization{
					{
						Name: "theorg",
						ID:   MustIDBase16(orgOneID),
					},
				},
				Checks: []*platform.Check{
					{
						ID:             MustIDBase16(checkOneID),
						OrganizationID: MustIDBase16(orgOneID),
						Name:           "abc",
					},
					{
						ID:             MustIDBase16(checkTwoID),
						OrganizationID: MustIDBase16(orgOneID),
						Name:           "def",
					},
					{
						ID:             MustIDBase16(checkThreeID),
						OrganizationID: MustIDBase16(orgOneID),
						Name:           "xyz",
					},
				},
			},
			args: args{
				findOptions: platform.FindOptions{
					Offset:     1,
					Descending: true,
				},
			},
			wants: wants{
				checks: []*platform.Check{
					{
						ID:             MustIDBase16(checkTwoID),
						OrganizationID: MustIDBase16(orgOneID),
						Name:           "def",
					},
					{
						ID:             MustIDBase16(checkOneID),
						OrganizationID: MustIDBase16(orgOneID),
						Name:           "abc",
					},
				},
			},
		},
		{
			name: "find checks by organization name",
			fields: CheckFields{
				Organizations: []*platform.Organization{
					{
						Name: "theorg",
						ID:   MustIDBase16(orgOneID),
					},
					{
						Name: "otherorg",
						ID:   MustIDBase16(orgTwoID),
					},
				},
				Checks: []*platform.Check{
					{
						ID:             MustIDBase16(checkOneID),
						OrganizationID: MustIDBase16(orgOneID),
						Name:           "abc",
					},
					{
						ID:             MustIDBase16(checkTwoID),
						OrganizationID: MustIDBase16(orgTwoID),
						Name:           "xyz",
					},
					{
						ID:             MustIDBase16(checkThreeID),
						OrganizationID: MustIDBase16(orgOneID),
						Name:           "123",
					},
				},
			},
			args: args{
				organization: "theorg",
			},
			wants: wants{
				checks: []*platform.Check{
					{
						ID:             MustIDBase16(checkOneID),
						OrganizationID: MustIDBase16(orgOneID),
						Name:           "abc",
					},
					{
						ID:             MustIDBase16(checkThreeID),
						OrganizationID: MustIDBase16(orgOneID),
						Name:           "123",
					},
				},
			},
		},
		{
			name: "find checks by organization id",
			fields: CheckFields{
				Organizations: []*platform.Organization{
					{
						Name: "theorg",
						ID:   MustIDBase16(orgOneID),
					},
					{
						Name: "otherorg",
						ID:   MustIDBase16(orgTwoID),
					},
				},
				Checks: []*platform.Check{
					{
						ID:             MustIDBase16(checkOneID),
						OrganizationID: MustIDBase16(orgOneID),
						Name:           "abc",
					},
					{
						ID:             MustIDBase16(checkTwoID),
						OrganizationID: MustIDBase16(orgTwoID),
						Name:           "xyz",
					},
					{
						ID:             MustIDBase16(checkThreeID),
						OrganizationID: MustIDBase16(orgOneID),
						Name:           "123",
					},
				},
			},
			args: args{
				organizationID: MustIDBase16(orgOneID),
			},
			wants: wants{
				checks: []*platform.Check{
					{
						ID:             MustIDBase16(checkOneID),
						OrganizationID: MustIDBase16(orgOneID),
						Name:           "abc",
					},
					{
						ID:             MustIDBase16(checkThreeID),
						OrganizationID: MustIDBase16(orgOneID),
						Name:           "123",
					},
				},
			},
		},
		{
			name: "find check by name",
			fields: CheckFields{
				Organizations: []*platform.Organization{
					{
						Name: "theorg",
						ID:   MustIDBase16(orgOneID),
					},
				},
				Checks: []*platform.Check{
					{
						ID:             MustIDBase16(checkOneID),
						OrganizationID: MustIDBase16(orgOneID),
						Name:           "abc",
					},
					{
						ID:             MustIDBase16(checkTwoID),
						OrganizationID: MustIDBase16(orgOneID),
						Name:           "xyz",
					},
				},
			},
			args: args{
				name: "xyz",
			},
			wants: wants{
				checks: []*platform.Check{
					{
						ID:             MustIDBase16(checkTwoID),
						OrganizationID: MustIDBase16(orgOneID),
						Name:           "xyz",
					},
				},
			},
		},
		{
			name: "missing check returns no checks",
			fields: CheckFields{
				Organizations: []*platform.Organization{
					{
						Name: "theorg",
						ID:   MustIDBase16(orgOneID),
					},
				},
				Checks: []*platform.Check{},
			},
			args: args{
				name: "xyz",
			},
			wants: wants{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, opPrefix, done := init(tt.fields, t)
			defer done()
			ctx := context.Background()

			filter := platform.CheckFilter{}
			if tt.args.ID.Valid() {
				filter.ID = &tt.args.ID
			}
			if tt.args.organizationID.Valid() {
				filter.OrganizationID = &tt.args.organizationID
			}
			if tt.args.organization != "" {
				filter.Org = &tt.args.organization
			}
			if tt.args.name != "" {
				filter.Name = &tt.args.name
			}

			checks, _, err := s.FindChecks(ctx, filter, tt.args.findOptions)
			diffPlatformErrors(tt.name, err, tt.wants.err, opPrefix, t)

			if diff := cmp.Diff(checks, tt.wants.checks, checkCmpOptions...); diff != "" {
				t.Errorf("checks are different -got/+want\ndiff %s", diff)
			}
		})
	}
}

// DeleteCheck testing
func DeleteCheck(
	init func(CheckFields, *testing.T) (platform.CheckService, string, func()),
	t *testing.T,
) {
	type args struct {
		ID string
	}
	type wants struct {
		err    error
		checks []*platform.Check
	}

	tests := []struct {
		name   string
		fields CheckFields
		args   args
		wants  wants
	}{
		{
			name: "delete checks using exist id",
			fields: CheckFields{
				Organizations: []*platform.Organization{
					{
						Name: "theorg",
						ID:   MustIDBase16(orgOneID),
					},
				},
				Checks: []*platform.Check{
					{
						Name:           "A",
						ID:             MustIDBase16(checkOneID),
						OrganizationID: MustIDBase16(orgOneID),
					},
					{
						Name:           "B",
						ID:             MustIDBase16(checkThreeID),
						OrganizationID: MustIDBase16(orgOneID),
					},
				},
			},
			args: args{
				ID: checkOneID,
			},
			wants: wants{
				checks: []*platform.Check{
					{
						Name:           "B",
						ID:             MustIDBase16(checkThreeID),
						OrganizationID: MustIDBase16(orgOneID),
					},
				},
			},
		},
		{
			name: "delete checks using id that does not exist",
			fields: CheckFields{
				Organizations: []*platform.Organization{
					{
						Name: "theorg",
						ID:   MustIDBase16(orgOneID),
					},
				},
				Checks: []*platform.Check{
					{
						Name:           "A",
						ID:             MustIDBase16(checkOneID),
						OrganizationID: MustIDBase16(orgOneID),
					},
					{
						Name:           "B",
						ID:             MustIDBase16(checkThreeID),
						OrganizationID: MustIDBase16(orgOneID),
					},
				},
			},
			args: args{
				ID: "1234567890654321",
			},
			wants: wants{
				err: &platform.Error{
					Op:   platform.OpDeleteCheck,
					Msg:  "check not found",
					Code: platform.ENotFound,
				},
				checks: []*platform.Check{
					{
						Name:           "A",
						ID:             MustIDBase16(checkOneID),
						OrganizationID: MustIDBase16(orgOneID),
					},
					{
						Name:           "B",
						ID:             MustIDBase16(checkThreeID),
						OrganizationID: MustIDBase16(orgOneID),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, opPrefix, done := init(tt.fields, t)
			defer done()
			ctx := context.Background()
			err := s.DeleteCheck(ctx, MustIDBase16(tt.args.ID))
			diffPlatformErrors(tt.name, err, tt.wants.err, opPrefix, t)

			filter := platform.CheckFilter{}
			checks, _, err := s.FindChecks(ctx, filter)
			if err != nil {
				t.Fatalf("failed to retrieve checks: %v", err)
			}
			if diff := cmp.Diff(checks, tt.wants.checks, checkCmpOptions...); diff != "" {
				t.Errorf("checks are different -got/+want\ndiff %s", diff)
			}
		})
	}
}

// FindCheck testing
func FindCheck(
	init func(CheckFields, *testing.T) (platform.CheckService, string, func()),
	t *testing.T,
) {
	type args struct {
		name           string
		organizationID platform.ID
	}

	type wants struct {
		check *platform.Check
		err   error
	}

	tests := []struct {
		name   string
		fields CheckFields
		args   args
		wants  wants
	}{
		{
			name: "find check by name",
			fields: CheckFields{
				Organizations: []*platform.Organization{
					{
						Name: "theorg",
						ID:   MustIDBase16(orgOneID),
					},
				},
				Checks: []*platform.Check{
					{
						ID:             MustIDBase16(checkOneID),
						OrganizationID: MustIDBase16(orgOneID),
						Name:           "abc",
					},
					{
						ID:             MustIDBase16(checkTwoID),
						OrganizationID: MustIDBase16(orgOneID),
						Name:           "xyz",
					},
				},
			},
			args: args{
				name:           "abc",
				organizationID: MustIDBase16(orgOneID),
			},
			wants: wants{
				check: &platform.Check{
					ID:             MustIDBase16(checkOneID),
					OrganizationID: MustIDBase16(orgOneID),
					Name:           "abc",
				},
			},
		},
		{
			name: "missing check returns error",
			fields: CheckFields{
				Organizations: []*platform.Organization{
					{
						Name: "theorg",
						ID:   MustIDBase16(orgOneID),
					},
				},
				Checks: []*platform.Check{},
			},
			args: args{
				name:           "xyz",
				organizationID: MustIDBase16(orgOneID),
			},
			wants: wants{
				err: &platform.Error{
					Code: platform.ENotFound,
					Op:   platform.OpFindCheck,
					Msg:  "check \"xyz\" not found",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, opPrefix, done := init(tt.fields, t)
			defer done()
			ctx := context.Background()
			filter := platform.CheckFilter{}
			if tt.args.name != "" {
				filter.Name = &tt.args.name
			}
			if tt.args.organizationID.Valid() {
				filter.OrganizationID = &tt.args.organizationID
			}

			check, err := s.FindCheck(ctx, filter)
			diffPlatformErrors(tt.name, err, tt.wants.err, opPrefix, t)

			if diff := cmp.Diff(check, tt.wants.check, checkCmpOptions...); diff != "" {
				t.Errorf("checks are different -got/+want\ndiff %s", diff)
			}
		})
	}
}

// UpdateCheck testing
func UpdateCheck(
	init func(CheckFields, *testing.T) (platform.CheckService, string, func()),
	t *testing.T,
) {
	type args struct {
		name        string
		id          platform.ID
		retention   int
		description *string
	}
	type wants struct {
		err   error
		check *platform.Check
	}

	tests := []struct {
		name   string
		fields CheckFields
		args   args
		wants  wants
	}{
		{
			name: "update name",
			fields: CheckFields{
				TimeGenerator: mock.TimeGenerator{FakeValue: time.Date(2006, 5, 4, 1, 2, 3, 0, time.UTC)},
				Organizations: []*platform.Organization{
					{
						Name: "theorg",
						ID:   MustIDBase16(orgOneID),
					},
				},
				Checks: []*platform.Check{
					{
						ID:             MustIDBase16(checkOneID),
						OrganizationID: MustIDBase16(orgOneID),
						Name:           "check1",
					},
					{
						ID:             MustIDBase16(checkTwoID),
						OrganizationID: MustIDBase16(orgOneID),
						Name:           "check2",
					},
				},
			},
			args: args{
				id:   MustIDBase16(checkOneID),
				name: "changed",
			},
			wants: wants{
				check: &platform.Check{
					ID:             MustIDBase16(checkOneID),
					OrganizationID: MustIDBase16(orgOneID),
					Name:           "changed",
					CRUDLog: platform.CRUDLog{
						UpdatedAt: time.Date(2006, 5, 4, 1, 2, 3, 0, time.UTC),
					},
				},
			},
		},
		{
			name: "update name unique",
			fields: CheckFields{
				TimeGenerator: mock.TimeGenerator{FakeValue: time.Date(2006, 5, 4, 1, 2, 3, 0, time.UTC)},
				Organizations: []*platform.Organization{
					{
						Name: "theorg",
						ID:   MustIDBase16(orgOneID),
					},
				},
				Checks: []*platform.Check{
					{
						ID:             MustIDBase16(checkOneID),
						OrganizationID: MustIDBase16(orgOneID),
						Name:           "check1",
					},
					{
						ID:             MustIDBase16(checkTwoID),
						OrganizationID: MustIDBase16(orgOneID),
						Name:           "check2",
					},
				},
			},
			args: args{
				id:   MustIDBase16(checkOneID),
				name: "check2",
			},
			wants: wants{
				err: &platform.Error{
					Code: platform.EConflict,
					Msg:  "check name is not unique",
				},
			},
		},
		{
			name: "update description",
			fields: CheckFields{
				TimeGenerator: mock.TimeGenerator{FakeValue: time.Date(2006, 5, 4, 1, 2, 3, 0, time.UTC)},
				Organizations: []*platform.Organization{
					{
						Name: "theorg",
						ID:   MustIDBase16(orgOneID),
					},
				},
				Checks: []*platform.Check{
					{
						ID:             MustIDBase16(checkOneID),
						OrganizationID: MustIDBase16(orgOneID),
						Name:           "check1",
					},
					{
						ID:             MustIDBase16(checkTwoID),
						OrganizationID: MustIDBase16(orgOneID),
						Name:           "check2",
					},
				},
			},
			args: args{
				id:          MustIDBase16(checkOneID),
				description: stringPtr("desc1"),
			},
			wants: wants{
				check: &platform.Check{
					ID:             MustIDBase16(checkOneID),
					OrganizationID: MustIDBase16(orgOneID),
					Name:           "check1",
					Description:    "desc1",
					CRUDLog: platform.CRUDLog{
						UpdatedAt: time.Date(2006, 5, 4, 1, 2, 3, 0, time.UTC),
					},
				},
			},
		},
		{
			name: "update retention and name",
			fields: CheckFields{
				TimeGenerator: mock.TimeGenerator{FakeValue: time.Date(2006, 5, 4, 1, 2, 3, 0, time.UTC)},
				Organizations: []*platform.Organization{
					{
						Name: "theorg",
						ID:   MustIDBase16(orgOneID),
					},
				},
				Checks: []*platform.Check{
					{
						ID:             MustIDBase16(checkOneID),
						OrganizationID: MustIDBase16(orgOneID),
						Name:           "check1",
					},
					{
						ID:             MustIDBase16(checkTwoID),
						OrganizationID: MustIDBase16(orgOneID),
						Name:           "check2",
					},
				},
			},
			args: args{
				id:        MustIDBase16(checkTwoID),
				retention: 101,
				name:      "changed",
			},
			wants: wants{
				check: &platform.Check{
					ID:             MustIDBase16(checkTwoID),
					OrganizationID: MustIDBase16(orgOneID),
					Name:           "changed",
					CRUDLog: platform.CRUDLog{
						UpdatedAt: time.Date(2006, 5, 4, 1, 2, 3, 0, time.UTC),
					},
				},
			},
		},
		{
			name: "update retention and same name",
			fields: CheckFields{
				TimeGenerator: mock.TimeGenerator{FakeValue: time.Date(2006, 5, 4, 1, 2, 3, 0, time.UTC)},
				Organizations: []*platform.Organization{
					{
						Name: "theorg",
						ID:   MustIDBase16(orgOneID),
					},
				},
				Checks: []*platform.Check{
					{
						ID:             MustIDBase16(checkOneID),
						OrganizationID: MustIDBase16(orgOneID),
						Name:           "check1",
					},
					{
						ID:             MustIDBase16(checkTwoID),
						OrganizationID: MustIDBase16(orgOneID),
						Name:           "check2",
					},
				},
			},
			args: args{
				id:        MustIDBase16(checkTwoID),
				retention: 101,
				name:      "check2",
			},
			wants: wants{
				check: &platform.Check{
					ID:             MustIDBase16(checkTwoID),
					OrganizationID: MustIDBase16(orgOneID),
					Name:           "check2",
					CRUDLog: platform.CRUDLog{
						UpdatedAt: time.Date(2006, 5, 4, 1, 2, 3, 0, time.UTC),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, opPrefix, done := init(tt.fields, t)
			defer done()
			ctx := context.Background()

			upd := platform.CheckUpdate{}

			upd.Description = tt.args.description

			check, err := s.UpdateCheck(ctx, tt.args.id, upd)
			diffPlatformErrors(tt.name, err, tt.wants.err, opPrefix, t)

			if diff := cmp.Diff(check, tt.wants.check, checkCmpOptions...); diff != "" {
				t.Errorf("check is different -got/+want\ndiff %s", diff)
			}
		})
	}
}
