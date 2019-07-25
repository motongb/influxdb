package kv

import (
	"context"
	"encoding/json"

	"github.com/influxdata/influxdb"
	"github.com/influxdata/influxdb/kit/tracing"
)

var (
	checkCheck = []byte("checksv1")
	checkIndex = []byte("checkindexv1")
)

var _ influxdb.CheckService = (*Service)(nil)

func (s *Service) initializeChecks(ctx context.Context, tx Tx) error {
	if _, err := s.checksBucket(tx); err != nil {
		return err
	}
	if _, err := s.checksIndexBucket(tx); err != nil {
		return err
	}
	return nil
}

func (s *Service) checksBucket(tx Tx) (Bucket, error) {
	b, err := tx.Bucket(checkCheck)
	if err != nil {
		return nil, UnexpectedBucketError(err)
	}

	return b, nil
}

func (s *Service) checksIndexBucket(tx Tx) (Bucket, error) {
	b, err := tx.Bucket(checkIndex)
	if err != nil {
		return nil, UnexpectedBucketIndexError(err)
	}

	return b, nil
}

// FindCheckByID retrieves a check by id.
func (s *Service) FindCheckByID(ctx context.Context, id influxdb.ID) (*influxdb.Check, error) {
	span, ctx := tracing.StartSpanFromContext(ctx)
	defer span.Finish()

	var b *influxdb.Check
	var err error

	err = s.kv.View(ctx, func(tx Tx) error {
		bkt, pe := s.findCheckByID(ctx, tx, id)
		if pe != nil {
			err = pe
			return err
		}
		b = bkt
		return nil
	})

	if err != nil {
		return nil, err
	}

	return b, nil
}

func (s *Service) findCheckByID(ctx context.Context, tx Tx, id influxdb.ID) (*influxdb.Check, error) {
	span, _ := tracing.StartSpanFromContext(ctx)
	defer span.Finish()

	var b influxdb.Check

	encodedID, err := id.Encode()
	if err != nil {
		return nil, &influxdb.Error{
			Code: influxdb.EInvalid,
			Err:  err,
		}
	}

	bkt, err := s.checksBucket(tx)
	if err != nil {
		return nil, err
	}

	v, err := bkt.Get(encodedID)
	if IsNotFound(err) {
		return nil, &influxdb.Error{
			Code: influxdb.ENotFound,
			Msg:  "check not found",
		}
	}

	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(v, &b); err != nil {
		return nil, &influxdb.Error{
			Err: err,
		}
	}

	return &b, nil
}

// FindCheck retrives a check using an arbitrary check filter.
// Filters using ID, or OrganizationID and check Name should be efficient.
// Other filters will do a linear scan across checks until it finds a match.
func (s *Service) FindCheck(ctx context.Context, filter influxdb.CheckFilter) (*influxdb.Check, error) {
	span, ctx := tracing.StartSpanFromContext(ctx)
	defer span.Finish()

	var b *influxdb.Check
	var err error

	if filter.ID != nil {
		b, err = s.FindCheckByID(ctx, *filter.ID)
		if err != nil {
			return nil, &influxdb.Error{
				Err: err,
			}
		}
		return b, nil
	}

	err = s.kv.View(ctx, func(tx Tx) error {
		if filter.Org != nil {
			o, err := s.findOrganizationByName(ctx, tx, *filter.Org)
			if err != nil {
				return err
			}
			filter.OrganizationID = &o.ID
		}

		filterFn := filterChecksFn(filter)
		return s.forEachCheck(ctx, tx, false, func(bkt *influxdb.Check) bool {
			if filterFn(bkt) {
				b = bkt
				return false
			}
			return true
		})
	})

	if err != nil {
		return nil, &influxdb.Error{
			Err: err,
		}
	}

	if b == nil {
		return nil, &influxdb.Error{
			Code: influxdb.ENotFound,
			Msg:  "check not found",
		}
	}

	return b, nil
}

func filterChecksFn(filter influxdb.CheckFilter) func(b *influxdb.Check) bool {
	if filter.ID != nil {
		return func(b *influxdb.Check) bool {
			return b.ID == *filter.ID
		}
	}

	if filter.OrganizationID != nil {
		return func(b *influxdb.Check) bool {
			return b.OrganizationID == *filter.OrganizationID
		}
	}

	return func(b *influxdb.Check) bool { return true }
}

// FindChecks retrives all checks that match an arbitrary check filter.
// Filters using ID, or OrganizationID and check Name should be efficient.
// Other filters will do a linear scan across all checks searching for a match.
func (s *Service) FindChecks(ctx context.Context, filter influxdb.CheckFilter, opts ...influxdb.FindOptions) ([]*influxdb.Check, int, error) {
	span, ctx := tracing.StartSpanFromContext(ctx)
	defer span.Finish()

	if filter.ID != nil {
		b, err := s.FindCheckByID(ctx, *filter.ID)
		if err != nil {
			return nil, 0, err
		}

		return []*influxdb.Check{b}, 1, nil
	}

	bs := []*influxdb.Check{}
	err := s.kv.View(ctx, func(tx Tx) error {
		bkts, err := s.findChecks(ctx, tx, filter, opts...)
		if err != nil {
			return err
		}
		bs = bkts
		return nil
	})

	if err != nil {
		return nil, 0, err
	}

	return bs, len(bs), nil
}

func (s *Service) findChecks(ctx context.Context, tx Tx, filter influxdb.CheckFilter, opts ...influxdb.FindOptions) ([]*influxdb.Check, error) {
	span, ctx := tracing.StartSpanFromContext(ctx)
	defer span.Finish()

	bs := []*influxdb.Check{}
	if filter.Org != nil {
		o, err := s.findOrganizationByName(ctx, tx, *filter.Org)
		if err != nil {
			return nil, &influxdb.Error{
				Err: err,
			}
		}
		filter.OrganizationID = &o.ID
	}

	var offset, limit, count int
	var descending bool
	if len(opts) > 0 {
		offset = opts[0].Offset
		limit = opts[0].Limit
		descending = opts[0].Descending
	}

	filterFn := filterChecksFn(filter)
	err := s.forEachCheck(ctx, tx, descending, func(b *influxdb.Check) bool {
		if filterFn(b) {
			if count >= offset {
				bs = append(bs, b)
			}
			count++
		}

		if limit > 0 && len(bs) >= limit {
			return false
		}

		return true
	})

	if err != nil {
		return nil, &influxdb.Error{
			Err: err,
		}
	}

	return bs, nil
}

// CreateCheck creates a influxdb check and sets b.ID.
func (s *Service) CreateCheck(ctx context.Context, b *influxdb.Check) error {
	span, ctx := tracing.StartSpanFromContext(ctx)
	defer span.Finish()

	return s.kv.Update(ctx, func(tx Tx) error {
		return s.createCheck(ctx, tx, b)
	})
}

func (s *Service) createCheck(ctx context.Context, tx Tx, b *influxdb.Check) error {
	if b.OrganizationID.Valid() {
		span, ctx := tracing.StartSpanFromContext(ctx)
		defer span.Finish()

		_, pe := s.findOrganizationByID(ctx, tx, b.OrganizationID)
		if pe != nil {
			return &influxdb.Error{
				Err: pe,
			}
		}
	}

	b.ID = s.IDGenerator.ID()
	b.CreatedAt = s.Now()
	b.UpdatedAt = s.Now()

	if err := s.putCheck(ctx, tx, b); err != nil {
		return err
	}

	if err := s.createCheckUserResourceMappings(ctx, tx, b); err != nil {
		return err
	}
	return nil
}

// PutCheck will put a check without setting an ID.
func (s *Service) PutCheck(ctx context.Context, b *influxdb.Check) error {
	return s.kv.Update(ctx, func(tx Tx) error {
		var err error
		pe := s.putCheck(ctx, tx, b)
		if pe != nil {
			err = pe
		}
		return err
	})
}

func (s *Service) createCheckUserResourceMappings(ctx context.Context, tx Tx, b *influxdb.Check) error {
	span, ctx := tracing.StartSpanFromContext(ctx)
	defer span.Finish()

	ms, err := s.findUserResourceMappings(ctx, tx, influxdb.UserResourceMappingFilter{
		ResourceType: influxdb.OrgsResourceType,
		ResourceID:   b.OrganizationID,
	})
	if err != nil {
		return &influxdb.Error{
			Err: err,
		}
	}

	for _, m := range ms {
		if err := s.createUserResourceMapping(ctx, tx, &influxdb.UserResourceMapping{
			ResourceType: influxdb.ChecksResourceType,
			ResourceID:   b.ID,
			UserID:       m.UserID,
			UserType:     m.UserType,
		}); err != nil {
			return &influxdb.Error{
				Err: err,
			}
		}
	}

	return nil
}

func (s *Service) putCheck(ctx context.Context, tx Tx, b *influxdb.Check) error {
	span, _ := tracing.StartSpanFromContext(ctx)
	defer span.Finish()

	v, err := json.Marshal(b)
	if err != nil {
		return &influxdb.Error{
			Err: err,
		}
	}

	encodedID, err := b.ID.Encode()
	if err != nil {
		return &influxdb.Error{
			Err: err,
		}
	}
	key, pe := checkIndexKey(b)
	if err != nil {
		return pe
	}

	idx, err := s.checksIndexBucket(tx)
	if err != nil {
		return err
	}

	if err := idx.Put(key, encodedID); err != nil {
		return &influxdb.Error{
			Err: err,
		}
	}

	bkt, err := s.checksBucket(tx)
	if bkt.Put(encodedID, v); err != nil {
		return &influxdb.Error{
			Err: err,
		}
	}
	return nil
}

// checkIndexKey is a combination of the orgID and the check name.
func checkIndexKey(b *influxdb.Check) ([]byte, error) {
	orgID, err := b.OrganizationID.Encode()
	if err != nil {
		return nil, &influxdb.Error{
			Code: influxdb.EInvalid,
			Err:  err,
		}
	}
	k := make([]byte, influxdb.IDLength+len(b.Name))
	copy(k, orgID)
	copy(k[influxdb.IDLength:], []byte(b.Name))
	return k, nil
}

// forEachCheck will iterate through all checks while fn returns true.
func (s *Service) forEachCheck(ctx context.Context, tx Tx, descending bool, fn func(*influxdb.Check) bool) error {
	span, _ := tracing.StartSpanFromContext(ctx)
	defer span.Finish()

	bkt, err := s.checksBucket(tx)
	if err != nil {
		return err
	}

	cur, err := bkt.Cursor()
	if err != nil {
		return err
	}

	var k, v []byte
	if descending {
		k, v = cur.Last()
	} else {
		k, v = cur.First()
	}

	for k != nil {
		b := &influxdb.Check{}
		if err := json.Unmarshal(v, b); err != nil {
			return err
		}
		if !fn(b) {
			break
		}

		if descending {
			k, v = cur.Prev()
		} else {
			k, v = cur.Next()
		}
	}

	return nil
}

// UpdateCheck updates a check according the parameters set on upd.
func (s *Service) UpdateCheck(ctx context.Context, id influxdb.ID, upd influxdb.CheckUpdate) (*influxdb.Check, error) {
	span, ctx := tracing.StartSpanFromContext(ctx)
	defer span.Finish()

	var b *influxdb.Check
	err := s.kv.Update(ctx, func(tx Tx) error {
		bkt, err := s.updateCheck(ctx, tx, id, upd)
		if err != nil {
			return err
		}
		b = bkt
		return nil
	})

	return b, err
}

func (s *Service) updateCheck(ctx context.Context, tx Tx, id influxdb.ID, upd influxdb.CheckUpdate) (*influxdb.Check, error) {
	span, ctx := tracing.StartSpanFromContext(ctx)
	defer span.Finish()

	b, err := s.findCheckByID(ctx, tx, id)
	if err != nil {
		return nil, err
	}

	// if upd.RetentionPeriod != nil {
	// 	b.RetentionPeriod = *upd.RetentionPeriod
	// }

	// TODO(jm): fix this
	// if upd.Description != nil {
	// 	b.Description = *upd.Description
	// }

	// if upd.Name != nil {
	// 	b0, err := s.findCheckByName(ctx, tx, b.OrganizationID, *upd.Name)
	// 	if err == nil && b0.ID != id {
	// 		return nil, &influxdb.Error{
	// 			Code: influxdb.EConflict,
	// 			Msg:  "check name is not unique",
	// 		}
	// 	}
	// 	key, err := checkIndexKey(b)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	idx, err := s.checksIndexBucket(tx)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	// Checks are indexed by name and so the check index must be pruned when name is modified.
	// 	if err := idx.Delete(key); err != nil {
	// 		return nil, err
	// 	}
	// 	b.Name = *upd.Name
	// }

	b.UpdatedAt = s.Now()

	if err := s.putCheck(ctx, tx, b); err != nil {
		return nil, err
	}

	return b, nil
}

// DeleteCheck deletes a check and prunes it from the index.
func (s *Service) DeleteCheck(ctx context.Context, id influxdb.ID) error {
	return s.kv.Update(ctx, func(tx Tx) error {
		var err error
		if pe := s.deleteCheck(ctx, tx, id); pe != nil {
			err = pe
		}
		return err
	})
}

func (s *Service) deleteCheck(ctx context.Context, tx Tx, id influxdb.ID) error {
	b, pe := s.findCheckByID(ctx, tx, id)
	if pe != nil {
		return pe
	}

	key, pe := checkIndexKey(b)
	if pe != nil {
		return pe
	}

	idx, err := s.checksIndexBucket(tx)
	if err != nil {
		return err
	}

	if err := idx.Delete(key); err != nil {
		return &influxdb.Error{
			Err: err,
		}
	}

	encodedID, err := id.Encode()
	if err != nil {
		return &influxdb.Error{
			Code: influxdb.EInvalid,
			Err:  err,
		}
	}

	bkt, err := s.checksBucket(tx)
	if err != nil {
		return err
	}

	if err := bkt.Delete(encodedID); err != nil {
		return &influxdb.Error{
			Err: err,
		}
	}

	if err := s.deleteUserResourceMappings(ctx, tx, influxdb.UserResourceMappingFilter{
		ResourceID:   id,
		ResourceType: influxdb.ChecksResourceType,
	}); err != nil {
		return err
	}

	return nil
}
