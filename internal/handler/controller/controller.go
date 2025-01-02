package controller

import (
	"fmt"
	"io/ioutil"
	"encoding/json"
	"net/http"

	"github.com/go-order/internal/core"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"

	"github.com/go-order/internal/erro"
	"github.com/go-order/internal/lib"
	"github.com/go-order/internal/service"
)

var childLogger = log.With().Str("handler", "controller").Logger()

type HttpWorkerAdapter struct {
	workerService 	*service.WorkerService
}

func NewHttpWorkerAdapter(workerService *service.WorkerService) HttpWorkerAdapter {
	childLogger.Debug().Msg("NewHttpWorkerAdapter")

	return HttpWorkerAdapter{
		workerService: workerService,
	}
}

type APIError struct {
	StatusCode	int  `json:"statusCode"`
	Msg			string `json:"msg"`
}

func (e APIError) Error() string {
	return e.Msg
}

func NewAPIError(statusCode int, err error) APIError {
	return APIError{
		StatusCode: statusCode,
		Msg:		err.Error(),
	}
}

func WriteJSON(rw http.ResponseWriter, code int, v any) error{
	rw.WriteHeader(code)
	return json.NewEncoder(rw).Encode(v)
}

func (h *HttpWorkerAdapter) Health(rw http.ResponseWriter, req *http.Request) {
	childLogger.Debug().Msg("Health")

	health := true
	json.NewEncoder(rw).Encode(health)
}

func (h *HttpWorkerAdapter) Live(rw http.ResponseWriter, req *http.Request) {
	childLogger.Debug().Msg("Live")

	live := true
	json.NewEncoder(rw).Encode(live)
}

func (h *HttpWorkerAdapter) Header(rw http.ResponseWriter, req *http.Request) {
	childLogger.Debug().Msg("Header")
	
	json.NewEncoder(rw).Encode(req.Header)
}

func (h *HttpWorkerAdapter) Get(rw http.ResponseWriter, req *http.Request) error {
	childLogger.Debug().Msg("Get")

	span := lib.Span(req.Context(), "controller.Get")
	defer span.End()
	
	order := core.Order{}
	vars := mux.Vars(req)
	varID := vars["id"]

	order.OrderID = varID
	
	res, err := h.workerService.Get(req.Context(), &order)
	if err != nil {
		var apiError APIError
		switch err {
			case erro.ErrNotFound:
				apiError = NewAPIError(http.StatusNotFound, err)
			default:
				apiError = NewAPIError(http.StatusInternalServerError, err)
		}
		return apiError
	}

	return WriteJSON(rw, http.StatusOK, res)
}

func (h *HttpWorkerAdapter) Add( rw http.ResponseWriter, req *http.Request) error {
	childLogger.Debug().Msg("Add")

	span := lib.Span(req.Context(), "controller.Add")
	defer span.End()

	order := core.Order{}
	err := json.NewDecoder(req.Body).Decode(&order)
    if err != nil {
		apiError := NewAPIError(http.StatusBadRequest, erro.ErrUnmarshal)
		return apiError
    }
	defer req.Body.Close()

	res, err := h.workerService.Add(req.Context(), &order)
	if err != nil {
		var apiError APIError
		switch err {
		default:
			apiError = NewAPIError(http.StatusInternalServerError, err)
		}
		return apiError
	}

	return WriteJSON(rw, http.StatusOK, res)
}

func (h *HttpWorkerAdapter) AddAsync( rw http.ResponseWriter, req *http.Request) error {
	childLogger.Debug().Msg("AddAsync")

	span := lib.Span(req.Context(), "controller.AddAsync")
	defer span.End()

	order := core.Order{}
	err := json.NewDecoder(req.Body).Decode(&order)

    if err != nil {
		apiError := NewAPIError(http.StatusBadRequest, erro.ErrUnmarshal)
		return apiError
    }
	defer req.Body.Close()

	res, err := h.workerService.AddAsync(req.Context(), &order)
	if err != nil {
		var apiError APIError
		switch err {
		default:
			apiError = NewAPIError(http.StatusInternalServerError, err)
		}
		return apiError
	}

	return WriteJSON(rw, http.StatusOK, res)
}

func (h *HttpWorkerAdapter) UploadImage(rw http.ResponseWriter, req *http.Request) error {
	childLogger.Debug().Msg("UploadImage")

	span := lib.Span(req.Context(), "controller.UploadImage")
	defer span.End()

	var apiError APIError

	err := req.ParseMultipartForm(20 << 20) //20Mb
	if err != nil {
		apiError = NewAPIError(http.StatusBadRequest, err)
		return apiError
	}

	file, handler, err := req.FormFile("file")
	if err != nil {
		fmt.Print(err)
		switch err {
			case erro.ErrNotFound:
				apiError = NewAPIError(http.StatusNotFound, err)
			default:
				apiError = NewAPIError(http.StatusInternalServerError, err)
		}
		return apiError
	}
	defer file.Close()

	fmt.Printf("File name: %+v\n", handler.Filename)
	fmt.Printf("File size: %+v\n", handler.Size)
	fmt.Printf("File header: %+v\n", handler.Header)

	orderFile := core.OrderFile{}
	orderFile.Name = handler.Filename
	orderFile.File, err = ioutil.ReadAll(file)

	//fmt.Printf("++++ > orderFile: %+v\n", orderFile)

	res, err := h.workerService.UploadImage(req.Context(), &orderFile)
	if err != nil {
		var apiError APIError
		switch err {
		default:
			apiError = NewAPIError(http.StatusInternalServerError, err)
		}
		return apiError
	}

	return WriteJSON(rw, http.StatusOK, res)
}