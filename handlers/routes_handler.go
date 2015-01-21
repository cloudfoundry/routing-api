package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/pivotal-cf-experimental/routing-api/db"
	"github.com/pivotal-golang/lager"
)

type RoutesHandler struct {
	maxTTL    int
	validator RouteValidator
	db        db.DB
	logger    lager.Logger
}

func NewRoutesHandler(maxTTL int, validator RouteValidator, database db.DB, logger lager.Logger) *RoutesHandler {
	return &RoutesHandler{
		maxTTL:    maxTTL,
		validator: validator,
		db:        database,
		logger:    logger,
	}
}

func (h *RoutesHandler) Routes(w http.ResponseWriter, req *http.Request) {
	log := h.logger.Session("create-route")
	decoder := json.NewDecoder(req.Body)

	var routes []db.Route
	err := decoder.Decode(&routes)
	if err != nil {
		handleProcessRequestError(w, err, log)
		return
	}

	log.Info("request", lager.Data{"route_creation": routes})

	apiErr := h.validator.ValidateCreate(routes, h.maxTTL)
	if apiErr != nil {
		handleApiError(w, apiErr, log)
		return
	}

	for _, route := range routes {
		err = h.db.SaveRoute(route)
		if err != nil {
			handleDBError(w, err, log)
			return
		}
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *RoutesHandler) Delete(w http.ResponseWriter, req *http.Request) {
	log := h.logger.Session("delete-route")
	decoder := json.NewDecoder(req.Body)

	var routes []db.Route
	err := decoder.Decode(&routes)
	if err != nil {
		handleProcessRequestError(w, err, log)
		return
	}

	log.Info("request", lager.Data{"route_deletion": routes})

	apiErr := h.validator.ValidateDelete(routes)
	if apiErr != nil {
		handleApiError(w, apiErr, log)
		return
	}

	for _, route := range routes {
		err = h.db.DeleteRoute(route)
		if err != nil {
			handleDBError(w, err, log)
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

func handleProcessRequestError(w http.ResponseWriter, procErr error, log lager.Logger) {
	log.Error("error", procErr)

	retErr, _ := json.Marshal(Error{ProcessRequestError, "Cannot process request"})

	w.WriteHeader(http.StatusBadRequest)
	w.Write(retErr)
}

func handleApiError(w http.ResponseWriter, apiErr *Error, log lager.Logger) {
	log.Error("error", apiErr)

	retErr, _ := json.Marshal(apiErr)

	w.WriteHeader(http.StatusBadRequest)
	w.Write(retErr)
}

func handleDBError(w http.ResponseWriter, err error, log lager.Logger) {
	log.Error("error", err)

	retErr, _ := json.Marshal(Error{DBCommunicationError, err.Error()})

	w.WriteHeader(http.StatusInternalServerError)
	w.Write(retErr)
}
