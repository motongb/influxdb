package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	platform "github.com/influxdata/influxdb"
	"github.com/influxdata/influxdb/inmem"
	"github.com/influxdata/influxdb/mock"
	platformtesting "github.com/influxdata/influxdb/testing"
	"github.com/julienschmidt/httprouter"
	"go.uber.org/zap"
)

// NewMockCheckBackend returns a CheckBackend with mock services.
func NewMockCheckBackend() *CheckBackend {
	return &CheckBackend{
		Logger: zap.NewNop().With(zap.String("handler", "check")),

		CheckService:               mock.NewCheckService(),
		UserResourceMappingService: mock.NewUserResourceMappingService(),
		LabelService:               mock.NewLabelService(),
		UserService:                mock.NewUserService(),
		OrganizationService:        mock.NewOrganizationService(),
	}
}

func TestService_handleGetChecks(t *testing.T) {
	type fields struct {
		CheckService platform.CheckService
		LabelService platform.LabelService
	}
	type args struct {
		queryParams map[string][]string
	}
	type wants struct {
		statusCode  int
		contentType string
		body        string
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		wants  wants
	}{
		{
			name: "get all checks",
			fields: fields{
				&mock.CheckService{
					FindChecksFn: func(ctx context.Context, filter platform.CheckFilter, opts ...platform.FindOptions) ([]*platform.Check, int, error) {
						return []*platform.Check{
							{
								ID:              platformtesting.MustIDBase16("0b501e7e557ab1ed"),
								Name:            "hello",
								OrgID:           platformtesting.MustIDBase16("50f7ba1150f7ba11"),
								RetentionPeriod: 2 * time.Second,
							},
							{
								ID:              platformtesting.MustIDBase16("c0175f0077a77005"),
								Name:            "example",
								OrgID:           platformtesting.MustIDBase16("7e55e118dbabb1ed"),
								RetentionPeriod: 24 * time.Hour,
							},
						}, 2, nil
					},
				},
				&mock.LabelService{
					FindResourceLabelsFn: func(ctx context.Context, f platform.LabelMappingFilter) ([]*platform.Label, error) {
						labels := []*platform.Label{
							{
								ID:   platformtesting.MustIDBase16("fc3dc670a4be9b9a"),
								Name: "label",
								Properties: map[string]string{
									"color": "fff000",
								},
							},
						}
						return labels, nil
					},
				},
			},
			args: args{
				map[string][]string{
					"limit": {"1"},
				},
			},
			wants: wants{
				statusCode:  http.StatusOK,
				contentType: "application/json; charset=utf-8",
				body: `
{
  "links": {
    "self": "/api/v2/checks?descending=false&limit=1&offset=0",
    "next": "/api/v2/checks?descending=false&limit=1&offset=1"
  },
  "checks": [
    {
      "links": {
        "org": "/api/v2/orgs/50f7ba1150f7ba11",
        "self": "/api/v2/checks/0b501e7e557ab1ed",
        "logs": "/api/v2/checks/0b501e7e557ab1ed/logs",
        "labels": "/api/v2/checks/0b501e7e557ab1ed/labels",
        "owners": "/api/v2/checks/0b501e7e557ab1ed/owners",
        "members": "/api/v2/checks/0b501e7e557ab1ed/members",
        "write": "/api/v2/write?org=50f7ba1150f7ba11&check=0b501e7e557ab1ed"
	  },
	  "createdAt": "0001-01-01T00:00:00Z",
	  "updatedAt": "0001-01-01T00:00:00Z",
      "id": "0b501e7e557ab1ed",
      "orgID": "50f7ba1150f7ba11",
      "name": "hello",
      "retentionRules": [{"type": "expire", "everySeconds": 2}],
			"labels": [
        {
          "id": "fc3dc670a4be9b9a",
          "name": "label",
          "properties": {
            "color": "fff000"
          }
        }
      ]
    },
    {
      "links": {
        "org": "/api/v2/orgs/7e55e118dbabb1ed",
        "self": "/api/v2/checks/c0175f0077a77005",
        "logs": "/api/v2/checks/c0175f0077a77005/logs",
        "labels": "/api/v2/checks/c0175f0077a77005/labels",
        "members": "/api/v2/checks/c0175f0077a77005/members",
        "owners": "/api/v2/checks/c0175f0077a77005/owners",
        "write": "/api/v2/write?org=7e55e118dbabb1ed&check=c0175f0077a77005"
	  },
	  "createdAt": "0001-01-01T00:00:00Z",
	  "updatedAt": "0001-01-01T00:00:00Z",
      "id": "c0175f0077a77005",
      "orgID": "7e55e118dbabb1ed",
      "name": "example",
      "retentionRules": [{"type": "expire", "everySeconds": 86400}],
      "labels": [
        {
          "id": "fc3dc670a4be9b9a",
          "name": "label",
          "properties": {
            "color": "fff000"
          }
        }
      ]
    }
  ]
}
`,
			},
		},
		{
			name: "get all checks when there are none",
			fields: fields{
				&mock.CheckService{
					FindChecksFn: func(ctx context.Context, filter platform.CheckFilter, opts ...platform.FindOptions) ([]*platform.Check, int, error) {
						return []*platform.Check{}, 0, nil
					},
				},
				&mock.LabelService{},
			},
			args: args{
				map[string][]string{
					"limit": {"1"},
				},
			},
			wants: wants{
				statusCode:  http.StatusOK,
				contentType: "application/json; charset=utf-8",
				body: `
{
  "links": {
    "self": "/api/v2/checks?descending=false&limit=1&offset=0"
  },
  "checks": []
}`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checkBackend := NewMockCheckBackend()
			checkBackend.CheckService = tt.fields.CheckService
			checkBackend.LabelService = tt.fields.LabelService
			h := NewCheckHandler(checkBackend)

			r := httptest.NewRequest("GET", "http://any.url", nil)

			qp := r.URL.Query()
			for k, vs := range tt.args.queryParams {
				for _, v := range vs {
					qp.Add(k, v)
				}
			}
			r.URL.RawQuery = qp.Encode()

			w := httptest.NewRecorder()

			h.handleGetChecks(w, r)

			res := w.Result()
			content := res.Header.Get("Content-Type")
			body, _ := ioutil.ReadAll(res.Body)

			if res.StatusCode != tt.wants.statusCode {
				t.Errorf("%q. handleGetChecks() = %v, want %v", tt.name, res.StatusCode, tt.wants.statusCode)
			}
			if tt.wants.contentType != "" && content != tt.wants.contentType {
				t.Errorf("%q. handleGetChecks() = %v, want %v", tt.name, content, tt.wants.contentType)
			}
			if eq, diff, err := jsonEqual(string(body), tt.wants.body); err != nil || tt.wants.body != "" && !eq {
				t.Errorf("%q. handleGetChecks() = ***%v***", tt.name, diff)
			}
		})
	}
}

func TestService_handleGetCheck(t *testing.T) {
	type fields struct {
		CheckService platform.CheckService
	}
	type args struct {
		id string
	}
	type wants struct {
		statusCode  int
		contentType string
		body        string
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		wants  wants
	}{
		{
			name: "get a check by id",
			fields: fields{
				&mock.CheckService{
					FindCheckByIDFn: func(ctx context.Context, id platform.ID) (*platform.Check, error) {
						if id == platformtesting.MustIDBase16("020f755c3c082000") {
							return &platform.Check{
								ID:              platformtesting.MustIDBase16("020f755c3c082000"),
								OrgID:           platformtesting.MustIDBase16("020f755c3c082000"),
								Name:            "hello",
								RetentionPeriod: 30 * time.Second,
							}, nil
						}

						return nil, fmt.Errorf("not found")
					},
				},
			},
			args: args{
				id: "020f755c3c082000",
			},
			wants: wants{
				statusCode:  http.StatusOK,
				contentType: "application/json; charset=utf-8",
				body: `
		{
		  "links": {
		    "org": "/api/v2/orgs/020f755c3c082000",
		    "self": "/api/v2/checks/020f755c3c082000",
		    "logs": "/api/v2/checks/020f755c3c082000/logs",
		    "labels": "/api/v2/checks/020f755c3c082000/labels",
		    "members": "/api/v2/checks/020f755c3c082000/members",
		    "owners": "/api/v2/checks/020f755c3c082000/owners",
		    "write": "/api/v2/write?org=020f755c3c082000&check=020f755c3c082000"
		  },
		  "createdAt": "0001-01-01T00:00:00Z",
		  "updatedAt": "0001-01-01T00:00:00Z",
		  "id": "020f755c3c082000",
		  "orgID": "020f755c3c082000",
		  "name": "hello",
		  "retentionRules": [{"type": "expire", "everySeconds": 30}],
      "labels": []
		}
		`,
			},
		},
		{
			name: "not found",
			fields: fields{
				&mock.CheckService{
					FindCheckByIDFn: func(ctx context.Context, id platform.ID) (*platform.Check, error) {
						return nil, &platform.Error{
							Code: platform.ENotFound,
							Msg:  "check not found",
						}
					},
				},
			},
			args: args{
				id: "020f755c3c082000",
			},
			wants: wants{
				statusCode: http.StatusNotFound,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checkBackend := NewMockCheckBackend()
			checkBackend.HTTPErrorHandler = ErrorHandler(0)
			checkBackend.CheckService = tt.fields.CheckService
			h := NewCheckHandler(checkBackend)

			r := httptest.NewRequest("GET", "http://any.url", nil)

			r = r.WithContext(context.WithValue(
				context.Background(),
				httprouter.ParamsKey,
				httprouter.Params{
					{
						Key:   "id",
						Value: tt.args.id,
					},
				}))

			w := httptest.NewRecorder()

			h.handleGetCheck(w, r)

			res := w.Result()
			content := res.Header.Get("Content-Type")
			body, _ := ioutil.ReadAll(res.Body)
			t.Logf(res.Header.Get("X-Influx-Error"))

			if res.StatusCode != tt.wants.statusCode {
				t.Errorf("%q. handleGetCheck() = %v, want %v", tt.name, res.StatusCode, tt.wants.statusCode)
			}
			if tt.wants.contentType != "" && content != tt.wants.contentType {
				t.Errorf("%q. handleGetCheck() = %v, want %v", tt.name, content, tt.wants.contentType)
			}
			if tt.wants.body != "" {
				if eq, diff, err := jsonEqual(string(body), tt.wants.body); err != nil {
					t.Errorf("%q, handleGetCheck(). error unmarshaling json %v", tt.name, err)
				} else if !eq {
					t.Errorf("%q. handleGetCheck() = ***%s***", tt.name, diff)
				}
			}
		})
	}
}

func TestService_handlePostCheck(t *testing.T) {
	type fields struct {
		CheckService        platform.CheckService
		OrganizationService platform.OrganizationService
	}
	type args struct {
		check *platform.Check
	}
	type wants struct {
		statusCode  int
		contentType string
		body        string
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		wants  wants
	}{
		{
			name: "create a new check",
			fields: fields{
				CheckService: &mock.CheckService{
					CreateCheckFn: func(ctx context.Context, c *platform.Check) error {
						c.ID = platformtesting.MustIDBase16("020f755c3c082000")
						return nil
					},
				},
				OrganizationService: &mock.OrganizationService{
					FindOrganizationF: func(ctx context.Context, f platform.OrganizationFilter) (*platform.Organization, error) {
						return &platform.Organization{ID: platformtesting.MustIDBase16("6f626f7274697320")}, nil
					},
				},
			},
			args: args{
				check: &platform.Check{
					Name:  "hello",
					OrgID: platformtesting.MustIDBase16("6f626f7274697320"),
				},
			},
			wants: wants{
				statusCode:  http.StatusCreated,
				contentType: "application/json; charset=utf-8",
				body: `
{
  "links": {
    "org": "/api/v2/orgs/6f626f7274697320",
    "self": "/api/v2/checks/020f755c3c082000",
    "logs": "/api/v2/checks/020f755c3c082000/logs",
    "labels": "/api/v2/checks/020f755c3c082000/labels",
    "members": "/api/v2/checks/020f755c3c082000/members",
    "owners": "/api/v2/checks/020f755c3c082000/owners",
    "write": "/api/v2/write?org=6f626f7274697320&check=020f755c3c082000"
  },
  "createdAt": "0001-01-01T00:00:00Z",
  "updatedAt": "0001-01-01T00:00:00Z",
  "id": "020f755c3c082000",
  "orgID": "6f626f7274697320",
  "name": "hello",
  "retentionRules": [],
  "labels": []
}
`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checkBackend := NewMockCheckBackend()
			checkBackend.CheckService = tt.fields.CheckService
			checkBackend.OrganizationService = tt.fields.OrganizationService
			h := NewCheckHandler(checkBackend)

			b, err := json.Marshal(newCheck(tt.args.check))
			if err != nil {
				t.Fatalf("failed to unmarshal check: %v", err)
			}

			r := httptest.NewRequest("GET", "http://any.url?org=30", bytes.NewReader(b))
			w := httptest.NewRecorder()

			h.handlePostCheck(w, r)

			res := w.Result()
			content := res.Header.Get("Content-Type")
			body, _ := ioutil.ReadAll(res.Body)

			if res.StatusCode != tt.wants.statusCode {
				t.Errorf("%q. handlePostCheck() = %v, want %v", tt.name, res.StatusCode, tt.wants.statusCode)
			}
			if tt.wants.contentType != "" && content != tt.wants.contentType {
				t.Errorf("%q. handlePostCheck() = %v, want %v", tt.name, content, tt.wants.contentType)
			}
			if tt.wants.body != "" {
				if eq, diff, err := jsonEqual(string(body), tt.wants.body); err != nil {
					t.Errorf("%q, handlePostCheck(). error unmarshaling json %v", tt.name, err)
				} else if !eq {
					t.Errorf("%q. handlePostCheck() = ***%s***", tt.name, diff)
				}
			}
		})
	}
}

func TestService_handleDeleteCheck(t *testing.T) {
	type fields struct {
		CheckService platform.CheckService
	}
	type args struct {
		id string
	}
	type wants struct {
		statusCode  int
		contentType string
		body        string
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		wants  wants
	}{
		{
			name: "remove a check by id",
			fields: fields{
				&mock.CheckService{
					DeleteCheckFn: func(ctx context.Context, id platform.ID) error {
						if id == platformtesting.MustIDBase16("020f755c3c082000") {
							return nil
						}

						return fmt.Errorf("wrong id")
					},
				},
			},
			args: args{
				id: "020f755c3c082000",
			},
			wants: wants{
				statusCode: http.StatusNoContent,
			},
		},
		{
			name: "check not found",
			fields: fields{
				&mock.CheckService{
					DeleteCheckFn: func(ctx context.Context, id platform.ID) error {
						return &platform.Error{
							Code: platform.ENotFound,
							Msg:  "check not found",
						}
					},
				},
			},
			args: args{
				id: "020f755c3c082000",
			},
			wants: wants{
				statusCode: http.StatusNotFound,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checkBackend := NewMockCheckBackend()
			checkBackend.HTTPErrorHandler = ErrorHandler(0)
			checkBackend.CheckService = tt.fields.CheckService
			h := NewCheckHandler(checkBackend)

			r := httptest.NewRequest("GET", "http://any.url", nil)

			r = r.WithContext(context.WithValue(
				context.Background(),
				httprouter.ParamsKey,
				httprouter.Params{
					{
						Key:   "id",
						Value: tt.args.id,
					},
				}))

			w := httptest.NewRecorder()

			h.handleDeleteCheck(w, r)

			res := w.Result()
			content := res.Header.Get("Content-Type")
			body, _ := ioutil.ReadAll(res.Body)

			if res.StatusCode != tt.wants.statusCode {
				t.Errorf("%q. handleDeleteCheck() = %v, want %v", tt.name, res.StatusCode, tt.wants.statusCode)
			}
			if tt.wants.contentType != "" && content != tt.wants.contentType {
				t.Errorf("%q. handleDeleteCheck() = %v, want %v", tt.name, content, tt.wants.contentType)
			}
			if tt.wants.body != "" {
				if eq, diff, err := jsonEqual(string(body), tt.wants.body); err != nil {
					t.Errorf("%q, handleDeleteCheck(). error unmarshaling json %v", tt.name, err)
				} else if !eq {
					t.Errorf("%q. handleDeleteCheck() = ***%s***", tt.name, diff)
				}
			}
		})
	}
}

func TestService_handlePatchCheck(t *testing.T) {
	type fields struct {
		CheckService platform.CheckService
	}
	type args struct {
		id        string
		name      string
		retention time.Duration
	}
	type wants struct {
		statusCode  int
		contentType string
		body        string
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		wants  wants
	}{
		{
			name: "update a check name and retention",
			fields: fields{
				&mock.CheckService{
					UpdateCheckFn: func(ctx context.Context, id platform.ID, upd platform.CheckUpdate) (*platform.Check, error) {
						if id == platformtesting.MustIDBase16("020f755c3c082000") {
							d := &platform.Check{
								ID:    platformtesting.MustIDBase16("020f755c3c082000"),
								Name:  "hello",
								OrgID: platformtesting.MustIDBase16("020f755c3c082000"),
							}

							if upd.Name != nil {
								d.Name = *upd.Name
							}

							if upd.RetentionPeriod != nil {
								d.RetentionPeriod = *upd.RetentionPeriod
							}

							return d, nil
						}

						return nil, fmt.Errorf("not found")
					},
				},
			},
			args: args{
				id:        "020f755c3c082000",
				name:      "example",
				retention: 2 * time.Second,
			},
			wants: wants{
				statusCode:  http.StatusOK,
				contentType: "application/json; charset=utf-8",
				body: `
{
  "links": {
    "org": "/api/v2/orgs/020f755c3c082000",
    "self": "/api/v2/checks/020f755c3c082000",
    "logs": "/api/v2/checks/020f755c3c082000/logs",
    "labels": "/api/v2/checks/020f755c3c082000/labels",
    "members": "/api/v2/checks/020f755c3c082000/members",
    "owners": "/api/v2/checks/020f755c3c082000/owners",
    "write": "/api/v2/write?org=020f755c3c082000&check=020f755c3c082000"
  },
  "createdAt": "0001-01-01T00:00:00Z",
  "updatedAt": "0001-01-01T00:00:00Z",
  "id": "020f755c3c082000",
  "orgID": "020f755c3c082000",
  "name": "example",
  "retentionRules": [{"type": "expire", "everySeconds": 2}],
  "labels": []
}
`,
			},
		},
		{
			name: "check not found",
			fields: fields{
				&mock.CheckService{
					UpdateCheckFn: func(ctx context.Context, id platform.ID, upd platform.CheckUpdate) (*platform.Check, error) {
						return nil, &platform.Error{
							Code: platform.ENotFound,
							Msg:  "check not found",
						}
					},
				},
			},
			args: args{
				id:        "020f755c3c082000",
				name:      "hello",
				retention: time.Second,
			},
			wants: wants{
				statusCode: http.StatusNotFound,
			},
		},
		{
			name: "update check to no retention and new name",
			fields: fields{
				&mock.CheckService{
					UpdateCheckFn: func(ctx context.Context, id platform.ID, upd platform.CheckUpdate) (*platform.Check, error) {
						if id == platformtesting.MustIDBase16("020f755c3c082000") {
							d := &platform.Check{
								ID:    platformtesting.MustIDBase16("020f755c3c082000"),
								Name:  "hello",
								OrgID: platformtesting.MustIDBase16("020f755c3c082000"),
							}

							if upd.Name != nil {
								d.Name = *upd.Name
							}

							if upd.RetentionPeriod != nil {
								d.RetentionPeriod = *upd.RetentionPeriod
							}

							return d, nil
						}

						return nil, fmt.Errorf("not found")
					},
				},
			},
			args: args{
				id:        "020f755c3c082000",
				name:      "check with no retention",
				retention: 0,
			},
			wants: wants{
				statusCode:  http.StatusOK,
				contentType: "application/json; charset=utf-8",
				body: `
{
  "links": {
    "org": "/api/v2/orgs/020f755c3c082000",
    "self": "/api/v2/checks/020f755c3c082000",
    "logs": "/api/v2/checks/020f755c3c082000/logs",
    "labels": "/api/v2/checks/020f755c3c082000/labels",
    "members": "/api/v2/checks/020f755c3c082000/members",
    "owners": "/api/v2/checks/020f755c3c082000/owners",
    "write": "/api/v2/write?org=020f755c3c082000&check=020f755c3c082000"
  },
  "createdAt": "0001-01-01T00:00:00Z",
  "updatedAt": "0001-01-01T00:00:00Z",
  "id": "020f755c3c082000",
  "orgID": "020f755c3c082000",
  "name": "check with no retention",
  "retentionRules": [],
  "labels": []
}
`,
			},
		},
		{
			name: "update retention policy to 'nothing'",
			fields: fields{
				&mock.CheckService{
					UpdateCheckFn: func(ctx context.Context, id platform.ID, upd platform.CheckUpdate) (*platform.Check, error) {
						if id == platformtesting.MustIDBase16("020f755c3c082000") {
							d := &platform.Check{
								ID:    platformtesting.MustIDBase16("020f755c3c082000"),
								Name:  "b1",
								OrgID: platformtesting.MustIDBase16("020f755c3c082000"),
							}

							if upd.Name != nil {
								d.Name = *upd.Name
							}

							if upd.RetentionPeriod != nil {
								d.RetentionPeriod = *upd.RetentionPeriod
							}

							return d, nil
						}

						return nil, &platform.Error{
							Code: platform.ENotFound,
							Msg:  "check not found",
						}
					},
				},
			},
			args: args{
				id:        "020f755c3c082000",
				retention: 0,
			},
			wants: wants{
				statusCode:  http.StatusOK,
				contentType: "application/json; charset=utf-8",
				body: `
{
  "links": {
    "org": "/api/v2/orgs/020f755c3c082000",
    "self": "/api/v2/checks/020f755c3c082000",
    "logs": "/api/v2/checks/020f755c3c082000/logs",
    "labels": "/api/v2/checks/020f755c3c082000/labels",
    "members": "/api/v2/checks/020f755c3c082000/members",
    "owners": "/api/v2/checks/020f755c3c082000/owners",
    "write": "/api/v2/write?org=020f755c3c082000&check=020f755c3c082000"
  },
  "createdAt": "0001-01-01T00:00:00Z",
  "updatedAt": "0001-01-01T00:00:00Z",
  "id": "020f755c3c082000",
  "orgID": "020f755c3c082000",
  "name": "b1",
  "retentionRules": [],
  "labels": []
}
`,
			},
		},
		{
			name: "update a check name with invalid retention policy is an error",
			fields: fields{
				&mock.CheckService{
					UpdateCheckFn: func(ctx context.Context, id platform.ID, upd platform.CheckUpdate) (*platform.Check, error) {
						if id == platformtesting.MustIDBase16("020f755c3c082000") {
							d := &platform.Check{
								ID:    platformtesting.MustIDBase16("020f755c3c082000"),
								Name:  "hello",
								OrgID: platformtesting.MustIDBase16("020f755c3c082000"),
							}

							if upd.Name != nil {
								d.Name = *upd.Name
							}

							if upd.RetentionPeriod != nil {
								d.RetentionPeriod = *upd.RetentionPeriod
							}

							return d, nil
						}

						return nil, &platform.Error{
							Code: platform.ENotFound,
							Msg:  "check not found",
						}
					},
				},
			},
			args: args{
				id:        "020f755c3c082000",
				name:      "example",
				retention: -10,
			},
			wants: wants{
				statusCode: http.StatusUnprocessableEntity,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checkBackend := NewMockCheckBackend()
			checkBackend.HTTPErrorHandler = ErrorHandler(0)
			checkBackend.CheckService = tt.fields.CheckService
			h := NewCheckHandler(checkBackend)

			upd := platform.CheckUpdate{}
			if tt.args.name != "" {
				upd.Name = &tt.args.name
			}

			if tt.args.retention != 0 {
				upd.RetentionPeriod = &tt.args.retention
			}

			b, err := json.Marshal(newCheckUpdate(&upd))
			if err != nil {
				t.Fatalf("failed to unmarshal check update: %v", err)
			}

			r := httptest.NewRequest("GET", "http://any.url", bytes.NewReader(b))

			r = r.WithContext(context.WithValue(
				context.Background(),
				httprouter.ParamsKey,
				httprouter.Params{
					{
						Key:   "id",
						Value: tt.args.id,
					},
				}))

			w := httptest.NewRecorder()

			h.handlePatchCheck(w, r)

			res := w.Result()
			content := res.Header.Get("Content-Type")
			body, _ := ioutil.ReadAll(res.Body)

			if res.StatusCode != tt.wants.statusCode {
				t.Errorf("%q. handlePatchCheck() = %v, want %v %v", tt.name, res.StatusCode, tt.wants.statusCode, w.Header())
			}
			if tt.wants.contentType != "" && content != tt.wants.contentType {
				t.Errorf("%q. handlePatchCheck() = %v, want %v", tt.name, content, tt.wants.contentType)
			}
			if tt.wants.body != "" {
				if eq, diff, err := jsonEqual(string(body), tt.wants.body); err != nil {
					t.Errorf("%q, handlePatchCheck(). error unmarshaling json %v", tt.name, err)
				} else if !eq {
					t.Errorf("%q. handlePatchCheck() = ***%s***", tt.name, diff)
				}
			}
		})
	}
}

func TestService_handlePostCheckMember(t *testing.T) {
	type fields struct {
		UserService platform.UserService
	}
	type args struct {
		checkID string
		user    *platform.User
	}
	type wants struct {
		statusCode  int
		contentType string
		body        string
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		wants  wants
	}{
		{
			name: "add a check member",
			fields: fields{
				UserService: &mock.UserService{
					FindUserByIDFn: func(ctx context.Context, id platform.ID) (*platform.User, error) {
						return &platform.User{
							ID:   id,
							Name: "name",
						}, nil
					},
				},
			},
			args: args{
				checkID: "020f755c3c082000",
				user: &platform.User{
					ID: platformtesting.MustIDBase16("6f626f7274697320"),
				},
			},
			wants: wants{
				statusCode:  http.StatusCreated,
				contentType: "application/json; charset=utf-8",
				body: `
{
  "links": {
    "logs": "/api/v2/users/6f626f7274697320/logs",
    "self": "/api/v2/users/6f626f7274697320"
  },
  "role": "member",
  "id": "6f626f7274697320",
  "name": "name"
}
`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checkBackend := NewMockCheckBackend()
			checkBackend.UserService = tt.fields.UserService
			h := NewCheckHandler(checkBackend)

			b, err := json.Marshal(tt.args.user)
			if err != nil {
				t.Fatalf("failed to marshal user: %v", err)
			}

			path := fmt.Sprintf("/api/v2/checks/%s/members", tt.args.checkID)
			r := httptest.NewRequest("POST", path, bytes.NewReader(b))
			w := httptest.NewRecorder()

			h.ServeHTTP(w, r)

			res := w.Result()
			content := res.Header.Get("Content-Type")
			body, _ := ioutil.ReadAll(res.Body)

			if res.StatusCode != tt.wants.statusCode {
				t.Errorf("%q. handlePostCheckMember() = %v, want %v", tt.name, res.StatusCode, tt.wants.statusCode)
			}
			if tt.wants.contentType != "" && content != tt.wants.contentType {
				t.Errorf("%q. handlePostCheckMember() = %v, want %v", tt.name, content, tt.wants.contentType)
			}
			if eq, diff, err := jsonEqual(string(body), tt.wants.body); err != nil {
				t.Errorf("%q, handlePostCheckMember(). error unmarshaling json %v", tt.name, err)
			} else if tt.wants.body != "" && !eq {
				t.Errorf("%q. handlePostCheckMember() = ***%s***", tt.name, diff)
			}
		})
	}
}

func TestService_handlePostCheckOwner(t *testing.T) {
	type fields struct {
		UserService platform.UserService
	}
	type args struct {
		checkID string
		user    *platform.User
	}
	type wants struct {
		statusCode  int
		contentType string
		body        string
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		wants  wants
	}{
		{
			name: "add a check owner",
			fields: fields{
				UserService: &mock.UserService{
					FindUserByIDFn: func(ctx context.Context, id platform.ID) (*platform.User, error) {
						return &platform.User{
							ID:   id,
							Name: "name",
						}, nil
					},
				},
			},
			args: args{
				checkID: "020f755c3c082000",
				user: &platform.User{
					ID: platformtesting.MustIDBase16("6f626f7274697320"),
				},
			},
			wants: wants{
				statusCode:  http.StatusCreated,
				contentType: "application/json; charset=utf-8",
				body: `
{
  "links": {
    "logs": "/api/v2/users/6f626f7274697320/logs",
    "self": "/api/v2/users/6f626f7274697320"
  },
  "role": "owner",
  "id": "6f626f7274697320",
  "name": "name"
}
`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checkBackend := NewMockCheckBackend()
			checkBackend.UserService = tt.fields.UserService
			h := NewCheckHandler(checkBackend)

			b, err := json.Marshal(tt.args.user)
			if err != nil {
				t.Fatalf("failed to marshal user: %v", err)
			}

			path := fmt.Sprintf("/api/v2/checks/%s/owners", tt.args.checkID)
			r := httptest.NewRequest("POST", path, bytes.NewReader(b))
			w := httptest.NewRecorder()

			h.ServeHTTP(w, r)

			res := w.Result()
			content := res.Header.Get("Content-Type")
			body, _ := ioutil.ReadAll(res.Body)

			if res.StatusCode != tt.wants.statusCode {
				t.Errorf("%q. handlePostCheckOwner() = %v, want %v", tt.name, res.StatusCode, tt.wants.statusCode)
			}
			if tt.wants.contentType != "" && content != tt.wants.contentType {
				t.Errorf("%q. handlePostCheckOwner() = %v, want %v", tt.name, content, tt.wants.contentType)
			}
			if eq, diff, err := jsonEqual(string(body), tt.wants.body); err != nil {
				t.Errorf("%q, handlePostCheckOwner(). error unmarshaling json %v", tt.name, err)
			} else if tt.wants.body != "" && !eq {
				t.Errorf("%q. handlePostCheckOwner() = ***%s***", tt.name, diff)
			}
		})
	}
}

func initCheckService(f platformtesting.CheckFields, t *testing.T) (platform.CheckService, string, func()) {
	svc := inmem.NewService()
	svc.IDGenerator = f.IDGenerator
	svc.TimeGenerator = f.TimeGenerator
	if f.TimeGenerator == nil {
		svc.TimeGenerator = platform.RealTimeGenerator{}
	}

	ctx := context.Background()
	for _, o := range f.Organizations {
		if err := svc.PutOrganization(ctx, o); err != nil {
			t.Fatalf("failed to populate organizations")
		}
	}
	for _, b := range f.Checks {
		if err := svc.PutCheck(ctx, b); err != nil {
			t.Fatalf("failed to populate checks")
		}
	}

	checkBackend := NewMockCheckBackend()
	checkBackend.HTTPErrorHandler = ErrorHandler(0)
	checkBackend.CheckService = svc
	checkBackend.OrganizationService = svc
	handler := NewCheckHandler(checkBackend)
	server := httptest.NewServer(handler)
	client := CheckService{
		Addr:     server.URL,
		OpPrefix: inmem.OpPrefix,
	}
	done := server.Close

	return &client, inmem.OpPrefix, done
}

func TestCheckService(t *testing.T) {
	platformtesting.CheckService(initCheckService, t)
}
