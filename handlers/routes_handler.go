package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/pivotal-golang/lager"
)

type RoutesHandler struct {
	maxTTL int
	logger lager.Logger
}

type RouteDeclaration struct {
	Route   string `json:"route"`
	Port    int    `json:"port"`
	IP      string `json:"ip"`
	TTL     int    `json:"ttl"`
	LogGuid string `json:"log_guid"`
}

func NewRoutesHandler(maxTTL int, logger lager.Logger) *RoutesHandler {
	return &RoutesHandler{
		maxTTL: maxTTL,
		logger: logger,
	}
}

func (h *RoutesHandler) Routes(w http.ResponseWriter, req *http.Request) {
	log := h.logger.Session("create-route")
	decoder := json.NewDecoder(req.Body)

	var t RouteDeclaration
	err := decoder.Decode(&t)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	log.Info("request", lager.Data{"route_declaration": t})

	var apiErr error

	if t.Route == "" {
		apiErr = errors.New("Request requires a route")
	}

	if t.Port <= 0 {
		apiErr = errors.New("Request requires a port greater than 0")
	}

	if t.IP == "" {
		apiErr = errors.New("Request requires a valid ip")
	}

	if t.TTL <= 0 {
		apiErr = errors.New("Request requires a ttl greater than 0")
	}

	if t.TTL > h.maxTTL {
		apiErr = errors.New(fmt.Sprintf("Max ttl is %d", h.maxTTL))
	}

	if apiErr != nil {
		log.Error("error", apiErr)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(apiErr.Error()))
		return
	}

	w.WriteHeader(http.StatusCreated)
}
