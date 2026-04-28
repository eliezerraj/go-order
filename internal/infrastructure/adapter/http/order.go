package http

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/go-order/internal/domain/model"
	"github.com/go-order/shared/erro"

	"go.opentelemetry.io/otel/codes"
)

// About add order
func (h *HttpRouters) AddOrder(rw http.ResponseWriter, req *http.Request) error {
	ctx, cancel, span := h.withContext(req, "AddPayment")
	defer cancel()
	defer span.End()
	
	// decode payload
	var payload struct {
		model.Order
		OrderDate string `json:"order_date,omitempty"`
	}
	err := json.NewDecoder(req.Body).Decode(&payload)
	defer req.Body.Close()

	order := payload.Order
	if payload.OrderDate != "" {
		if parsedDate, errParse := time.Parse("2006-01-02", payload.OrderDate); errParse == nil {
			order.Date = parsedDate
		}
	} else {
		order.Date = time.Now()
	}

    if err != nil {
		span.RecordError(err) 
        span.SetStatus(codes.Error, err.Error())
		return h.ErrorHandler(h.getTraceID(ctx), erro.ErrBadRequest)
    }

	res, err := h.workerService.AddOrder(ctx, &order)
	if err != nil {
		return h.ErrorHandler(h.getTraceID(ctx), err)
	}
	
	return h.writeJSON(rw, http.StatusCreated, res)
}

// About get order service
func (h *HttpRouters) GetOrder(rw http.ResponseWriter, req *http.Request) error {
	ctx, cancel, span := h.withContext(req, "GetPayment")
	defer cancel()
	defer span.End()

	// decode payload			
	vars := mux.Vars(req)
	orderID, err := h.parseIDParam(vars)
    if err != nil {
		span.RecordError(err) 
        span.SetStatus(codes.Error, err.Error())
		return h.ErrorHandler(h.getTraceID(ctx), err)
    }

	order := model.Order{ID: orderID}

	res, err := h.workerService.GetOrder(ctx, &order)
	if err != nil {
		return h.ErrorHandler(h.getTraceID(ctx), err)
	}

	return h.writeJSON(rw, http.StatusOK, res)
}

// About add order
func (h *HttpRouters) Checkout(rw http.ResponseWriter, req *http.Request) error {
	ctx, cancel, span := h.withContext(req, "GetPayment")
	defer cancel()
	defer span.End()

	// decode payload
	order := model.Order{}
	err := json.NewDecoder(req.Body).Decode(&order)
	defer req.Body.Close()
    if err != nil {
		span.RecordError(err) 
        span.SetStatus(codes.Error, err.Error())
		return h.ErrorHandler(h.getTraceID(ctx), err)
    }

	res, err := h.workerService.Checkout(ctx, &order)
	if err != nil {
		return h.ErrorHandler(h.getTraceID(ctx), err)
	}
	
	return h.writeJSON(rw, http.StatusOK, res)
}
