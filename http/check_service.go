package http

import (
	"context"
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

}

// handleGetCheck is the HTTP handler for the GET /api/v2/checks/:id route.
func (h *CheckHandler) handleGetCheck(w http.ResponseWriter, r *http.Request) {

}

// handleUpdateCheck is the HTTP handler for the PATCH /api/v2/checks/:id route.
func (h *CheckHandler) handleUpdateCheck(w http.ResponseWriter, r *http.Request) {

}

// handleDeleteCheck is the HTTP handler for the DELETE /api/v2/checks/:id route.
func (h *CheckHandler) handleDeleteCheck(w http.ResponseWriter, r *http.Request) {

}
