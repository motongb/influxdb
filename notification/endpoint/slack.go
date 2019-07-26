package endpoint

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/influxdata/influxdb"
)

var _ influxdb.NotificationEndpoint = &Slack{}

// Slack is the notification endpoint config of slack.
type Slack struct {
	Base
	// Path is the API path of Slack
	// example: https://slack.com/api/chat.postMessage
	URL string `json:"url"`
	// Token is the bearer token for authorization
	Token string `json:"token"`
}

// Valid returns error if some configuration is invalid
func (s Slack) Valid() error {
	if err := s.Base.valid(); err != nil {
		return err
	}
	if s.URL == "" {
		return &influxdb.Error{
			Code: influxdb.EInvalid,
			Msg:  "slack endpoint URL is empty",
		}
	}
	if _, err := url.Parse(s.URL); err != nil {
		return &influxdb.Error{
			Code: influxdb.EInvalid,
			Msg:  fmt.Sprintf("slack endpoint URL is invalid: %s", err.Error()),
		}
	}
	if s.Token == "" {
		return &influxdb.Error{
			Code: influxdb.EInvalid,
			Msg:  "slack endpoint token is empty",
		}
	}
	return nil
}

type slackAlias Slack

// MarshalJSON implement json.Marshaler interface.
func (s Slack) MarshalJSON() ([]byte, error) {
	return json.Marshal(
		struct {
			slackAlias
			Type string `json:"type"`
		}{
			slackAlias: slackAlias(s),
			Type:       s.Type(),
		})
}

// Type returns the
func (s Slack) Type() string {
	return "slack"
}

// ParseResponse will parse the http response from slack.
func (s Slack) ParseResponse(resp *http.Response) error {
	if resp.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		type response struct {
			Error string `json:"error"`
		}
		r := new(response)
		if err = json.Unmarshal(body, r); err != nil {
			r.Error = fmt.Sprintf("failed to understand Slack response. code: %d content: %s", resp.StatusCode, string(body))
		}

		return &influxdb.Error{
			Msg: r.Error,
		}
	}
	return nil
}
