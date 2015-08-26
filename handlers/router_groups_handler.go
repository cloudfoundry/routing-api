package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/cloudfoundry-incubator/routing-api/db"
	"github.com/cloudfoundry-incubator/routing-api/helpers"
	"github.com/pivotal-golang/lager"
)

const (
	RouterGroupsReadScope = "router_groups.read"
)

type RouterGroupsHandler struct {
	logger lager.Logger
}

func NewRouteGroupsHandler(logger lager.Logger) *RouterGroupsHandler {
	return &RouterGroupsHandler{
		logger: logger,
	}
}

func (h *RouterGroupsHandler) ListRouterGroups(w http.ResponseWriter, req *http.Request) {
	log := h.logger.Session("list-router-groups")
	log.Debug("started")
	defer log.Debug("completed")

	defaultRouterGroup := helpers.GetDefaultRouterGroup()

	jsonBytes, err := json.Marshal([]db.RouterGroup{defaultRouterGroup})
	if err != nil {
		log.Error("failed-to-marshal", err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
	w.Header().Set("Content-Length", strconv.Itoa(len(jsonBytes)))
}
