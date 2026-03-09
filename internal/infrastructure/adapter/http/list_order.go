package http

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/go-order/internal/domain/model"
)

// About list order
func (h *HttpRouters) ListOrder(rw http.ResponseWriter, req *http.Request) error {
	
	ctx, cancel, span := h.withContext(req, "ListOrder")
	defer cancel()
	defer span.End()

	// decode payload			
	vars := mux.Vars(req)
	user := vars["id"]

	order := model.Order{User: user}

	res, err := h.workerService.ListOrder(ctx, &order)
	if err != nil {
		return h.ErrorHandler(h.getTraceID(ctx), err)
	}
	
	return h.writeJSON(rw, http.StatusOK, res)
}
