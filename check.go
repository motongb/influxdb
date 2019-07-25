package influxdb

import "context"

const (
	CheckDefaultPageSize = 100
	CheckMaxPageSize     = 500
)

// Check represents the information required to generate a periodic check task.
type Check struct {
	ID                    ID         `json:"id"`
	OrganizationID        ID         `json:"orgID,omitempty"`
	TaskID                ID         `json:"-"` // the generated task
	Tags                  []CheckTag `json:"tags"`
	StatusMessageTemplate string     `json:"statusMessageTemplate"`
	Query                 string     `json:"query"`

	// These are necessary for creating/recreating the underlying task.
	// They should be treated as the source of truth, as they'll effectively overwrite any changes
	// made directly to the task if the task is regenerated.

	// AuthorizationID ID     `json:"authorizationID"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	// Status          string `json:"status"`
	// Every           string `json:"every,omitempty"`
	// Cron            string `json:"cron,omitempty"`
	// Offset          string `json:"offset,omitempty"`

	// Read only. This shouldn't stored on the object, it's decorated on
	// for populating the API response.
	Task Task `json:"task,omitempty"`

	CRUDLog
	// Properties CheckProperties
}

// ops for checks error
var (
	OpFindCheckByID = "FindCheckByID"
	OpFindCheck     = "FindCheck"
	OpFindChecks    = "FindChecks"
	OpCreateCheck   = "CreateCheck"
	OpUpdateCheck   = "UpdateCheck"
	OpDeleteCheck   = "DeleteCheck"
)

// CheckTag is a tag k/v pair used when a check writes to the system bucket.
type CheckTag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Valid returns an error if the checktag is missing fields
func (t *CheckTag) Valid() error {
	if t.Key == "" || t.Value == "" {
		return &Error{
			Code: EInvalid,
			Msg:  "checktag must contain a key and a value",
		}
	}
	return nil
}

// CheckService represents a service for managing checks.
type CheckService interface {
	// FindCheckByID returns a single check by ID.
	FindCheckByID(ctx context.Context, id ID) (*Check, error)

	// FindCheck returns the first check that matches filter.
	FindCheck(ctx context.Context, filter CheckFilter) (*Check, error)

	// FindChecks returns a list of checks that match filter and the total count of matching checkns.
	// Additional options provide pagination & sorting.
	FindChecks(ctx context.Context, filter CheckFilter, opt ...FindOptions) ([]*Check, int, error)

	// CreateCheck creates a new check and sets b.ID with the new identifier.
	CreateCheck(ctx context.Context, c *Check) error

	// UpdateCheck updates a single bucket with changeset.
	// Returns the new check state after update.
	UpdateCheck(ctx context.Context, id ID, upd CheckUpdate) (*Check, error)

	// DeleteCheck removes a bucket by ID.
	DeleteCheck(ctx context.Context, id ID) error
}

// CheckUpdate are properties than can be updated on a check
type CheckUpdate struct {
	StatusMessageTemplate *string    `json:"statusMessageTemplate"`
	Tags                  []CheckTag `json:"tags"`
	Query                 *string    `json:"flux,omitempty"`

	// For the task
	Status      *string `json:"status,omitempty"`
	Description *string `json:"description,omitempty"`
}

// CheckFilter represents a set of filters that restrict the returned results.
type CheckFilter struct {
	ID             *ID
	Name           *string
	OrganizationID *ID
	Org            *string
	Limit          int
}

// func (c *Check) UnJSON(data []byte) error {
// 	return nil
// }
//
// func (c *Check) MarshalJSON() ([]byte, error) {
// 	return nil, nil
// }
//
// func UnmarshalCheckPropertiesJSON(b []byte) (CheckProperties, error) {
// 	var v struct {
// 		B json.RawMessage `json:"properties"`
// 	}
//
// 	if err := json.Unmarshal(b, &v); err != nil {
// 		return nil, err
// 	}
//
// 	// if len(v.B)
//
// 	// var t struct {
// 	//   Type: string `json:"type"`
// 	// }
// 	return nil, nil
// }
//
// type CheckProperties interface {
// 	GetType() string
// }
