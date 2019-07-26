package http

import (
	"context"
	"encoding/json"
	"fmt"
	http "net/http"
	"strconv"

	influxdb "github.com/influxdata/influxdb"
	"github.com/julienschmidt/httprouter"
	"go.uber.org/zap"
)

// CheckBackend are all services a checkhandler requires
type CheckBackend struct {
	influxdb.HTTPErrorHandler
	Logger *zap.Logger

	CheckService influxdb.CheckService
}

// NewCheckBackend returns a new checkbackend
func NewCheckBackend(b *APIBackend) *CheckBackend {
	return &CheckBackend{
		HTTPErrorHandler: b.HTTPErrorHandler,
		Logger:           b.Logger.With(zap.String("handler", "check")),
		CheckService:     b.CheckService,
	}
}

// CheckHandler responds to /api/v2/checks requests
type CheckHandler struct {
	*httprouter.Router
	influxdb.HTTPErrorHandler
	logger *zap.Logger

	CheckService influxdb.CheckService
}

const (
	checksPath   = "/api/v2/checks"
	checksIDPath = "/api/v2/checks/:id"
)

// NewCheckHandler returns a new checkhandler
func NewCheckHandler(b *CheckBackend) *CheckHandler {
	h := &CheckHandler{
		Router: NewRouter(b.HTTPErrorHandler),
		logger: zap.NewNop(),
	}

	h.HandlerFunc("GET", checksPath, h.handleGetChecks)
	h.HandlerFunc("POST", checksPath, h.handleCreateCheck)

	h.HandlerFunc("GET", checksIDPath, h.handleGetCheck)
	h.HandlerFunc("PATCH", checksIDPath, h.handleUpdateCheck)
	h.HandlerFunc("DELETE", checksIDPath, h.handleDeleteCheck)

	return h
}

// check is used internally for serialization/deserialization.
type check struct {
}

// toInfluxDB converts a check to its public HTTP response
func (c *check) toInfluxDB() (*influxdb.Check, error) {
	return &influxdb.Check{}, nil
}

// handleGetChecks is the HTTP handler for the GET /api/v2/checks route.
func (h *CheckHandler) handleGetChecks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.Debug("checks retrieve request", zap.String("r", fmt.Sprint(r)))
	req, err := decodeGetChecksRequest(ctx, r)
	if err != nil {
		err = &influxdb.Error{
			Err:  err,
			Code: influxdb.EInvalid,
			Msg:  "failed to decode request",
		}
		h.HandleHTTPError(ctx, err, w)
		return
	}

	checks, _, err := h.CheckService.FindChecks(ctx, req.filter)
	if err != nil {
		err = &influxdb.Error{
			Err: err,
			Msg: "failed to find checks",
		}
		h.HandleHTTPError(ctx, err, w)
		return
	}
	h.logger.Debug("checks retrieved", zap.String("checks", fmt.Sprint(checks)))
	if err := encodeResponse(ctx, w, http.StatusOK, newChecksResponse(ctx, checks, req.filter, h.LabelService)); err != nil {
		logEncodingError(h.logger, r, err)
		return
	}
}

type getChecksRequest struct {
	filter influxdb.CheckFilter
}

func decodeGetChecksRequest(ctx context.Context, r *http.Request) (*getChecksRequest, error) {
	qp := r.URL.Query()
	req := &getChecksRequest{}

	if limit := qp.Get("limit"); limit != "" {
		lim, err := strconv.Atoi(limit)
		if err != nil {
			return nil, err
		}
		if lim < 1 || lim > influxdb.CheckMaxPageSize {
			return nil, &influxdb.Error{
				Code: influxdb.EUnprocessableEntity,
				Msg:  fmt.Sprintf("limit must be between 1 and %d", influxdb.CheckMaxPageSize),
			}
		}
		req.filter.Limit = lim
	} else {
		req.filter.Limit = influxdb.CheckDefaultPageSize
	}

	return req, nil
}

// handleCreateCheck is the HTTP handler for the POST /api/v2/checks route.
func (h *CheckHandler) handleCreateCheck(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	h.logger.Debug("create check request", zap.String("r", fmt.Sprint(r)))
	req, err := decodePostCheckRequest(ctx, r)
	if err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}

	if err := h.CheckService.CreateCheck(ctx, req.Check); err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}
	h.logger.Debug("check created", zap.String("check", fmt.Sprint(req.Check)))

	if err := encodeResponse(ctx, w, http.StatusCreated, newCheckResponse(req.Check, []*influxdb.Label{})); err != nil {
		logEncodingError(h.logger, r, err)
		return
	}
}

type postCheckRequest struct {
	Check *influxdb.Check
}

func (b postCheckRequest) Validate() error {
	if !b.Check.OrganizationID.Valid() {
		return fmt.Errorf("check requires an organization")
	}
	return nil
}

func decodePostCheckRequest(ctx context.Context, r *http.Request) (*postCheckRequest, error) {
	c := &check{}
	if err := json.NewDecoder(r.Body).Decode(c); err != nil {
		return nil, err
	}

	ic, err := c.toInfluxDB()
	if err != nil {
		return nil, err
	}

	req := &postCheckRequest{
		Check: ic,
	}

	return req, req.Validate()
}

// handleGetCheck is the HTTP handler for the GET /api/v2/checks/:id route.
func (h *CheckHandler) handleGetCheck(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	h.logger.Debug("retrieve check request", zap.String("r", fmt.Sprint(r)))

	req, err := decodeGetCheckRequest(ctx, r)
	if err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}

	b, err := h.CheckService.FindCheckByID(ctx, req.CheckID)
	if err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}

	labels, err := h.LabelService.FindResourceLabels(ctx, influxdb.LabelMappingFilter{ResourceID: b.ID})
	if err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}

	h.logger.Debug("check retrieved", zap.String("check", fmt.Sprint(b)))

	if err := encodeResponse(ctx, w, http.StatusOK, newCheckResponse(b, labels)); err != nil {
		logEncodingError(h.logger, r, err)
		return
	}
}

type getCheckRequest struct {
	CheckID influxdb.ID
}

func decodeGetCheckRequest(ctx context.Context, r *http.Request) (*getCheckRequest, error) {
	params := httprouter.ParamsFromContext(ctx)
	id := params.ByName("id")
	if id == "" {
		return nil, &influxdb.Error{
			Code: influxdb.EInvalid,
			Msg:  "url missing id",
		}
	}

	var i influxdb.ID
	if err := i.DecodeFromString(id); err != nil {
		return nil, err
	}
	req := &getCheckRequest{
		CheckID: i,
	}

	return req, nil
}

// handleUpdateCheck is the HTTP handler for the PATCH /api/v2/checks/:id route.
func (h *CheckHandler) handleUpdateCheck(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.Debug("update check request", zap.String("r", fmt.Sprint(r)))

	req, err := decodePatchCheckRequest(ctx, r)
	if err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}

	b, err := h.CheckService.UpdateCheck(ctx, req.CheckID, req.Update)
	if err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}

	labels, err := h.LabelService.FindResourceLabels(ctx, influxdb.LabelMappingFilter{ResourceID: b.ID})
	if err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}
	h.logger.Debug("check updated", zap.String("check", fmt.Sprint(b)))

	if err := encodeResponse(ctx, w, http.StatusOK, newCheckResponse(b, labels)); err != nil {
		logEncodingError(h.logger, r, err)
		return
	}
}

type patchCheckRequest struct {
	Update  influxdb.CheckUpdate
	CheckID influxdb.ID
}

func decodePatchCheckRequest(ctx context.Context, r *http.Request) (*patchCheckRequest, error) {
	params := httprouter.ParamsFromContext(ctx)
	id := params.ByName("id")
	if id == "" {
		return nil, &influxdb.Error{
			Code: influxdb.EInvalid,
			Msg:  "url missing id",
		}
	}

	var i influxdb.ID
	if err := i.DecodeFromString(id); err != nil {
		return nil, &influxdb.Error{
			Code: influxdb.EInvalid,
			Msg:  err.Error(),
		}
	}

	cu := &influxdb.CheckUpdate{}
	if err := json.NewDecoder(r.Body).Decode(cu); err != nil {
		return nil, &influxdb.Error{
			Code: influxdb.EInvalid,
			Msg:  err.Error(),
		}
	}

	return &patchCheckRequest{
		Update:  *cu,
		CheckID: i,
	}, nil
}

// handleDeleteCheck is the HTTP handler for the DELETE /api/v2/checks/:id route.
func (h *CheckHandler) handleDeleteCheck(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.Debug("delete check request", zap.String("r", fmt.Sprint(r)))

	req, err := decodeDeleteCheckRequest(ctx, r)
	if err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}

	if err := h.CheckService.DeleteCheck(ctx, req.CheckID); err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}

	h.logger.Debug("check deleted", zap.String("checkID", req.CheckID.String()))

	w.WriteHeader(http.StatusNoContent)
}

type deleteCheckRequest struct {
	CheckID influxdb.ID
}

func decodeDeleteCheckRequest(ctx context.Context, r *http.Request) (*deleteCheckRequest, error) {
	params := httprouter.ParamsFromContext(ctx)
	id := params.ByName("id")
	if id == "" {
		return nil, &influxdb.Error{
			Code: influxdb.EInvalid,
			Msg:  "url missing id",
		}
	}

	var i influxdb.ID
	if err := i.DecodeFromString(id); err != nil {
		return nil, err
	}
	req := &deleteCheckRequest{
		CheckID: i,
	}

	return req, nil
}
