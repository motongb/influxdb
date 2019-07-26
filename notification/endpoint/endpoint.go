package endpoint

import "github.com/influxdata/influxdb"

// Base is the embed struct of every notification endpoint.
type Base struct {
	ID          influxdb.ID     `json:"id,omitempty"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	EndpointID  *influxdb.ID    `json:"endpointID,omitempty"`
	OrgID       influxdb.ID     `json:"orgID,omitempty"`
	OwnerID     influxdb.ID     `json:"userID,omitempty"`
	Status      influxdb.Status `json:"status"`
	influxdb.CRUDLog
}

func (b Base) valid() error {
	if !b.ID.Valid() {
		return &influxdb.Error{
			Code: influxdb.EInvalid,
			Msg:  "Notification Rule ID is invalid",
		}
	}
	if b.Name == "" {
		return &influxdb.Error{
			Code: influxdb.EInvalid,
			Msg:  "Notification Rule Name can't be empty",
		}
	}
	if b.EndpointID != nil && !b.EndpointID.Valid() {
		return &influxdb.Error{
			Code: influxdb.EInvalid,
			Msg:  "Notification Rule EndpointID is invalid",
		}
	}
	if b.Status != influxdb.Active && b.Status != influxdb.Inactive {
		return &influxdb.Error{
			Code: influxdb.EInvalid,
			Msg:  "invalid status",
		}
	}
	return nil
}

// GetID implements influxdb.Getter interface.
func (b Base) GetID() influxdb.ID {
	return b.ID
}

// GetOrgID implements influxdb.Getter interface.
func (b Base) GetOrgID() influxdb.ID {
	return b.OrgID
}

// GetCRUDLog implements influxdb.Getter interface.
func (b Base) GetCRUDLog() influxdb.CRUDLog {
	return b.CRUDLog
}

// SetID will set the primary key.
func (b *Base) SetID(id influxdb.ID) {
	b.ID = id
}

// SetOrgID will set the org key.
func (b *Base) SetOrgID(id influxdb.ID) {
	b.OrgID = id
}
