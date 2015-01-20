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

	var route db.Route
	err := decoder.Decode(&route)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	log.Info("request", lager.Data{"route_declaration": route})

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

	if route.TTL <= 0 {
		apiErr = &Error{RouteInvalidError, "Request requires a ttl greater than 0"}
	}

	if route.TTL > h.maxTTL {
		apiErr = &Error{RouteInvalidError, fmt.Sprintf("Max ttl is %d", h.maxTTL)}
	}

	if apiErr != nil {
		log.Error("error", apiErr)
		w.WriteHeader(http.StatusBadRequest)

		retErr, _ := json.Marshal(apiErr)

		w.Write(retErr)
		return
	}

	err = h.db.SaveRoute(route)
	if err != nil {
		log.Error("error", err)

		retErr, _ := json.Marshal(Error{DBCommunicationError, err.Error()})

		w.WriteHeader(http.StatusInternalServerError)
		w.Write(retErr)
		return
	}

	w.WriteHeader(http.StatusCreated)
}
