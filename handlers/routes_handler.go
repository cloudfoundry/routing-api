package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pivotal-cf-experimental/routing-api/db"
	"github.com/pivotal-golang/lager"
)

type RoutesHandler struct {
	maxTTL int
	db     db.DB
	logger lager.Logger
}

func NewRoutesHandler(maxTTL int, database db.DB, logger lager.Logger) *RoutesHandler {
	return &RoutesHandler{
		maxTTL: maxTTL,
		db:     database,
		logger: logger,
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

	for _, route := range routes {
		log.Info("request", lager.Data{"route_declaration": route})
		apiErr := routeValidator(route)

		if route.TTL <= 0 {
			apiErr = &Error{RouteInvalidError, "Request requires a ttl greater than 0"}
		}

		if route.TTL > h.maxTTL {
			apiErr = &Error{RouteInvalidError, fmt.Sprintf("Max ttl is %d", h.maxTTL)}
		}

		if apiErr != nil {
			handleApiError(w, apiErr, log)
			return
		}
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

	for _, route := range routes {
		log.Info("request", lager.Data{"route_deletion": route})

		apiErr := routeValidator(route)

		if apiErr != nil {
			handleApiError(w, apiErr, log)
			return
		}
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

func routeValidator(route db.Route) *Error {
	var apiErr *Error

	if route.Route == "" {
		apiErr = &Error{RouteInvalidError, "Request requires a route"}
	}

	if route.Port <= 0 {
		apiErr = &Error{RouteInvalidError, "Request requires a port greater than 0"}
	}

	if route.IP == "" {
		apiErr = &Error{RouteInvalidError, "Request requires a valid ip"}
	}

	return apiErr
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
