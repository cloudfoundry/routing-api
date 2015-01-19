package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type RoutesHandler struct {
	maxTTL int
}

type RouteDeclaration struct {
	Route   string `json:"route"`
	Port    int    `json:"port"`
	IP      string `json:"ip"`
	TTL     int    `json:"ttl"`
	LogGuid string `json:"log_guid"`
}

func NewRoutesHandler(maxTTL int) *RoutesHandler {
	return &RoutesHandler{maxTTL: maxTTL}
}

func (h *RoutesHandler) Routes(w http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	var t RouteDeclaration
	err := decoder.Decode(&t)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if t.Route == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Request requires a route"))
		return
	}

	if t.Port <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Request requires a port greater than 0"))
		return
	}

	if t.IP == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Request requires a valid ip"))
		return
	}

	if t.TTL <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Request requires a ttl greater than 0"))
		return
	}

	if t.TTL > h.maxTTL {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("Max ttl is %d", h.maxTTL)))
		return
	}

	w.WriteHeader(http.StatusCreated)
}
