package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/cloudfoundry-incubator/routing-api/helpers"
	"github.com/cloudfoundry-incubator/routing-api/models"
	uaaclient "github.com/cloudfoundry-incubator/uaa-go-client"
	"github.com/pivotal-golang/lager"
)

type RouterGroupsHandler struct {
	uaaClient uaaclient.Client
	logger    lager.Logger
}

func NewRouteGroupsHandler(uaaClient uaaclient.Client, logger lager.Logger) *RouterGroupsHandler {
	return &RouterGroupsHandler{
		uaaClient: uaaClient,
		logger:    logger,
	}
}

func (h *RouterGroupsHandler) ListRouterGroups(w http.ResponseWriter, req *http.Request) {
	log := h.logger.Session("list-router-groups")
	log.Debug("started")
	defer log.Debug("completed")

	err := h.uaaClient.DecodeToken(req.Header.Get("Authorization"), RouterGroupsReadScope)
	if err != nil {
		handleUnauthorizedError(w, err, log)
		return
	}

	defaultRouterGroup := helpers.GetDefaultRouterGroup()

	jsonBytes, err := json.Marshal([]models.RouterGroup{defaultRouterGroup})
	if err != nil {
		log.Error("failed-to-marshal", err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
	w.Header().Set("Content-Length", strconv.Itoa(len(jsonBytes)))
}
